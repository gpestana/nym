// monitor.go - Blockchain monitor.
// Copyright (C) 2019  Jedrzej Stuczynski.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// Package monitor implements the support for monitoring the state of the Tendermint Blockchain
// (later Ethereum I guess?) to sign all committed requests.
package monitor

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/nymtech/nym/logger"
	"github.com/nymtech/nym/server/storage"
	tmclient "github.com/nymtech/nym/tendermint/client"
	"github.com/nymtech/nym/worker"
	atypes "github.com/tendermint/tendermint/abci/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
	"gopkg.in/op/go-logging.v1"
)

const (
	maxInterval = time.Second * 30

	// todo: figure out if we want query per tx or per block

	// txs for actual data, block header to know how many should have arrived
	// (needed if node died after sending only part of them)
	headersQuery = "tm.event = 'NewBlockHeader'"
	txsQuery     = "tm.event = 'Tx'"
)

// Monitor represents the Blockchain monitor
type Monitor struct {
	sync.Mutex
	worker.Worker
	store                      *storage.Database
	tmClient                   *tmclient.Client
	subscriberStr              string
	txsEventsCh                <-chan ctypes.ResultEvent
	headersEventsCh            <-chan ctypes.ResultEvent
	latestConsecutiveProcessed int64              // everything up to that point (including it) is already stored on disk
	processedBlocks            map[int64]struct{} // think of it as a set rather than a hashmap
	unprocessedBlocks          map[int64]*block

	log *logging.Logger
}

type block struct {
	sync.Mutex
	creationTime   time.Time // approximate creation time of the given struct, NOT the actual block on the chain
	height         int64
	NumTxs         int64
	receivedHeader bool
	beingProcessed bool

	Txs []*tx
}

func (b *block) isFull() bool {
	b.Lock()
	defer b.Unlock()

	if int64(len(b.Txs)) != b.NumTxs {
		return false
	}

	for i := range b.Txs {
		if b.Txs[i] == nil {
			return false
		}
	}
	return true
}

func (b *block) addTx(newTx *tx) {
	b.Lock()
	defer b.Unlock()

	if len(b.Txs) < int(newTx.index)+1 {
		newTxs := make([]*tx, newTx.index+1)
		for _, oldTx := range b.Txs {
			if oldTx != nil {
				newTxs[oldTx.index] = oldTx
			}
		}
		b.Txs = newTxs
	}
	b.Txs[newTx.index] = newTx
}

func startNewBlock(header types.Header) *block {
	return &block{
		creationTime:   time.Now(),
		height:         header.Height,
		NumTxs:         header.NumTxs,
		receivedHeader: true,
		Txs:            make([]*tx, int(header.NumTxs)),
	}
}

type tx struct {
	height int64
	index  uint32
	Code   uint32
	Events []atypes.Event
	isNil  bool
}

func startNewTx(txData types.EventDataTx) *tx {
	return &tx{
		height: txData.Height,
		index:  txData.Index,
		Code:   txData.Result.Code,
		Events: txData.Result.Events,
	}
}

// FinalizeHeight gets called when all txs from a particular block are processed.
func (m *Monitor) FinalizeHeight(height int64) {
	m.log.Debugf("Finalizing height %v", height)
	m.log.Debugf("Unprocessed: \n%v\n\n\nProcessed: \n%v", m.unprocessedBlocks, m.processedBlocks)
	m.Lock()
	defer m.Unlock()
	if height == m.latestConsecutiveProcessed+1 {
		m.latestConsecutiveProcessed = height
		for i := height + 1; ; i++ {
			if _, ok := m.processedBlocks[i]; ok {
				m.log.Debugf("Also finalizing %v", i)
				m.latestConsecutiveProcessed = i
				delete(m.processedBlocks, i)
			} else {
				m.log.Debugf("%v is not in processed blocks", i)
				break
			}
		}
		m.store.FinalizeHeight(m.latestConsecutiveProcessed)
	} else {
		m.processedBlocks[height] = struct{}{}
	}
	delete(m.unprocessedBlocks, height)
}

func (m *Monitor) forceUpdateBlock(height int64) {
	m.Lock()

	if m.latestConsecutiveProcessed >= height {
		m.log.Debugf("Another goroutine already added forced %v. Commited.", height)
		return
	}
	if _, ok := m.processedBlocks[height]; ok {
		m.log.Debugf("Another goroutine already added forced %v. Processed.", height)
		return
	}
	if b, ok := m.unprocessedBlocks[height]; ok && b.creationTime.Add(maxInterval).After(time.Now()) {
		m.log.Debugf("Another goroutine already added forced %v. Unprocessed.", height)
		return
	}
	m.Unlock()

	m.log.Debugf("Force update height: %v", height)
	res, err := m.tmClient.BlockResults(&height)
	if err != nil {
		m.log.Errorf("Could not obtain results for height: %v", height)
		return
	}
	m.addNewCatchUpBlock(res, true)
}

// GetLowestFullUnprocessedBlock returns block on lowest height that is currently not being processed.
// FIXME: it doesn't actually return the lowest, but does it matter?
//nolint: golint
func (m *Monitor) GetLowestFullUnprocessedBlock() (int64, *block) {
	m.Lock()
	defer m.Unlock()
	for k, v := range m.unprocessedBlocks {
		if v.isFull() && !v.beingProcessed { // allows for multiple processors
			return k, v
		}
		if !v.isFull() && v.creationTime.Add(maxInterval).Before(time.Now()) {
			// we've had this block in the queue for a while and didn't get all txs, so let's query for its entirety
			go m.forceUpdateBlock(k)
		}
		// m.log.Errorf("Nope %v, full: %v, processed: %v", k, v.isFull(), v.beingProcessed)
	}
	return -1, nil
}

// TODO: the method is currently being unused, do we really need it?
func (m *Monitor) lowestUnprocessedBlockHeight() int64 {
	m.Lock()
	defer m.Unlock()
	if len(m.unprocessedBlocks) == 0 {
		return -1
	}
	var lowestHeight int64 = math.MaxInt64
	for k := range m.unprocessedBlocks {
		if k < lowestHeight {
			lowestHeight = k
		}
	}
	return lowestHeight
}

func (m *Monitor) addNewBlock(b *block) {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.unprocessedBlocks[b.height]; !ok {
		m.unprocessedBlocks[b.height] = b
		return
	}

	m.log.Debugf("Block at height: %v already present", b.height)
	if m.unprocessedBlocks[b.height].receivedHeader {
		// that's really an undefined behaviour. we probably received the same header twice?
		// ignore for now
	} else {
		oldTxs := m.unprocessedBlocks[b.height].Txs
		for _, oldTx := range oldTxs {
			if oldTx != nil {
				b.Txs[oldTx.index] = oldTx
			}
		}
		m.unprocessedBlocks[b.height] = b
	}
}

func (m *Monitor) addNewTx(newTx *tx) {
	m.Lock()
	defer m.Unlock()
	b, ok := m.unprocessedBlocks[newTx.height]
	if !ok {
		// we haven't received block header  and this is the first tx we received for that block
		tempBlock := &block{
			creationTime:   time.Now(),
			height:         newTx.height,
			NumTxs:         -1,
			receivedHeader: false,
			Txs:            make([]*tx, int(newTx.index)+1), // we know that there are at least that many txs in the block
		}
		tempBlock.Txs[newTx.index] = newTx
		m.unprocessedBlocks[newTx.height] = tempBlock
		return
	}
	b.addTx(newTx)
}

func (m *Monitor) addNewCatchUpBlock(res *ctypes.ResultBlockResults, overwrite bool) {
	m.Lock()
	defer m.Unlock()

	m.log.Debugf("Catching up on block %v", res.Height)

	// ensure it's not in blocks to be processed or that are processed

	// we don't care about the current status of this height, whether it's processed or not
	if overwrite {
		m.log.Debugf("Overwriting block at height %v", res.Height)
		// according to godocs, if map is nil or element doesn't exist, delete is a no-op so this is fine
		delete(m.unprocessedBlocks, res.Height)
		delete(m.processedBlocks, res.Height)
	}

	if _, ok := m.unprocessedBlocks[res.Height]; !ok && res.Height > m.latestConsecutiveProcessed {
		if _, ok := m.processedBlocks[res.Height]; !ok {

			b := &block{
				creationTime:   time.Now(),
				height:         res.Height,
				NumTxs:         int64(len(res.Results.DeliverTx)),
				receivedHeader: true,
				Txs:            make([]*tx, len(res.Results.DeliverTx)),
			}

			for i, resTx := range res.Results.DeliverTx {
				if resTx == nil {
					b.Txs[i] = &tx{
						isNil: true,
					}
					continue
				}
				b.Txs[i] = &tx{
					height: res.Height,
					index:  uint32(i),
					Code:   resTx.Code,
					Events: resTx.Events,
				}
			}
			m.unprocessedBlocks[res.Height] = b
		}
	}
}

// gets blockchain data from startHeight to endHeight (both inclusive)
func (m *Monitor) catchUp(startHeight, endHeight int64) {
	m.log.Debugf("Catching up from %v to %v", startHeight, endHeight)
	// according to docs, blockchaininfo can return at most 20 items
	if endHeight-startHeight >= 20 {
		m.log.Debug("There are more than 20 blocks to catchup on")
		m.catchUp(startHeight, startHeight+19)
		m.catchUp(startHeight+20, endHeight)
	}

	res, err := m.tmClient.BlockchainInfo(startHeight, endHeight)
	if err != nil {
		// TODO:
		// how should we behave on error, panic, return, etc?
		m.log.Critical("Error on catchup")
	}

	for _, blockMeta := range res.BlockMetas {
		header := blockMeta.Header
		if header.NumTxs == 0 {
			// then we can just add the block and forget about it
			m.addNewBlock(startNewBlock(header))
		} else {
			// otherwise we need to get tx details
			// TODO: parallelize it perhaps?
			blockRes, err := m.tmClient.BlockResults(&header.Height)
			if err != nil {
				// TODO:
				// same issue, how to behave?; panic, return, etc?
				m.log.Critical("Error on catchup")
			}
			m.addNewCatchUpBlock(blockRes, false)
		}
	}
}

func (m *Monitor) resyncWithBlockchain() error {
	latestStored := m.store.GetHighest()
	latestBlock, err := m.tmClient.BlockResults(nil)
	if err != nil {
		return err
	}
	m.log.Debugf("Resyncing blocks with the chain; latestStored: %v, latestBlock: %v", latestStored, latestBlock.Height)

	if latestStored < (latestBlock.Height - 1) {
		m.log.Warningf("Monitor is behind the blockchain. Latest stored height: %v, latest block height: %v",
			latestStored,
			latestBlock.Height,
		)
		m.addNewCatchUpBlock(latestBlock, false)
		m.catchUp(latestStored+1, latestBlock.Height-1)
	} else {
		m.log.Debug("Monitor is up to date with the blockchain")
	}
	return nil
}

func (m *Monitor) resubscribeToBlockchain() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	headersEventsCh, err := m.tmClient.Subscribe(ctx, m.subscriberStr, headersQuery)
	if err != nil {
		return err
	}
	m.log.Debug("Resubscribed to new headers")

	txsEventsCh, err := m.tmClient.Subscribe(ctx, m.subscriberStr, txsQuery)
	if err != nil {
		return err
	}
	m.log.Debug("Resubscribed to new txs")

	m.headersEventsCh = headersEventsCh
	m.txsEventsCh = txsEventsCh

	return nil
}

func (m *Monitor) resubscribeToBlockchainFull() error {
	m.log.Debug("Resubscribing to the blockchain")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := m.tmClient.UnsubscribeAll(ctx, m.subscriberStr); err != nil {
		m.log.Noticef("%v", err)
	}

	if err := m.resubscribeToBlockchain(); err != nil {
		err := m.tmClient.ForceReconnect()
		if err != nil {
			return err
		}
		// after reconnecting to new node we try to recreate the subscriptions again
		return m.resubscribeToBlockchain()
	}
	return nil
}

// we only care about processed blocks.
func (m *Monitor) fillBlockGaps() {
	m.log.Debug("Filling missing blocks")
	m.Lock()
	if len(m.processedBlocks) == 0 {
		m.Unlock()
		return
	}

	remainingBlocks := len(m.processedBlocks)
	gaps := make([]int64, 0)
	for h := m.latestConsecutiveProcessed + 1; ; h++ {
		if remainingBlocks == 0 {
			// no point going past the last processed block
			break
		}

		if _, ok := m.processedBlocks[h]; ok {
			remainingBlocks--
		} else if _, ok := m.unprocessedBlocks[h]; !ok {
			m.log.Debugf("Found gap at height: %v remaining: %v", h, remainingBlocks)
			// if it's not in processed nor unprocessed blocks, it means we never got the data hence it's a gap
			gaps = append(gaps, h)
		}
	}

	// give up the lock since we don't need it anymore and other goroutines could do their work, like block processing
	m.Unlock()

	for _, gap := range gaps {
		if gap <= 0 {
			m.log.Errorf("Gap block with invalid height: %v", gap)
		}
		m.log.Debugf("Going to fill in the gap at height: %v", gap)
		m.forceUpdateBlock(gap)
	}
}

// for now assume we receive all subscription events and nodes never go down
func (m *Monitor) worker() {
	// TODO: goroutine monitoring processed/unprocessed maps and forcing update of blanks;
	// alternatively force resync?

	timeoutTicker := time.NewTicker(maxInterval)
	missingBlocksTicker := time.NewTicker(maxInterval)
	for {
		select {
		case e := <-m.headersEventsCh:
			headerData := e.Data.(types.EventDataNewBlockHeader).Header

			m.log.Debugf("Received header for height : %v", headerData.Height)

			// TODO: update based on new case
			m.addNewBlock(startNewBlock(headerData))
			// reset ticker on each successful read
			timeoutTicker = time.NewTicker(maxInterval)

		case e := <-m.txsEventsCh:
			txData := e.Data.(types.EventDataTx)

			m.log.Debugf("Received tx %v height: %v", txData.Index, txData.Height)

			// TODO: update based on new case
			m.addNewTx(startNewTx(txData))
			// reset ticker on each successful read
			timeoutTicker = time.NewTicker(maxInterval)

		case <-missingBlocksTicker.C:
			// look for gaps in our processed/unprocessed blocks and query for them individually
			go func() {
				m.fillBlockGaps()
				// reset the timer when func returns
				missingBlocksTicker = time.NewTicker(maxInterval)
			}()

		case <-timeoutTicker.C:
			// on target environment we assume regular-ish block intervals with empty blocks if needed.
			// if we dont hear anything back, we assume a failure.
			m.log.Warningf("Timeout - Didn't receive any transactions from Tendermint chain in %v seconds", maxInterval)
			m.log.Debugf("%v blocks to be processed", len(m.unprocessedBlocks))

			if err := m.resubscribeToBlockchainFull(); err != nil {
				// what to do now?
				m.log.Critical(fmt.Sprintf("Couldn't resubscribe to the blockchain: %v", err))
				return
			}
			if err := m.resyncWithBlockchain(); err != nil {
				// again, what to do now? But at least we're connected so we could theoretically receive some data?
				m.log.Errorf("Couldn't resync with the blockchain: %v", err)
			}

			// for now do a dummy catchup as in on everything after last stored block.
			// later improve and do it selectively

		case <-m.HaltCh():
			return
		}
	}
}

// Halt stops the monitor and unsubscribes from any open queries.
func (m *Monitor) Halt() {
	m.log.Debugf("Halting the monitor")
	m.Worker.Halt()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := m.tmClient.UnsubscribeAll(ctx, m.subscriberStr); err != nil {
		m.log.Noticef("%v", err)
	}
}

// New creates a new monitor.
func New(l *logger.Logger, tmClient *tmclient.Client, store *storage.Database, id string) (*Monitor, error) {
	// read db with current state etc
	subscriberStr := fmt.Sprintf("monitor%v", id)
	log := l.GetLogger("Monitor")

	// in case we didn't shutdown cleanly last time
	if err := tmClient.UnsubscribeAll(context.Background(), subscriberStr); err != nil {
		log.Noticef("%v", err)
	}

	monitor := &Monitor{
		tmClient:                   tmClient,
		store:                      store,
		log:                        log,
		subscriberStr:              subscriberStr,
		unprocessedBlocks:          make(map[int64]*block),
		processedBlocks:            make(map[int64]struct{}),
		latestConsecutiveProcessed: store.GetHighest(),
	}

	if err := monitor.resubscribeToBlockchain(); err != nil {
		return nil, err
	}

	if err := monitor.resyncWithBlockchain(); err != nil {
		return nil, err
	}

	monitor.Go(monitor.worker)
	return monitor, nil
}
