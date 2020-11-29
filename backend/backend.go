package backend

import (
	"fmt"
	"github.com/bytepowered/flux"
	"github.com/bytepowered/flux/ext"
)

var (
	ErrBackendResponseDecoderNotFound = &flux.StateError{
		StatusCode: flux.StatusServerError,
		ErrorCode:  flux.ErrorCodeGatewayInternal,
		Message:    "BACKEND:RESPONSE_DECODER:NOT_FOUND",
	}
)

func DoExchange(ctx flux.Context, exchange flux.Backend) *flux.StateError {
	endpoint := ctx.Endpoint()
	resp, err := exchange.Invoke(endpoint.Service, ctx)
	if err != nil {
		return err
	}
	// decode responseWriter
	decoder, ok := ext.LoadBackendResponseDecoder(endpoint.Service.RpcProto)
	if !ok {
		return ErrBackendResponseDecoderNotFound
	}
	if code, headers, body, err := decoder(ctx, resp); nil == err {
		ctx.Response().SetStatusCode(code)
		ctx.Response().SetHeaders(headers)
		ctx.Response().SetBody(body)
		return nil
	} else {
		return &flux.StateError{
			StatusCode: flux.StatusServerError,
			ErrorCode:  flux.ErrorCodeGatewayInternal,
			Message:    "BACKEND:DECODE_RESPONSE",
			Internal:   err,
		}
	}
}

// DoInvoke 执行后端服务，获取响应结果；
func DoInvoke(service flux.BackendService, ctx flux.Context) (interface{}, *flux.StateError) {
	backend, ok := ext.LoadBackend(service.RpcProto)
	if !ok {
		return nil, &flux.StateError{
			StatusCode: flux.StatusServerError,
			ErrorCode:  flux.ErrorCodeGatewayInternal,
			Message:    "GATEWAY:UNKNOWN_PROTOCOL",
			Internal:   fmt.Errorf("unknown protocol:%s", service.RpcProto),
		}
	}
	return backend.Invoke(service, ctx)
}
