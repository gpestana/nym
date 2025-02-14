// requesthandler.go - handlers for coconut requests.
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

// Package requesthandler contains functions that are used by issuing authorities and service providers
package requesthandler

import (
	"context"
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/nymtech/nym/common/comm/commands"
	coconut "github.com/nymtech/nym/crypto/coconut/scheme"
	"github.com/nymtech/nym/server/issuer/utils"
)

// TODO: perhaps if it's too expensive, replace reflect.Type with some string or even a byte?
type HandlerRegistry map[reflect.Type]ResolveRequestHandlerFunc

type ResolveRequestHandlerFunc func(context.Context, <-chan *commands.Response) proto.Message

func makeProtoStatus(code commands.StatusCode, message string) *commands.Status {
	return &commands.Status{
		Code:    int32(code),
		Message: message,
	}
}

func waitUntilResolved(ctx context.Context, resCh <-chan *commands.Response) (interface{}, *commands.Status) {
	var protoStatus *commands.Status
	var data interface{}

	select {
	case resp := <-resCh:
		if resp.Data != nil &&
			resp.ErrorMessage == commands.DefaultResponseErrorMessage &&
			resp.ErrorStatus == commands.DefaultResponseErrorStatusCode {
			resp.ErrorStatus = commands.StatusCode_OK
		}

		data = resp.Data
		protoStatus = makeProtoStatus(resp.ErrorStatus, resp.ErrorMessage)

	// TODO: a way to cancel the request because even though it timeouts, the worker is still working on it
	// (basically need to pass the same context when sending our request)
	case <-ctx.Done():
		protoStatus = makeProtoStatus(commands.StatusCode_REQUEST_TIMEOUT, "Request took too long to resolve.")
	}

	return data, protoStatus
}

func ResolveVerificationKeyRequestHandler(ctx context.Context, resCh <-chan *commands.Response) proto.Message {
	data, protoStatus := waitUntilResolved(ctx, resCh)

	var err error
	protoVk := &coconut.ProtoVerificationKey{}
	issuerID := int64(-1)
	if data != nil {
		tvk := data.(*coconut.ThresholdVerificationKey)
		protoVk, err = tvk.VerificationKey.ToProto()
		if err != nil {
			protoStatus = makeProtoStatus(commands.StatusCode_PROCESSING_ERROR, "Failed to marshal response.")
		}
		issuerID = tvk.ID()
	}
	return &commands.VerificationKeyResponse{
		Vk:       protoVk,
		IssuerID: issuerID,
		Status:   protoStatus,
	}
}

func ResolveSignRequestHandler(ctx context.Context, resCh <-chan *commands.Response) proto.Message {
	data, protoStatus := waitUntilResolved(ctx, resCh)

	var err error
	protoSig := &coconut.ProtoSignature{}
	issuerID := int64(-1)
	if data != nil {
		resolvedData := data.(utils.IssuedSignature)
		protoSig, err = resolvedData.Sig.(*coconut.Signature).ToProto()
		if err != nil {
			protoStatus = makeProtoStatus(commands.StatusCode_PROCESSING_ERROR, "Failed to marshal response.")
		}
		issuerID = resolvedData.IssuerID
	}
	return &commands.SignResponse{
		Sig:      protoSig,
		IssuerID: issuerID,
		Status:   protoStatus,
	}
}

func ResolveVerifyRequestHandler(ctx context.Context, resCh <-chan *commands.Response) proto.Message {
	data, protoStatus := waitUntilResolved(ctx, resCh)

	return &commands.VerifyResponse{
		IsValid: data.(bool),
		Status:  protoStatus,
	}
}

func ResolveBlindSignRequestHandler(ctx context.Context, resCh <-chan *commands.Response) proto.Message {
	data, protoStatus := waitUntilResolved(ctx, resCh)

	var err error
	protoBlindSig := &coconut.ProtoBlindedSignature{}
	issuerID := int64(-1)
	if data != nil {
		resolvedData := data.(utils.IssuedSignature)
		protoBlindSig, err = resolvedData.Sig.(*coconut.BlindedSignature).ToProto()
		if err != nil {
			protoStatus = makeProtoStatus(commands.StatusCode_PROCESSING_ERROR, "Failed to marshal response.")
		}
		issuerID = resolvedData.IssuerID
	}
	return &commands.BlindSignResponse{
		Sig:      protoBlindSig,
		IssuerID: issuerID,
		Status:   protoStatus,
	}
}

func ResolveBlindVerifyRequestHandler(ctx context.Context, resCh <-chan *commands.Response) proto.Message {
	data, protoStatus := waitUntilResolved(ctx, resCh)

	return &commands.BlindVerifyResponse{
		IsValid: data.(bool),
		Status:  protoStatus,
	}
}

func ResolveLookUpCredentialRequestHandler(ctx context.Context, resCh <-chan *commands.Response) proto.Message {
	data, protoStatus := waitUntilResolved(ctx, resCh)

	credPair := (*commands.CredentialPair)(nil)
	if data != nil {
		credPair = data.(*commands.CredentialPair)
	}
	return &commands.LookUpCredentialResponse{
		CredentialPair: credPair,
		Status:         protoStatus,
	}
}

func ResolveLookUpBlockCredentialsRequestHandler(ctx context.Context, resCh <-chan *commands.Response) proto.Message {
	data, protoStatus := waitUntilResolved(ctx, resCh)

	credPairs := ([]*commands.CredentialPair)(nil)
	if data != nil {
		credPairs = data.([]*commands.CredentialPair)
	}
	return &commands.LookUpBlockCredentialsResponse{
		Credentials: credPairs,
		Status:      protoStatus,
	}
}

func ResolveSpendCredentialRequestHandler(ctx context.Context, resCh <-chan *commands.Response) proto.Message {
	data, protoStatus := waitUntilResolved(ctx, resCh)
	status := false
	if data != nil {
		status = data.(bool)
	}
	return &commands.SpendCredentialResponse{
		WasSuccessful: status,
		Status:        protoStatus,
	}
}
