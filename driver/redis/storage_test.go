package redis

import (
	"context"
	"github.com/go-redis/redis/v7"
	"github.com/spiral/kv"
	"github.com/stretchr/testify/assert"
	"strconv"
	"sync"
	"testing"
	"time"
)

func initStorage() kv.Storage {
	opt := &redis.UniversalOptions{Addrs: []string{"localhost:6379"}}
	return NewRedisClient(opt)
}

func cleanup(t *testing.T, s kv.Storage, keys ...string) {
	err := s.Delete(context.Background(), keys...)
	if err != nil {
		t.Fatalf("error during cleanup: %s", err.Error())
	}
}

func TestStorage_Has(t *testing.T) {
	s := initStorage()
	defer func() {
		cleanup(t, s, "key")
		if err := s.Close(); err != nil {
			panic(err)
		}
	}()

	v, err := s.Has(context.Background(), "key")
	assert.NoError(t, err)
	assert.False(t, v["key"])
}

func TestNilAndWrongArgs(t *testing.T) {
	s := initStorage()
	ctx := context.Background()
	defer func() {
		cleanup(t, s, "key")
		if err := s.Close(); err != nil {
			panic(err)
		}
	}()

	// check
	v, err := s.Has(ctx, "key")
	assert.NoError(t, err)
	assert.False(t, v["key"])

	_, err = s.Has(ctx, "")
	assert.Error(t, err)

	_, err = s.Get(ctx, "")
	assert.Error(t, err)

	_, err = s.Get(ctx, " ")
	assert.Error(t, err)

	_, err = s.Get(ctx, "                 ")
	assert.Error(t, err)

	_, err = s.MGet(ctx, "key", "key2", "")
	assert.Error(t, err)

	_, err = s.MGet(ctx, "key", "key2", "   ")
	assert.Error(t, err)

	assert.Error(t, s.Set(ctx, kv.Item{}))

	assert.Error(t, s.Set(ctx, kv.Item{
		Key:   "key",
		Value: "hello world",
		TTL:   "l",
	}))

	_, err = s.Has(ctx, "key")
	assert.NoError(t, err)

	err = s.Delete(ctx, "")
	assert.Error(t, err)

	err = s.Delete(ctx, "key", "")
	assert.Error(t, err)

	err = s.Delete(ctx, "key", "     ")
	assert.Error(t, err)

	err = s.Delete(ctx, "key")
	assert.NoError(t, err)
}

func TestStorage_Set_Get_Delete(t *testing.T) {
	s := initStorage()
	ctx := context.Background()
	defer func() {
		cleanup(t, s, "key")
		if err := s.Close(); err != nil {
			panic(err)
		}
	}()

	v, err := s.Has(ctx, "key")
	assert.NoError(t, err)
	assert.False(t, v["key"])

	assert.NoError(t, s.Set(ctx, kv.Item{
		Key:   "key",
		Value: "hello world",
		TTL:   "",
	}))

	v, err = s.Has(ctx, "key")
	assert.NoError(t, err)
	assert.True(t, v["key"])

	buf, err := s.Get(ctx, "key")
	assert.NoError(t, err)
	// be careful here, unsafe convert
	assert.Equal(t, "hello world", string(buf))

	assert.NoError(t, s.Delete(ctx, "key"))

	v, err = s.Has(ctx, "key")
	assert.NoError(t, err)
	assert.False(t, v["key"])
}

func TestStorage_Set_GetM(t *testing.T) {
	s := initStorage()
	ctx := context.Background()

	defer func() {
		cleanup(t, s, "key", "key2")
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	v, err := s.Has(ctx, "key")
	assert.NoError(t, err)
	assert.False(t, v["key"])

	assert.NoError(t, s.Set(ctx, kv.Item{
		Key:   "key",
		Value: "hello world",
		TTL:   "",
	}, kv.Item{
		Key:   "key2",
		Value: "hello world",
		TTL:   "",
	}))

	res, err := s.MGet(ctx, "key", "key2")
	assert.NoError(t, err)
	assert.Len(t, res, 2)
}

func TestStorage_SetExpire_TTL(t *testing.T) {
	s := initStorage()
	ctx := context.Background()
	defer func() {
		cleanup(t, s, "key", "key2")
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// ensure that storage is clean
	v, err := s.Has(ctx, "key", "key2")
	assert.NoError(t, err)
	assert.False(t, v["key"])
	assert.False(t, v["key2"])

	assert.NoError(t, s.Set(ctx, kv.Item{
		Key:   "key",
		Value: "hello world",
		TTL:   "",
	},
		kv.Item{
			Key:   "key2",
			Value: "hello world",
			TTL:   "",
		}))

	nowPlusFive := time.Now().Add(time.Second * 5).Format(time.RFC3339)

	// set timeout to 5 sec
	assert.NoError(t, s.Set(ctx, kv.Item{
		Key:   "key",
		Value: "value",
		TTL:   nowPlusFive,
	},
		kv.Item{
			Key:   "key2",
			Value: "value",
			TTL:   nowPlusFive,
		}))

	time.Sleep(time.Second * 2)
	m, err := s.TTL(ctx, "key", "key2")
	assert.NoError(t, err)
	keyTTL := m["key"].(float64)
	key2TTL := m["key2"].(float64)

	assert.True(t, keyTTL < float64(5))
	assert.True(t, keyTTL > float64(1))

	assert.True(t, key2TTL < float64(5))
	assert.True(t, key2TTL > float64(1))
}

func TestStorage_MExpire_TTL(t *testing.T) {
	s := initStorage()
	ctx := context.Background()
	defer func() {
		cleanup(t, s, "key", "key2")
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// ensure that storage is clean
	v, err := s.Has(ctx, "key", "key2")
	assert.NoError(t, err)
	assert.False(t, v["key"])
	assert.False(t, v["key2"])

	assert.NoError(t, s.Set(ctx, kv.Item{
		Key:   "key",
		Value: "hello world",
		TTL:   "",
	},
		kv.Item{
			Key:   "key2",
			Value: "hello world",
			TTL:   "",
		}))
	// set timeout to 5 sec
	nowPlusFive := time.Now().Add(time.Second * 5).Format(time.RFC3339)

	i1 := kv.Item{
		Key:   "key",
		Value: "",
		TTL:   nowPlusFive,
	}
	i2 := kv.Item{
		Key:   "key2",
		Value: "",
		TTL:   nowPlusFive,
	}
	assert.NoError(t, s.MExpire(ctx, i1, i2))

	time.Sleep(time.Second * 2)
	m, err := s.TTL(ctx, "key", "key2")
	assert.NoError(t, err)
	keyTTL := m["key"].(float64)
	key2TTL := m["key2"].(float64)

	assert.True(t, keyTTL < float64(5))
	assert.True(t, keyTTL > float64(1))

	assert.True(t, key2TTL < float64(5))
	assert.True(t, key2TTL > float64(1))
}

// not very fair test
func TestConcurrentReadWriteTransactions(t *testing.T) {
	s := initStorage()
	defer func() {
		cleanup(t, s, "key", "key2")
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	ctx := context.Background()
	v, err := s.Has(ctx, "key")
	assert.NoError(t, err)
	// no such key
	assert.False(t, v["key"])

	assert.NoError(t, s.Set(ctx, kv.Item{
		Key:   "key",
		Value: "hello world",
		TTL:   "",
	}, kv.Item{
		Key:   "key2",
		Value: "hello world",
		TTL:   "",
	}))

	v, err = s.Has(ctx, "key", "key2")
	assert.NoError(t, err)
	// no such key
	assert.True(t, v["key"])
	assert.True(t, v["key2"])

	wg := &sync.WaitGroup{}
	wg.Add(3)

	m := &sync.RWMutex{}
	// concurrently set the keys
	go func(s kv.Storage) {
		defer wg.Done()
		for i := 0; i <= 1000; i++ {
			m.Lock()
			// set is writable transaction
			// it should stop readable
			assert.NoError(t, s.Set(ctx, kv.Item{
				Key:   "key" + strconv.Itoa(i),
				Value: "hello world" + strconv.Itoa(i),
				TTL:   "",
			}, kv.Item{
				Key:   "key2" + strconv.Itoa(i),
				Value: "hello world" + strconv.Itoa(i),
				TTL:   "",
			}))
			m.Unlock()
		}
	}(s)

	// should be no errors
	go func(s kv.Storage) {
		defer wg.Done()
		for i := 0; i <= 1000; i++ {
			m.RLock()
			v, err = s.Has(ctx, "key")
			assert.NoError(t, err)
			// no such key
			assert.True(t, v["key"])
			m.RUnlock()
		}
	}(s)

	// should be no errors
	go func(s kv.Storage) {
		defer wg.Done()
		for i := 0; i <= 1000; i++ {
			m.Lock()
			err = s.Delete(ctx, "key"+strconv.Itoa(i))
			assert.NoError(t, err)
			m.Unlock()
		}
	}(s)

	wg.Wait()
}
