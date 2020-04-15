package kv

import (
	"context"
)

// Driver provides the ability to init one or multiple storage partitions.
type Driver interface {
	// Init initialize storage based on provided arguments.
	Init(StorageConfig) (Storage, error)
}

// Item represents general storage item
type Item struct {
	// Key of item
	Key string
	// Value of item
	Value string
	// time to live in seconds
	// for memcached also supported unix timestamps
	TTL int
}
// Storage represents single abstract storage.
type Storage interface {
	// Has checks if value exists.
	Has(ctx context.Context, args ...string) (map[string]bool, error)

	// Get loads value content into a byte slice.
	Get(ctx context.Context, key string) ([]byte, error)

	// MGet loads content of multiple values
	// TODO []interface{} -> map[string]interface{}
	MGet(ctx context.Context, args ...string) ([]interface{}, error)

	// Set used to upload item to KV with TTL
	// 0 value in TTL means no TTL
	Set(ctx context.Context, items ...Item) error

	// MExpire sets the TTL for multiply keys
	MExpire(ctx context.Context, timeout int, keys ...string) error

	// TTL return the rest time to live for provided keys
	// Not supported for the memcached and boltdb
	TTL(ctx context.Context, keys ...string) (map[string]interface{}, error)

	// Delete one or multiple keys.
	Delete(ctx context.Context, args ...string) error

	// Close closes the storage and underlying resources.
	Close() error
}