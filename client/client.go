// client.go - coconut client API
// Copyright (C) 2018-2019  Jedrzej Stuczynski.
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

// Package client encapsulates all calls to issuers and providers.
package client

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	"github.com/nymtech/nym/client/config"
	"github.com/nymtech/nym/client/cryptoworker"
	"github.com/nymtech/nym/common/comm"
	"github.com/nymtech/nym/common/comm/commands"
	"github.com/nymtech/nym/common/comm/packet"
	pb "github.com/nymtech/nym/common/grpc/services"
	coconut "github.com/nymtech/nym/crypto/coconut/scheme"
	"github.com/nymtech/nym/crypto/elgamal"
	ethclient "github.com/nymtech/nym/ethereum/client"
	"github.com/nymtech/nym/logger"
	nymclient "github.com/nymtech/nym/tendermint/client"
	"google.golang.org/grpc"
	"gopkg.in/op/go-logging.v1"
)

// Client represents an user of Coconut network
type Client struct {
	cfg *config.Config
	log *logging.Logger

	// elGamalPrivateKey *elgamal.PrivateKey
	// elGamalPublicKey  *elgamal.PublicKey
	cryptoworker       *cryptoworker.CryptoWorker
	defaultDialOptions []grpc.DialOption

	privateKey *ecdsa.PrivateKey
	nymClient  *nymclient.Client
	ethClient  *ethclient.Client
}

// used to share code for parsing BlindSign and GetCredential responses. They return same data but under different name
type blindedSignatureResponse interface {
	GetSig() *coconut.ProtoBlindedSignature
	commands.ProtoResponse
}

const (
	nonGRPCClientErr = "Non-gRPC client trying to call gRPC method"
	gRPCClientErr    = "gRPC client trying to call non-gRPC method"
)

func (c *Client) RandomBIG() *Curve.BIG {
	return c.cryptoworker.CoconutWorker().RandomBIG()
}

func (c *Client) checkResponseStatus(resp commands.ProtoResponse) error {
	if resp == nil || resp.GetStatus() == nil {
		return c.logAndReturnError("checkResponseStatus: Received response (or part of it) was nil")
	}
	if resp.GetStatus().Code != int32(commands.StatusCode_OK) {
		return c.logAndReturnError(
			"checkResponseStatus: Received invalid response with status: %v. Error: %v",
			resp.GetStatus().Code,
			resp.GetStatus().Message,
		)
	}
	return nil
}

func (c *Client) parseVkResponse(resp *commands.VerificationKeyResponse) (*coconut.VerificationKey, error) {
	if err := c.checkResponseStatus(resp); err != nil {
		return nil, err
	}
	vk := &coconut.VerificationKey{}
	if err := vk.FromProto(resp.Vk); err != nil {
		return nil, c.logAndReturnError("parseVkResponse: Failed to unmarshal received verification key")
	}
	return vk, nil
}

func (c *Client) parseSignResponse(resp *commands.SignResponse) (*coconut.Signature, error) {
	if err := c.checkResponseStatus(resp); err != nil {
		return nil, err
	}
	sig := &coconut.Signature{}
	if err := sig.FromProto(resp.Sig); err != nil {
		return nil, c.logAndReturnError("parseSignResponse: Failed to unmarshal received signature")
	}
	return sig, nil
}

func (c *Client) parseBlindSignResponse(resp blindedSignatureResponse,
	elGamalPrivateKey *elgamal.PrivateKey,
) (*coconut.Signature, error) {
	if err := c.checkResponseStatus(resp); err != nil {
		return nil, err
	}
	blindSig := &coconut.BlindedSignature{}
	if err := blindSig.FromProto(resp.GetSig()); err != nil {
		return nil, c.logAndReturnError("parseBlindSignResponse: Failed to unmarshal received signature")
	}
	return c.cryptoworker.CoconutWorker().UnblindWrapper(blindSig, elGamalPrivateKey), nil
}

func (c *Client) getGrpcResponses(dialOptions []grpc.DialOption, request proto.Message) []*comm.ServerResponseGrpc {
	responses := make([]*comm.ServerResponseGrpc, len(c.cfg.Client.IAgRPCAddresses))
	respCh := make(chan *comm.ServerResponseGrpc)
	reqCh, cancelFuncs := c.sendGRPCs(respCh, dialOptions)

	go func() {
		for i := range c.cfg.Client.IAgRPCAddresses {
			c.log.Debug("Writing request to %v", c.cfg.Client.IAgRPCAddresses[i])
			reqCh <- &comm.ServerRequestGrpc{
				Message: request,
				ServerMetadata: &comm.ServerMetadata{
					Address: c.cfg.Client.IAgRPCAddresses[i],
				},
			}
		}
	}()

	c.waitForGrpcResponses(respCh, responses, cancelFuncs)
	close(reqCh)
	return responses
}

func (c *Client) waitForGrpcResponses(respCh <-chan *comm.ServerResponseGrpc,
	responses []*comm.ServerResponseGrpc,
	cancelFuncs []context.CancelFunc,
) {
	i := 0
	for {
		select {
		case resp := <-respCh:
			c.log.Debug("Received a reply from IA (%v)", resp.ServerMetadata.Address)
			responses[i] = resp
			i++

			if i == len(responses) {
				c.log.Debug("Got responses from all servers")
				return
			}
		case <-time.After(time.Duration(c.cfg.Debug.RequestTimeout) * time.Millisecond):
			c.log.Notice("Timed out while sending requests. Cancelling all requests in progress.")
			for _, cancel := range cancelFuncs {
				cancel()
			}
			return
		}
	}
}

// errcheck is ignored to make it not complain about not checking for err in conn.Close()
// nolint: errcheck
func (c *Client) sendGRPCs(respCh chan<- *comm.ServerResponseGrpc,
	dialOptions []grpc.DialOption,
) (chan<- *comm.ServerRequestGrpc, []context.CancelFunc) {
	reqCh := make(chan *comm.ServerRequestGrpc)

	// there can be at most that many connections active at given time,
	// as each goroutine can only access a single index and will overwrite its previous entry
	cancelFuncs := make([]context.CancelFunc, c.cfg.Client.MaxRequests)

	for i := 0; i < c.cfg.Client.MaxRequests; i++ {
		go func(i int) {
			for {
				req, ok := <-reqCh
				if !ok {
					return
				}
				c.log.Debugf("Dialing %v", req.ServerMetadata.Address)
				conn, err := grpc.Dial(req.ServerMetadata.Address, dialOptions...)
				if err != nil {
					c.log.Errorf("Could not dial %v (%v)", req.ServerMetadata.Address, err)
				}

				defer conn.Close()

				// in the case of a provider,
				// it will be sent to a single server so no need to make it possible to include it in the loop
				cc := pb.NewIssuerClient(conn)
				ctx, cancel := context.WithTimeout(context.Background(),
					time.Millisecond*time.Duration(c.cfg.Debug.ConnectTimeout),
				)
				cancelFuncs[i] = cancel
				defer func() {
					cancelFuncs[i] = nil
					cancel()
				}()

				var resp proto.Message
				var errgrpc error
				switch reqt := req.Message.(type) {
				case *commands.SignRequest:
					resp, errgrpc = cc.SignAttributes(ctx, reqt)
				case *commands.VerificationKeyRequest:
					resp, errgrpc = cc.GetVerificationKey(ctx, reqt)
				case *commands.BlindSignRequest:
					resp, errgrpc = cc.BlindSignAttributes(ctx, reqt)
				default:
					errstr := fmt.Sprintf("Unknown command was passed: %v", reflect.TypeOf(req.Message))
					errgrpc = errors.New(errstr)
					c.log.Warning(errstr)
				}
				if errgrpc != nil {
					c.log.Errorf("Failed to obtain signature from %v, err: %v", req.ServerMetadata.Address, err)
				} else {
					respCh <- &comm.ServerResponseGrpc{
						Message: resp,
						ServerMetadata: &comm.ServerMetadata{
							Address: req.ServerMetadata.Address,
						},
					}
				}
			}
		}(i)
	}
	return reqCh, cancelFuncs
}

// currently it tries to parse everything and just ignores an invalid request,
// should it fail on any single invalid request?
func (c *Client) parseSignatureServerResponses(
	responses []*comm.ServerResponse,
	isThreshold bool,
	isBlind bool,
	elGamalPrivateKey *elgamal.PrivateKey,
) ([]*coconut.Signature, *coconut.PolynomialPoints) {

	if responses == nil {
		return nil, nil
	}

	sigs := make([]*coconut.Signature, 0, len(responses))
	xs := make([]*Curve.BIG, 0, len(responses))
	for i := range responses {
		if responses[i] != nil && responses[i].ServerMetadata != nil {
			var resp commands.ProtoResponse
			if isBlind {
				resp = &commands.BlindSignResponse{}
			} else {
				resp = &commands.SignResponse{}
			}
			if err := proto.Unmarshal(responses[i].MarshaledData, resp); err != nil {
				c.log.Errorf("Failed to unmarshal response from: %v", responses[i].ServerMetadata.Address)
				continue
			}

			var sig *coconut.Signature
			var err error
			issuerID := int64(-1)
			if isBlind && elGamalPrivateKey != nil {
				sig, err = c.parseBlindSignResponse(resp.(*commands.BlindSignResponse), elGamalPrivateKey)
				if err != nil {
					continue
				}
				issuerID = resp.(*commands.BlindSignResponse).IssuerID
			} else {
				sig, err = c.parseSignResponse(resp.(*commands.SignResponse))
				if err != nil {
					continue
				}
				issuerID = resp.(*commands.SignResponse).IssuerID
			}

			if isThreshold && issuerID <= 0 {
				c.log.Errorf("Invalid IssuerID: %v", issuerID)
				continue
			} else if isThreshold {
				xs = append(xs, Curve.NewBIGint(int(issuerID)))
			}
			sigs = append(sigs, sig)
		}
	}
	if isThreshold {
		return sigs, coconut.NewPP(xs)
	}
	if len(sigs) != len(responses) {
		c.log.Errorf("This is not threshold system and some of the received responses were invalid")
		return nil, nil
	}
	return sigs, nil
}

// nolint: gocyclo
func (c *Client) handleReceivedSignatures(sigs []*coconut.Signature,
	pp *coconut.PolynomialPoints,
) (*coconut.Signature, error) {
	// TODO: the code has very similar structure to comm.HandleVks. Can it somehow be generalised?

	if len(sigs) == 0 {
		return nil, c.logAndReturnError("handleReceivedSignatures: No signatures provided")
	}

	if c.cfg.Client.Threshold == 0 && pp != nil {
		return nil, c.logAndReturnError("handleReceivedSignatures: Passed pp to a non-threshold system")
	}

	if c.cfg.Client.Threshold > 0 && pp == nil {
		return nil, c.logAndReturnError("handleReceivedSignatures: nil pp in a threshold system")
	}

	entriesToRemove, err := comm.ValidateIDs(c.log, pp, c.cfg.Client.Threshold > 0)
	if err != nil {
		return nil, err
	}

	for i := range sigs {
		if !sigs[i].Validate() {
			entriesToRemove[i] = true
		}
	}

	if len(entriesToRemove) > 0 {
		if c.cfg.Client.Threshold > 0 {
			newXs := make([]*Curve.BIG, 0, len(pp.Xs()))
			for i, x := range pp.Xs() {
				if _, ok := entriesToRemove[i]; !ok {
					newXs = append(newXs, x)
				}
			}
			pp = coconut.NewPP(newXs)
		}
		newSigs := make([]*coconut.Signature, 0, len(sigs))
		for i, sig := range sigs {
			if _, ok := entriesToRemove[i]; !ok {
				newSigs = append(newSigs, sig)
			}
		}
		sigs = newSigs
	}

	if len(sigs) >= c.cfg.Client.Threshold && len(sigs) > 0 {
		if c.cfg.Client.Threshold > 0 && len(sigs) != len(pp.Xs()) {
			return nil,
				c.logAndReturnError("handleReceivedSignatures: Inconsistent response, sigs: %v, pp: %v\n",
					len(sigs),
					len(pp.Xs()),
				)
		}
		c.log.Notice("Number of signatures received is within threshold")
	} else {
		return nil, c.logAndReturnError("handleReceivedSignatures: Received less than threshold number of signatures")
	}

	// we only want threshold number of them, in future randomly choose them?
	//nolint: dupl
	if c.cfg.Client.Threshold > 0 {
		sigs = sigs[:c.cfg.Client.Threshold]
		pp = coconut.NewPP(pp.Xs()[:c.cfg.Client.Threshold])
	} else if (!c.cfg.Client.UseGRPC && len(sigs) != len(c.cfg.Client.IAAddresses)) ||
		(c.cfg.Client.UseGRPC && len(sigs) != len(c.cfg.Client.IAgRPCAddresses)) {
		c.log.Error("No threshold, but obtained only %v out of %v signatures", len(sigs), len(c.cfg.Client.IAAddresses))
		c.log.Warning("This behaviour is currently undefined by requirements.")
		// should it continue regardless and assume the servers are down permanently or just terminate?
	}

	aSig := c.cryptoworker.CoconutWorker().AggregateSignaturesWrapper(sigs, pp)
	c.log.Debugf("Aggregated %v signatures (threshold: %v)", len(sigs), c.cfg.Client.Threshold)

	rSig := c.cryptoworker.CoconutWorker().RandomizeWrapper(aSig)
	c.log.Debug("Randomised the signature")

	return rSig, nil
}

// SignAttributesGrpc sends sign request to all IA-grpc servers specified in the config
// with given set of public attributes.
// In the case of threshold system, first t results are aggregated and the result is randomised and returned.
// Otherwise all results are aggregated and then randomised.
// Error is returned if insufficient number of signatures was received.
func (c *Client) SignAttributesGrpc(pubM []*Curve.BIG) (*coconut.Signature, error) {
	if !c.cfg.Client.UseGRPC {
		return nil, c.logAndReturnError(nonGRPCClientErr)
	}

	grpcDialOptions := c.defaultDialOptions
	isThreshold := c.cfg.Client.Threshold > 0

	signRequest, err := commands.NewSignRequest(pubM)
	if err != nil {
		return nil, c.logAndReturnError("SignAttributesGrpc: Failed to create Sign request: %v", err)
	}

	c.log.Notice("Going to send Sign request (via gRPCs) to %v IAs", len(c.cfg.Client.IAgRPCAddresses))
	responses := c.getGrpcResponses(grpcDialOptions, signRequest)

	sigs := make([]*coconut.Signature, 0, len(c.cfg.Client.IAgRPCAddresses))
	xs := make([]*Curve.BIG, 0, len(c.cfg.Client.IAgRPCAddresses))

	for i := range responses {
		if responses[i] == nil {
			c.log.Error("nil response received")
			continue
		}
		sig, err := c.parseSignResponse(responses[i].Message.(*commands.SignResponse))
		if err != nil {
			continue
		}
		sigs = append(sigs, sig)
		if isThreshold {
			xs = append(xs, Curve.NewBIGint(int(responses[i].Message.(*commands.SignResponse).IssuerID)))
		}
	}
	if c.cfg.Client.Threshold > 0 {
		return c.handleReceivedSignatures(sigs, coconut.NewPP(xs))
	}
	return c.handleReceivedSignatures(sigs, nil)
}

// SignAttributes sends sign request to all IA servers specified in the config
// using TCP sockets with given set of public attributes.
// In the case of threshold system, first t results are aggregated and the result is randomised and returned.
// Otherwise all results are aggregated and then randomised.
// Error is returned if insufficient number of signatures was received.
func (c *Client) SignAttributes(pubM []*Curve.BIG) (*coconut.Signature, error) {
	if c.cfg.Client.UseGRPC {
		return nil, c.logAndReturnError(gRPCClientErr)
	}

	cmd, err := commands.NewSignRequest(pubM)
	if err != nil {
		return nil, c.logAndReturnError("SignAttributes: Failed to create Sign request: %v", err)
	}

	packetBytes, err := commands.CommandToMarshalledPacket(cmd)
	if err != nil {
		return nil, c.logAndReturnError("SignAttributes: Could not create data packet for sign command: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.cfg.Debug.RequestTimeout)*time.Millisecond)
	defer cancel()

	c.log.Notice("Going to send Sign request (via TCP socket) to %v IAs", len(c.cfg.Client.IAAddresses))
	responses := comm.GetServerResponses(
		ctx,
		&comm.RequestParams{
			MarshaledPacket:   packetBytes,
			MaxRequests:       c.cfg.Client.MaxRequests,
			ConnectionTimeout: time.Duration(c.cfg.Debug.ConnectTimeout) * time.Millisecond,
			ServerAddresses:   c.cfg.Client.IAAddresses,
		},
		c.log,
	)
	return c.handleReceivedSignatures(c.parseSignatureServerResponses(responses, c.cfg.Client.Threshold > 0, false, nil))
}

// nolint: gocyclo
func (c *Client) handleReceivedVerificationKeys(vks []*coconut.VerificationKey,
	pp *coconut.PolynomialPoints,
	shouldAggregate bool,
) ([]*coconut.VerificationKey, error) {
	vks, pp, err := comm.HandleVks(c.log, vks, pp, c.cfg.Client.Threshold)
	if err != nil {
		// error was already logged at HandleVks
		return nil, err
	}

	// we only want threshold number of them, in future randomly choose them?
	//nolint: dupl
	if c.cfg.Client.Threshold > 0 {
		vks = vks[:c.cfg.Client.Threshold]
		pp = coconut.NewPP(pp.Xs()[:c.cfg.Client.Threshold])
	} else if (!c.cfg.Client.UseGRPC && len(vks) != len(c.cfg.Client.IAAddresses)) ||
		(c.cfg.Client.UseGRPC && len(vks) != len(c.cfg.Client.IAgRPCAddresses)) {
		c.log.Error("No threshold, but obtained only %v out of %v verification keys", len(vks), len(c.cfg.Client.IAAddresses))
		c.log.Warning("This behaviour is currently undefined by requirements.")
		// should it continue regardless and assume the servers are down permanently or just terminate?
	}

	if shouldAggregate {
		avk := c.cryptoworker.CoconutWorker().AggregateVerificationKeysWrapper(vks, pp)
		c.log.Debugf("Aggregated %v verification keys (threshold: %v)", len(vks), c.cfg.Client.Threshold)

		return []*coconut.VerificationKey{avk}, nil
	}
	return vks, nil
}

// GetVerificationKeysGrpc sends GetVerificationKey request to all IA-grpc servers specified in the config.
// If the flag 'shouldAggregate' is set to true, the returned slice will consist of a single element,
// which will be the aggregated verification key.
// In the case of threshold system, first t results are aggregated, otherwise all results are aggregated.
// Error is returned if insufficient number of verification keys was received.
func (c *Client) GetVerificationKeysGrpc(shouldAggregate bool) ([]*coconut.VerificationKey, error) {
	return nil, errors.New("Implementation details has changed and grpc version hasnt been updated yet")

	if !c.cfg.Client.UseGRPC {
		return nil, c.logAndReturnError(nonGRPCClientErr)
	}

	grpcDialOptions := c.defaultDialOptions
	isThreshold := c.cfg.Client.Threshold > 0

	verificationKeyRequest, err := commands.NewVerificationKeyRequest()
	if err != nil {
		return nil, c.logAndReturnError("GetVerificationKeysGrpc: Failed to create Vk request: %v", err)
	}

	c.log.Notice("Going to send GetVk request (via gRPCs) to %v IAs", len(c.cfg.Client.IAgRPCAddresses))
	responses := c.getGrpcResponses(grpcDialOptions, verificationKeyRequest)

	vks := make([]*coconut.VerificationKey, 0, len(c.cfg.Client.IAgRPCAddresses))
	xs := make([]*Curve.BIG, 0, len(c.cfg.Client.IAgRPCAddresses))

	for i := range responses {
		if responses[i] == nil {
			c.log.Error("nil response received")
			continue
		}
		vk, err := c.parseVkResponse(responses[i].Message.(*commands.VerificationKeyResponse))
		if err != nil {
			continue
		}
		vks = append(vks, vk)
		if isThreshold {
			xs = append(xs, Curve.NewBIGint(int(responses[i].Message.(*commands.VerificationKeyResponse).IssuerID)))
		}
	}

	// TODO: FIXME: WHY WAS I SORTING THIS SLICE??
	// // works under assumption that servers specified in config file are ordered by their IDs
	// // which will in most cases be the case since they're just going to be 1,2,.., etc.
	// sort.Slice(responses, func(i, j int) bool { return responses[i].ServerMetadata.ID < responses[j].ServerMetadata.ID })

	if c.cfg.Client.Threshold > 0 {
		return c.handleReceivedVerificationKeys(vks, coconut.NewPP(xs), shouldAggregate)
	}
	return c.handleReceivedVerificationKeys(vks, nil, shouldAggregate)
}

// GetVerificationKeys sends GetVerificationKey request to all IA servers specified in the config using TCP sockets.
// If the flag 'shouldAggregate' is set to true, the returned slice will consist of a single element,
// which will be the aggregated verification key.
// In the case of threshold system, first t results are aggregated, otherwise all results are aggregated.
// Error is returned if insufficient number of verification keys was received.
func (c *Client) GetVerificationKeys(shouldAggregate bool) ([]*coconut.VerificationKey, error) {
	if c.cfg.Client.UseGRPC {
		return nil, c.logAndReturnError(gRPCClientErr)
	}

	cmd, err := commands.NewVerificationKeyRequest()
	if err != nil {
		return nil, c.logAndReturnError("GetVerificationKeys: Failed to create Vk request: %v", err)
	}

	packetBytes, err := commands.CommandToMarshalledPacket(cmd)
	if err != nil {
		return nil, c.logAndReturnError("GetVerificationKeys: Could not create data packet for getVK command: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.cfg.Debug.RequestTimeout)*time.Millisecond)
	defer cancel()

	c.log.Notice("Going to send GetVK request (via TCP socket) to %v IAs", len(c.cfg.Client.IAAddresses))
	responses := comm.GetServerResponses(
		ctx,
		&comm.RequestParams{
			MarshaledPacket:   packetBytes,
			MaxRequests:       c.cfg.Client.MaxRequests,
			ConnectionTimeout: time.Duration(c.cfg.Debug.ConnectTimeout) * time.Millisecond,
			ServerAddresses:   c.cfg.Client.IAAddresses,
		},
		c.log,
	)
	vks, pp := comm.ParseVerificationKeyResponses(responses, c.cfg.Client.Threshold > 0, c.log)
	return c.handleReceivedVerificationKeys(vks, pp, shouldAggregate)
}

// GetAggregateVerificationKeyGrpc is basically a wrapper for GetVerificationKeysGrpc,
// but returns a single vk rather than slice with one element.
func (c *Client) GetAggregateVerificationKeyGrpc() (*coconut.VerificationKey, error) {
	vks, err := c.GetVerificationKeysGrpc(true)
	if len(vks) == 1 && err == nil {
		return vks[0], nil
	}
	return nil, err
}

// GetAggregateVerificationKey is basically a wrapper for GetVerificationKeys,
// but returns a single vk rather than slice with one element.
func (c *Client) GetAggregateVerificationKey() (*coconut.VerificationKey, error) {
	vks, err := c.GetVerificationKeys(true)
	if len(vks) == 1 && err == nil {
		return vks[0], nil
	}
	return nil, err
}

// BlindSignAttributesGrpc sends blind sign request to all IA-grpc servers specified in the config
// with given set of public and private attributes.
// In the case of threshold system, after unblinding all results,
// first t results are aggregated and the result is randomised and returned.
// Otherwise all unblinded results are aggregated and then randomised.
// Error is returned if insufficient number of signatures was received.
func (c *Client) BlindSignAttributesGrpc(pubM []*Curve.BIG, privM []*Curve.BIG) (*coconut.Signature, error) {
	if !c.cfg.Client.UseGRPC {
		return nil, c.logAndReturnError(nonGRPCClientErr)
	}
	grpcDialOptions := c.defaultDialOptions
	isThreshold := c.cfg.Client.Threshold > 0

	elGamalPrivateKey, elGamalPublicKey := c.cryptoworker.CoconutWorker().ElGamalKeygenWrapper()

	if !coconut.ValidateBigSlice(pubM) || !coconut.ValidateBigSlice(privM) {
		return nil, c.logAndReturnError("BlindSignAttributesGrpc: invalid slice of attributes provided")
	}

	lambda, err := c.cryptoworker.CoconutWorker().PrepareBlindSignWrapper(elGamalPublicKey, pubM, privM)
	if err != nil {
		return nil, c.logAndReturnError("BlindSignAttributesGrpc: Could not create lambda: %v", err)
	}

	blindSignRequest, err := commands.NewBlindSignRequest(lambda, elGamalPublicKey, pubM)
	if err != nil {
		return nil, c.logAndReturnError("BlindSignAttributesGrpc: Failed to create BlindSign request: %v", err)
	}

	c.log.Notice("Going to send Blind Sign request (via gRPCs) to %v IAs", len(c.cfg.Client.IAgRPCAddresses))
	responses := c.getGrpcResponses(grpcDialOptions, blindSignRequest)

	sigs := make([]*coconut.Signature, 0, len(c.cfg.Client.IAgRPCAddresses))
	xs := make([]*Curve.BIG, 0, len(c.cfg.Client.IAgRPCAddresses))

	for i := range responses {
		if responses[i] == nil {
			c.log.Error("nil response received")
			continue
		}
		sig, err := c.parseBlindSignResponse(responses[i].Message.(*commands.BlindSignResponse), elGamalPrivateKey)
		if err != nil {
			continue
		}
		sigs = append(sigs, sig)
		if isThreshold {
			xs = append(xs, Curve.NewBIGint(int(responses[i].Message.(*commands.BlindSignResponse).IssuerID)))
		}
	}
	if c.cfg.Client.Threshold > 0 {
		return c.handleReceivedSignatures(sigs, coconut.NewPP(xs))
	}
	return c.handleReceivedSignatures(sigs, nil)
}

// BlindSignAttributes sends sign request to all IA servers specified in the config
// using TCP sockets with given set of public and private attributes.
// In the case of threshold system, after unblinding all results,
// first t results are aggregated and the result is randomised and returned.
// Otherwise all unblinded results are aggregated and then randomised.
// Error is returned if insufficient number of signatures was received.
func (c *Client) BlindSignAttributes(pubM []*Curve.BIG, privM []*Curve.BIG) (*coconut.Signature, error) {
	if c.cfg.Client.UseGRPC {
		return nil, c.logAndReturnError(gRPCClientErr)
	}

	elGamalPrivateKey, elGamalPublicKey := c.cryptoworker.CoconutWorker().ElGamalKeygenWrapper()

	if !coconut.ValidateBigSlice(pubM) || !coconut.ValidateBigSlice(privM) {
		return nil, c.logAndReturnError("BlindSignAttributes: invalid slice of attributes provided")
	}

	lambda, err := c.cryptoworker.CoconutWorker().PrepareBlindSignWrapper(elGamalPublicKey, pubM, privM)
	if err != nil {
		return nil, c.logAndReturnError("BlindSignAttributes: Could not create lambda: %v", err)
	}

	cmd, err := commands.NewBlindSignRequest(lambda, elGamalPublicKey, pubM)
	if err != nil {
		return nil, c.logAndReturnError("BlindSignAttributes: Failed to create BlindSign request: %v", err)
	}

	packetBytes, err := commands.CommandToMarshalledPacket(cmd)
	if err != nil {
		return nil, c.logAndReturnError("BlindSignAttributes: Could not create data packet for blind sign command: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.cfg.Debug.RequestTimeout)*time.Millisecond)
	defer cancel()

	c.log.Notice("Going to send Blind Sign request to %v IAs", len(c.cfg.Client.IAAddresses))
	responses := comm.GetServerResponses(
		ctx,
		&comm.RequestParams{
			MarshaledPacket:   packetBytes,
			MaxRequests:       c.cfg.Client.MaxRequests,
			ConnectionTimeout: time.Duration(c.cfg.Debug.ConnectTimeout) * time.Millisecond,
			ServerAddresses:   c.cfg.Client.IAAddresses,
		},
		c.log,
	)
	sigs, pp := c.parseSignatureServerResponses(responses, c.cfg.Client.Threshold > 0, true, elGamalPrivateKey)
	return c.handleReceivedSignatures(sigs, pp)
}

// SendCredentialsForVerificationGrpc sends a gRPC request to verify
// obtained credentials to some specified provider server.
// errcheck is ignored to make it not complain about not checking for err in conn.Close()
// nolint: errcheck
func (c *Client) SendCredentialsForVerificationGrpc(pubM []*Curve.BIG,
	sig *coconut.Signature,
	addr string,
) (bool, error) {
	if !c.cfg.Client.UseGRPC {
		return false, c.logAndReturnError(nonGRPCClientErr)
	}
	grpcDialOptions := c.defaultDialOptions
	verifyRequest, err := commands.NewVerifyRequest(pubM, sig)
	if err != nil {
		return false, c.logAndReturnError("SendCredentialsForVerificationGrpc: Failed to create Verify request: %v", err)
	}

	c.log.Debugf("Dialing %v", addr)
	conn, err := grpc.Dial(addr, grpcDialOptions...)
	if err != nil {
		return false, c.logAndReturnError("SendCredentialsForVerificationGrpc: Could not dial %v (%v)", addr, err)
	}
	defer conn.Close()
	cc := pb.NewProviderClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(c.cfg.Debug.ConnectTimeout))
	defer cancel()

	r, err := cc.VerifyCredentials(ctx, verifyRequest)
	if err != nil {
		return false,
			c.logAndReturnError("SendCredentialsForVerificationGrpc: Failed to receive response to verification request: %v",
				err,
			)
	} else if r.GetStatus().Code != int32(commands.StatusCode_OK) {
		return false, c.logAndReturnError(
			"SendCredentialsForVerificationGrpc: Received invalid response with status: %v. Error: %v",
			r.GetStatus().Code,
			r.GetStatus().Message,
		)
	}
	return r.GetIsValid(), nil
}

// SendCredentialsForVerification sends a TCP request to verify obtained credentials to some specified provider server.
func (c *Client) SendCredentialsForVerification(pubM []*Curve.BIG, sig *coconut.Signature, addr string) (bool, error) {
	if c.cfg.Client.UseGRPC {
		return false, c.logAndReturnError(gRPCClientErr)
	}
	cmd, err := commands.NewVerifyRequest(pubM, sig)
	if err != nil {
		return false, c.logAndReturnError("SendCredentialsForVerification: Failed to create Verify request: %v", err)
	}
	packetBytes, err := commands.CommandToMarshalledPacket(cmd)
	if err != nil {
		return false, c.logAndReturnError("Could not create data packet for verify command: %v", err)
	}

	c.log.Debugf("Dialing %v", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false, c.logAndReturnError("SendCredentialsForVerification: Could not dial %v (%v)", addr, err)
	}

	// currently will never be thrown since there is no writedeadline
	if _, werr := conn.Write(packetBytes); werr != nil {
		return false, c.logAndReturnError("SendCredentialsForVerification: Failed to write to connection: %v", werr)
	}

	sderr := conn.SetReadDeadline(time.Now().Add(time.Duration(c.cfg.Debug.ConnectTimeout) * time.Millisecond))
	if sderr != nil {
		return false,
			c.logAndReturnError("SendCredentialsForVerification: Failed to set read deadline for connection: %v",
				sderr,
			)
	}

	respPacket, err := comm.ReadPacketFromConn(conn)
	if err != nil {
		return false,
			c.logAndReturnError("SendCredentialsForVerification: Received invalid response from %v: %v",
				addr,
				err,
			)
	}

	verifyResponse := &commands.VerifyResponse{}
	if err := proto.Unmarshal(respPacket.Payload(), verifyResponse); err != nil {
		return false,
			c.logAndReturnError("SendCredentialsForVerification: Failed to recover verification result: %v",
				err,
			)
	} else if verifyResponse.GetStatus().Code != int32(commands.StatusCode_OK) {
		return false, c.logAndReturnError(
			"SendCredentialsForVerification: Received invalid response with status: %v. Error: %v",
			verifyResponse.GetStatus().Code,
			verifyResponse.GetStatus().Message,
		)
	}

	return verifyResponse.IsValid, nil
}

func (c *Client) parseBlindVerifyResponse(packetResponse *packet.Packet) (bool, error) {
	blindVerifyResponse := &commands.BlindVerifyResponse{}
	if err := proto.Unmarshal(packetResponse.Payload(), blindVerifyResponse); err != nil {
		return false, c.logAndReturnError("parseBlindVerifyResponse: Failed to recover verification result: %v", err)
	} else if blindVerifyResponse.GetStatus().Code != int32(commands.StatusCode_OK) {
		return false, c.logAndReturnError(
			"parseBlindVerifyResponse: Received invalid response with status: %v. Error: %v",
			blindVerifyResponse.GetStatus().Code,
			blindVerifyResponse.GetStatus().Message,
		)
	}
	return blindVerifyResponse.IsValid, nil
}

func (c *Client) prepareBlindVerifyRequest(pubM []*Curve.BIG,
	privM []*Curve.BIG,
	sig *coconut.Signature,
	vk *coconut.VerificationKey,
) (*commands.BlindVerifyRequest, error) {
	var err error
	if vk == nil {
		if c.cfg.Client.UseGRPC {
			vk, err = c.GetAggregateVerificationKeyGrpc()
		} else {
			vk, err = c.GetAggregateVerificationKey()
		}
		if err != nil {
			return nil,
				c.logAndReturnError("prepareBlindVerifyRequest: "+
					"Could not obtain aggregate verification key required to create proofs for verification: %v",
					err,
				)
		}
	}

	theta, err := c.cryptoworker.CoconutWorker().ShowBlindSignatureWrapper(vk, sig, privM)
	if err != nil {
		return nil, c.logAndReturnError("prepareBlindVerifyRequest: Failed when creating proofs for verification: %v", err)
	}

	blindVerifyRequest, err := commands.NewBlindVerifyRequest(theta, sig, pubM)
	if err != nil {
		return nil, c.logAndReturnError("prepareBlindVerifyRequest: Failed to create BlindVerify request: %v", err)
	}
	return blindVerifyRequest, nil
}

// SendCredentialsForBlindVerificationGrpc sends a gRPC request to verify
// obtained blind credentials to some specified provider server.
// If client does not provide aggregate verification key, the call will first try to obtain it.
// errcheck is ignored to make it not complain about not checking for err in conn.Close()
// nolint: errcheck
func (c *Client) SendCredentialsForBlindVerificationGrpc(pubM []*Curve.BIG,
	privM []*Curve.BIG,
	sig *coconut.Signature,
	addr string,
	vk *coconut.VerificationKey,
) (bool, error) {
	if !c.cfg.Client.UseGRPC {
		return false, c.logAndReturnError(nonGRPCClientErr)
	}
	grpcDialOptions := c.defaultDialOptions
	blindVerifyRequest, err := c.prepareBlindVerifyRequest(pubM, privM, sig, vk)
	if err != nil {
		return false,
			c.logAndReturnError("SendCredentialsForBlindVerificationGrpc: Failed to prepare blindverifyrequest: %v",
				err,
			)
	}

	c.log.Debugf("Dialing %v", addr)
	conn, err := grpc.Dial(addr, grpcDialOptions...)
	if err != nil {
		c.log.Errorf("Could not dial %v (%v)", addr, err)
	}
	defer conn.Close()
	cc := pb.NewProviderClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(c.cfg.Debug.ConnectTimeout))
	defer cancel()

	r, err := cc.BlindVerifyCredentials(ctx, blindVerifyRequest)
	if err != nil {
		return false,
			c.logAndReturnError("SendCredentialsForBlindVerificationGrpc: "+
				"Failed to receive response to verification request: %v",
				err,
			)
	} else if r.GetStatus().Code != int32(commands.StatusCode_OK) {
		return false, c.logAndReturnError(
			"SendCredentialsForBlindVerificationGrpc: Received invalid response with status: %v. Error: %v",
			r.GetStatus().Code,
			r.GetStatus().Message,
		)
	}
	return r.GetIsValid(), nil
}

// SendCredentialsForBlindVerification sends a TCP request to verify
// obtained blind credentials to some specified provider server.
// If client does not provide aggregate verification key, the call will first try to obtain it.
// nolint: dupl
func (c *Client) SendCredentialsForBlindVerification(pubM []*Curve.BIG,
	privM []*Curve.BIG,
	sig *coconut.Signature,
	addr string,
	vk *coconut.VerificationKey,
) (bool, error) {
	if c.cfg.Client.UseGRPC {
		return false, c.logAndReturnError(gRPCClientErr)
	}

	blindVerifyRequest, err := c.prepareBlindVerifyRequest(pubM, privM, sig, vk)
	if err != nil {
		return false,
			c.logAndReturnError("SendCredentialsForBlindVerification: Failed to prepare blindverifyrequest: %v",
				err,
			)
	}

	packetBytes, err := commands.CommandToMarshalledPacket(blindVerifyRequest)
	if err != nil {
		return false, c.logAndReturnError("Could not create data packet for blind verify command: %v", err)
	}

	c.log.Debugf("Dialing %v", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return false, c.logAndReturnError("SendCredentialsForBlindVerification: Could not dial %v (%v)", addr, err)
	}

	// currently will never be thrown since there is no writedeadline
	if _, werr := conn.Write(packetBytes); werr != nil {
		return false, c.logAndReturnError("SendCredentialsForBlindVerification: Failed to write to connection: %v", werr)
	}

	sderr := conn.SetReadDeadline(time.Now().Add(time.Duration(c.cfg.Debug.ConnectTimeout) * time.Millisecond))
	if sderr != nil {
		return false,
			c.logAndReturnError("SendCredentialsForBlindVerification: Failed to set read deadline for connection: %v",
				sderr,
			)
	}

	resp, err := comm.ReadPacketFromConn(conn)
	if err != nil {
		return false,
			c.logAndReturnError("SendCredentialsForBlindVerification: Received invalid response from %v: %v",
				addr,
				err,
			)
	}
	return c.parseBlindVerifyResponse(resp)
}

func (c *Client) logAndReturnError(fmtString string, a ...interface{}) error {
	errstr := fmtString
	if a != nil {
		errstr = fmt.Sprintf(fmtString, a...)
	}
	c.log.Error(errstr)
	return errors.New(errstr)
}

// Stop stops client instance
func (c *Client) Stop() {
	c.log.Notice("Starting graceful shutdown.")
	c.cryptoworker.Halt()
	c.log.Notice("Shutdown complete.")
}

// New returns a new Client instance parameterized with the specified configuration.
// nolint: gocyclo
func New(cfg *config.Config) (*Client, error) {
	// there is no need to further validate it, as if it's not nil, it was already done
	if cfg == nil {
		return nil, errors.New("nil config provided")
	}

	log, err := logger.New(cfg.Logging.File, cfg.Logging.Level, cfg.Logging.Disable)
	if err != nil {
		return nil, fmt.Errorf("failed to create a logger: %v", err)
	}
	clientLog := log.GetLogger("Client")
	clientLog.Noticef("Logging level set to %v", cfg.Logging.Level)

	if cfg.Nym == nil || cfg.Nym.AccountKeysFile == "" {
		clientLog.Error("No keys for the Nym Blockchain were specified.")
		return nil, errors.New("could not load blockchain keys")
	}

	privateKey, loadErr := ethcrypto.LoadECDSA(cfg.Nym.AccountKeysFile)
	if loadErr != nil {
		errStr := fmt.Sprintf("Failed to load Nym keys: %v", loadErr)
		clientLog.Error(errStr)
		return nil, errors.New(errStr)
	}
	clientLog.Notice("Loaded Nym Blochain keys from the file.")

	nymClient, err := nymclient.New(cfg.Nym.BlockchainNodeAddresses, log)
	if err != nil {
		errStr := fmt.Sprintf("Failed to create a nymClient: %v", err)
		clientLog.Error(errStr)
		return nil, errors.New(errStr)
	}

	ethCfg := ethclient.NewConfig(
		privateKey,
		cfg.Nym.EthereumNodeAddresses[0],
		cfg.Nym.NymContract,
		cfg.Nym.PipeAccount,
		log,
	)

	ethClient, err := ethclient.New(ethCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ethereum client: %v", err)
	}

	params, err := coconut.Setup(cfg.Client.MaximumAttributes)
	if err != nil {
		return nil, errors.New("error while generating coconut params")
	}

	cryptoworker := cryptoworker.New(uint64(1), log, params, cfg.Debug.NumJobWorkers)
	clientLog.Notice("Started Coconut Worker")

	c := &Client{
		cfg: cfg,
		log: clientLog,

		cryptoworker: cryptoworker,

		defaultDialOptions: []grpc.DialOption{
			grpc.WithInsecure(),
		},

		privateKey: privateKey,
		nymClient:  nymClient,
		ethClient:  ethClient,
	}

	clientLog.Noticef("Created %v client", cfg.Client.Identifier)
	return c, nil
}
