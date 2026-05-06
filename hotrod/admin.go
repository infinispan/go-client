package hotrod

import (
	"context"
	"fmt"

	"infinispan.org/go-client/internal/operation"
)

// CacheAdmin provides cache lifecycle operations.
type CacheAdmin struct {
	client *Client
}

// Administration returns a CacheAdmin for managing caches on the server.
func (c *Client) Administration() *CacheAdmin {
	return &CacheAdmin{client: c}
}

// CreateCache creates a cache with the given XML or JSON configuration.
// Returns an error if the cache already exists.
func (ca *CacheAdmin) CreateCache(ctx context.Context, name string, configuration string) error {
	return ca.execCacheTask(ctx, "@@cache@create", name, configuration, 0)
}

// GetOrCreateCache creates a cache if it does not already exist.
// If the cache exists, this is a no-op.
func (ca *CacheAdmin) GetOrCreateCache(ctx context.Context, name string, configuration string) error {
	return ca.execCacheTask(ctx, "@@cache@getorcreate", name, configuration, 0)
}

// RemoveCache removes a cache and all its data.
func (ca *CacheAdmin) RemoveCache(ctx context.Context, name string) error {
	_, err := ca.client.pool.Execute(ctx, &operation.ExecOp{
		TaskName: "@@cache@remove",
		Params: []operation.ExecParam{
			{Name: "name", Value: []byte(name)},
		},
	})
	return err
}

// CreateCacheWithFlags creates a cache with the given configuration and admin flags.
func (ca *CacheAdmin) CreateCacheWithFlags(ctx context.Context, name string, configuration string, flags AdminFlag) error {
	return ca.execCacheTask(ctx, "@@cache@create", name, configuration, flags)
}

func (ca *CacheAdmin) execCacheTask(ctx context.Context, task, name, configuration string, flags AdminFlag) error {
	params := []operation.ExecParam{
		{Name: "name", Value: []byte(name)},
	}
	if configuration != "" {
		params = append(params, operation.ExecParam{Name: "configuration", Value: []byte(configuration)})
	}
	if flags != 0 {
		params = append(params, operation.ExecParam{Name: "flags", Value: []byte(flags.String())})
	}

	_, err := ca.client.pool.Execute(ctx, &operation.ExecOp{
		TaskName: task,
		Params:   params,
	})
	if err != nil {
		return fmt.Errorf("%s %q: %w", task, name, err)
	}
	return nil
}

// AdminFlag controls cache creation behavior.
type AdminFlag int

const (
	AdminFlagPermanent AdminFlag = 1 << iota
	AdminFlagVolatile
)

// String returns the server-side name of the admin flag.
func (f AdminFlag) String() string {
	switch f {
	case AdminFlagPermanent:
		return "PERMANENT"
	case AdminFlagVolatile:
		return "VOLATILE"
	default:
		return ""
	}
}
