// commands.go - commands for coconut server
// Copyright (C) 2018  Jedrzej Stuczynski.
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

// Package commands define command types used by coconut server.
package commands

import (
	"context"
	"errors"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/proto"
	Curve "github.com/jstuczyn/amcl/version3/go/amcl/BLS381"
	"github.com/nymtech/nym/common/comm/packet"
	"github.com/nymtech/nym/constants"
	coconut "github.com/nymtech/nym/crypto/coconut/scheme"
	"github.com/nymtech/nym/crypto/elgamal"
)

const (
	// GetVerificationKeyID is commandID for getting server's verification key.
	GetVerificationKeyID CommandID = 100

	// SignID is commandID for signing public attributes.
	SignID CommandID = 101

	// VerifyID is commandID for verifying a signature on public attributes.
	VerifyID CommandID = 102

	// BlindSignID is commandID for blindly signing public and private attributes.
	BlindSignID CommandID = 103

	// BlindVerifyID is commandID for verifying a blind signature on public and private attributes.
	BlindVerifyID CommandID = 104

	// SpendCredentialID is commandID for spending given credential at particular provider.
	SpendCredentialID CommandID = 129

	// LookUpCredentialID is commandID for looking up credential issued at particular block height
	// identified by particular Gamma.
	LookUpCredentialID CommandID = 130

	// LookUpBlockCredentialsID is commandID for looking up all credentials issued at particular block height.
	LookUpBlockCredentialsID CommandID = 131

	// CredentialVerificationID is a commandID for an internal command of a verifier to verify credential
	// and notify tendermint chain of the result.
	// It is internal to the verifiers, however, it is defined here for the consistency sake.
	CredentialVerificationID CommandID = 132

	// DefaultResponseErrorStatusCode defines default value for the error status code of a server response.
	DefaultResponseErrorStatusCode = StatusCode_UNKNOWN
	// DefaultResponseErrorMessage defines default value for the error message of a server response.
	DefaultResponseErrorMessage = ""
)

// Command defines interface that is implemented by all commands defined in the package.
type Command interface {
	// basically generated protocol buffer messages
	Reset()
	String() string
	ProtoMessage()
}

// CommandID is wrapper for a byte defining ID of particular command.
type CommandID byte

// CommandToMarshalledPacket transforms the given command into a marshalled instance of a packet
// sent to a TCP socket.
func CommandToMarshalledPacket(cmd Command) ([]byte, error) {
	payloadBytes, err := proto.Marshal(cmd)
	if err != nil {
		return nil, err
	}
	var cmdID CommandID
	switch cmd.(type) {
	case *SignRequest:
		cmdID = SignID
	case *VerifyRequest:
		cmdID = VerifyID
	case *BlindSignRequest:
		cmdID = BlindSignID
	case *BlindVerifyRequest:
		cmdID = BlindVerifyID
	case *SpendCredentialRequest:
		cmdID = SpendCredentialID
	case *VerificationKeyRequest:
		cmdID = GetVerificationKeyID
	case *LookUpCredentialRequest:
		cmdID = LookUpCredentialID
	case *LookUpBlockCredentialsRequest:
		cmdID = LookUpBlockCredentialsID
	case *CredentialVerificationRequest:
		cmdID = CredentialVerificationID
	default:
		return nil, errors.New("unknown Command")
	}

	rawCmd := NewRawCommand(cmdID, payloadBytes)
	cmdBytes := rawCmd.ToBytes()

	packetIn := packet.NewPacket(cmdBytes)
	packetBytes, err := packetIn.MarshalBinary()
	if err != nil {
		return nil, err
	}
	return packetBytes, nil
}

// FromBytes creates a given Command object out of stream of bytes.
func FromBytes(b []byte) (Command, error) {
	id := CommandID(b[0])
	payload := b[1:]
	var cmd Command
	switch id {
	case GetVerificationKeyID:
		cmd = &VerificationKeyRequest{}
	case SignID:
		cmd = &SignRequest{}
	case VerifyID:
		cmd = &VerifyRequest{}
	case BlindSignID:
		cmd = &BlindSignRequest{}
	case BlindVerifyID:
		cmd = &BlindVerifyRequest{}
	case SpendCredentialID:
		cmd = &SpendCredentialRequest{}
	case LookUpCredentialID:
		cmd = &LookUpCredentialRequest{}
	case LookUpBlockCredentialsID:
		cmd = &LookUpBlockCredentialsRequest{}
	case CredentialVerificationID:
		cmd = &CredentialVerificationRequest{}
	default:
		return nil, errors.New("unknown CommandID")
	}

	if err := proto.Unmarshal(payload, cmd); err != nil {
		return nil, err
	}

	return cmd, nil
}

// RawCommand encapsulates arbitrary marshalled command and ID that defines it.
type RawCommand struct {
	id      CommandID
	payload []byte
}

// NewRawCommand creates new instance of RawCommand given ID and its payload.
func NewRawCommand(id CommandID, payload []byte) *RawCommand {
	return &RawCommand{id, payload}
}

// ID returns CommandID of RawCommand.
func (c *RawCommand) ID() CommandID {
	return c.id
}

// Payload returns Payload of RawCommand.
func (c *RawCommand) Payload() []byte {
	return c.payload
}

// ToBytes marshals RawCommand into a stream of bytes so that it could be turned into a packet.
func (c *RawCommand) ToBytes() []byte {
	b := make([]byte, 1+len(c.payload))
	b[0] = byte(c.id)
	copy(b[1:], c.payload)
	return b
}

// CommandRequest defines set of Command and chan that is used by client workers.
type CommandRequest struct {
	ctx   context.Context
	cmd   Command
	retCh chan *Response
}

// NewCommandRequest creates new instance of CommandRequest.
func NewCommandRequest(cmd Command, ch chan *Response) *CommandRequest {
	return &CommandRequest{cmd: cmd, retCh: ch}
}

// RetCh returns return channel of CommandRequest.
func (cr *CommandRequest) RetCh() chan *Response {
	return cr.retCh
}

// Cmd returns command of CommandRequest.
func (cr *CommandRequest) Cmd() Command {
	return cr.cmd
}

// Ctx returns context attached to the CommandRequest.
func (cr *CommandRequest) Ctx() context.Context {
	return cr.ctx
}

// WithContext attaches context to given CommandRequest.
func (cr *CommandRequest) WithContext(ctx context.Context) {
	cr.ctx = ctx
}

// Response represents a server response to client's query.
type Response struct {
	Data         interface{}
	ErrorStatus  StatusCode
	ErrorMessage string
}

// ProtoResponse is a protobuf server response.
type ProtoResponse interface {
	Reset()
	String() string
	ProtoMessage()
	GetStatus() *Status
}

// NewSignRequest returns new instance of a SignRequest given set of public attributes.
func NewSignRequest(pubM []*Curve.BIG) (*SignRequest, error) {
	if len(pubM) == 0 {
		return nil, errors.New("no attributes for signing")
	}
	pubMb, err := coconut.BigSliceToByteSlices(pubM)
	if err != nil {
		return nil, err
	}
	return &SignRequest{
		PubM: pubMb,
	}, nil
}

// NewVerificationKeyRequest returns new instance of a VerificationKeyRequest.
func NewVerificationKeyRequest() (*VerificationKeyRequest, error) {
	return &VerificationKeyRequest{}, nil
}

// NewVerifyRequest returns new instance of a VerifyRequest
// given set of public attributes and a coconut signature on them.
func NewVerifyRequest(pubM []*Curve.BIG, sig *coconut.Signature) (*VerifyRequest, error) {
	if len(pubM) == 0 {
		return nil, errors.New("no attributes for verifying")
	}
	protoSig, err := sig.ToProto()
	if err != nil {
		return nil, err
	}
	pubMb, err := coconut.BigSliceToByteSlices(pubM)
	if err != nil {
		return nil, err
	}
	return &VerifyRequest{
		Sig:  protoSig,
		PubM: pubMb,
	}, nil
}

// NewBlindSignRequest returns new instance of a BlindSignRequest
// given set of public attributes, lambda and corresponding ElGamal public key.
func NewBlindSignRequest(lambda *coconut.Lambda,
	egPub *elgamal.PublicKey,
	pubM []*Curve.BIG,
) (*BlindSignRequest, error) {
	protoLambda, err := lambda.ToProto()
	if err != nil {
		return nil, err
	}
	protoEgPub, err := egPub.ToProto()
	if err != nil {
		return nil, err
	}
	pubMb, err := coconut.BigSliceToByteSlices(pubM)
	if err != nil {
		return nil, err
	}
	return &BlindSignRequest{
		Lambda: protoLambda,
		EgPub:  protoEgPub,
		PubM:   pubMb,
	}, nil
}

// NewBlindVerifyRequest returns new instance of a BlinfVerifyRequest
// given set of public attributes, theta and a coconut signature on them.
func NewBlindVerifyRequest(theta *coconut.Theta,
	sig *coconut.Signature,
	pubM []*Curve.BIG,
) (*BlindVerifyRequest, error) {
	protoSig, err := sig.ToProto()
	if err != nil {
		return nil, err
	}
	protoTheta, err := theta.ToProto()
	if err != nil {
		return nil, err
	}
	pubMb, err := coconut.BigSliceToByteSlices(pubM)
	if err != nil {
		return nil, err
	}
	return &BlindVerifyRequest{
		Theta: protoTheta,
		Sig:   protoSig,
		PubM:  pubMb,
	}, nil
}

// NewSpendCredentialRequest returns new instance of a SpendCredentialRequest
// given credential and the required cryptographic materials.
func NewSpendCredentialRequest(sig *coconut.Signature,
	pubM []*Curve.BIG,
	theta *coconut.ThetaTumbler,
	val int64,
	address ethcommon.Address,
) (*SpendCredentialRequest, error) {
	protoSig, err := sig.ToProto()
	if err != nil {
		return nil, err
	}

	pubMb, err := coconut.BigSliceToByteSlices(pubM)
	if err != nil {
		return nil, err
	}

	if len(pubM) == 0 || Curve.Comp(pubM[0], Curve.NewBIGint(int(val))) != 0 || val <= 0 {
		return nil, errors.New("invalid credential value")
	}

	protoThetaTumbler, err := theta.ToProto()
	if err != nil {
		return nil, err
	}

	// it is not checked whether the proof is actually bound to the provided address,
	// if it's not, it will just fail verification.
	// Also some providers might not require it, so nil is also a valid value.
	return &SpendCredentialRequest{
		Sig:             protoSig,
		PubM:            pubMb,
		Theta:           protoThetaTumbler,
		Value:           val,
		MerchantAddress: address[:],
	}, nil
}

// NewLookUpCredentialRequest returns new instance of a LookUpCredentialRequest
// given height of the desired block and public ElGamal key used during the blind issuance.
func NewLookUpCredentialRequest(height int64, egPub *elgamal.PublicKey) (*LookUpCredentialRequest, error) {
	gammaB := make([]byte, constants.ECPLen)
	egPub.Gamma().ToBytes(gammaB, true)

	return &LookUpCredentialRequest{
		Height: height,
		Gamma:  gammaB,
	}, nil
}

// NewLookUpBlockCredentialsRequest returns new instance of a LookUpBlockCredentialsRequest
// given height of the desired block.
func NewLookUpBlockCredentialsRequest(height int64) (*LookUpBlockCredentialsRequest, error) {
	return &LookUpBlockCredentialsRequest{
		Height: height,
	}, nil
}
