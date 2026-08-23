package hotrod

import (
	"context"

	"infinispan.org/go-client/internal/connection"
	"infinispan.org/go-client/internal/operation"
)

func execute[T any](ctx context.Context, p *connection.Pool, op operation.Operation) (T, error) {
	result, err := p.Execute(ctx, op)
	if err != nil {
		var zero T
		return zero, err
	}
	return result.(T), nil
}

func executeOn[T any](ctx context.Context, p *connection.Pool, addr string, op operation.Operation) (T, error) {
	result, err := p.ExecuteOn(ctx, addr, op)
	if err != nil {
		var zero T
		return zero, err
	}
	return result.(T), nil
}

func executeWithAddr[T any](ctx context.Context, p *connection.Pool, op operation.Operation) (T, string, error) {
	result, addr, err := p.ExecuteWithAddr(ctx, op)
	if err != nil {
		var zero T
		return zero, "", err
	}
	return result.(T), addr, nil
}
