package memory

import (
	"context"
	"errors"
	"github.com/spiral/kv"
	"strings"
	"sync"
	"time"
)

type Storage struct {
	heap *sync.Map //map[string]item
	stop chan struct{}

	// config for RR integration
	cfg *Config

	// wait group is used to prevent Serve for early exit
	// is used together with stop
	// BUT NOT USED IN GENERAL API, ONLY FOR RoadRunner
	wg *sync.WaitGroup
}

func NewInMemoryStorage() kv.Storage {
	ttls := &Storage{
		heap: &sync.Map{},
		stop: make(chan struct{}),
	}

	go ttls.gcPhase()

	return ttls
}

func (s *Storage) Init(config *Config) (bool, error) {
	if config == nil {
		return false, kv.ErrNoConfig
	}
	s.cfg = config
	return true, nil
}

func (s Storage) Serve() error {
	if !s.cfg.Enabled {
		return nil
	}

	// init the wait group to prevent Serve to exit early, before RR called Stop
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// init in-memory
	s.heap = &sync.Map{}
	s.stop = make(chan struct{})

	// start in-memory gc for kv
	go s.gcPhase()

	wg.Wait()
	return nil
}

func (s Storage) Stop() {
	defer s.wg.Done()
	err := s.Close()
	if err != nil {
		s.cfg.log.Error("error during the stopping in-memory storage", err)
	}
}

func (s Storage) Has(ctx context.Context, keys ...string) (map[string]bool, error) {
	if keys == nil {
		return nil, kv.ErrNoKeys
	}
	m := make(map[string]bool)
	for _, key := range keys {

		keyTrimmed := strings.TrimSpace(key)
		if keyTrimmed == "" {
			return nil, kv.ErrEmptyKey
		}

		if _, ok := s.heap.Load(key); ok {
			m[key] = true
		} else {
			m[key] = false
		}
	}

	return m, nil
}

func (s Storage) Get(ctx context.Context, key string) ([]byte, error) {
	// to get cases like "  "
	keyTrimmed := strings.TrimSpace(key)
	if keyTrimmed == "" {
		return nil, kv.ErrEmptyKey
	}

	if data, exist := s.heap.Load(key); exist {
		// here might be a panic
		// but data only could be a string, see Set function
		return []byte(data.(kv.Item).Value), nil
	}
	return nil, nil
}

func (s Storage) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	if keys == nil {
		return nil, kv.ErrNoKeys
	}

	// should not be empty keys
	for _, key := range keys {
		keyTrimmed := strings.TrimSpace(key)
		if keyTrimmed == "" {
			return nil, kv.ErrEmptyKey
		}
	}

	m := make([]interface{}, 0, len(keys))

	for _, key := range keys {
		if value, ok := s.heap.Load(key); ok {
			m = append(m, value)
		}
	}

	return m, nil
}

func (s Storage) Set(ctx context.Context, items ...kv.Item) error {
	if items == nil {
		return kv.ErrNoKeys
	}
	for _, kvItem := range items {
		if kvItem.TTL > 0 {
			// heavy operation, but simple
			s.heap.Store(kvItem.Key, kv.Item{
				Key:   kvItem.Key,
				Value: kvItem.Value,
				TTL:   int(time.Unix(time.Now().Add(time.Second*time.Duration(kvItem.TTL)).Unix(), 0).Unix()),
			})
			continue
		}
		// heavy operation, but simple
		s.heap.Store(kvItem.Key, kv.Item{
			Key:   kvItem.Key,
			Value: kvItem.Value,
			TTL:   0,
		})
	}
	return nil
}

// MExpire sets the expiration time to the key
// If key already has the expiration time, it will be overwritten
func (s Storage) MExpire(ctx context.Context, timeout int, keys ...string) error {
	if timeout == 0 || keys == nil {
		return errors.New("should set timeout and at least one key")
	}

	ut := time.Unix(time.Now().Add(time.Second*time.Duration(timeout)).Unix(), 0).Unix()
	for _, key := range keys {
		// if key exist, overwrite it value
		if item, ok := s.heap.Load(key); ok {
			kvItem := item.(kv.Item)
			kvItem.TTL = int(ut)

			s.heap.Store(key, kvItem)
		}

	}
	return nil
}

func (s Storage) TTL(ctx context.Context, keys ...string) (map[string]interface{}, error) {
	if keys == nil {
		return nil, kv.ErrNoKeys
	}

	// should not be empty keys
	for _, key := range keys {
		keyTrimmed := strings.TrimSpace(key)
		if keyTrimmed == "" {
			return nil, kv.ErrEmptyKey
		}
	}

	m := make(map[string]interface{}, len(keys))

	for _, key := range keys {
		if item, ok := s.heap.Load(key); ok {
			m[key] = item.(kv.Item).TTL
		}
	}
	return m, nil
}

func (s Storage) Delete(ctx context.Context, keys ...string) error {
	if keys == nil {
		return kv.ErrNoKeys
	}

	// should not be empty keys
	for _, key := range keys {
		keyTrimmed := strings.TrimSpace(key)
		if keyTrimmed == "" {
			return kv.ErrEmptyKey
		}
	}

	for _, key := range keys {
		s.heap.Delete(key)
	}
	return nil
}

// Close clears the in-memory storage
func (s Storage) Close() error {
	s.heap = &sync.Map{}
	s.stop <- struct{}{}
	return nil
}

//================================== PRIVATE ======================================

func (s *Storage) gcPhase() {
	// TODO check
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-s.stop:
			ticker.Stop()
			return
		case now := <-ticker.C:
			// check every second
			s.heap.Range(func(key, value interface{}) bool {
				v := value.(kv.Item)
				if now.Unix() > time.Unix(int64(v.TTL), 0).Unix() {
					s.heap.Delete(key)
				}
				return true
			})
		}
	}

}