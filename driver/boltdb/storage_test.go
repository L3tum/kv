package boltdb

import (
	"context"
	"github.com/spiral/kv"
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

func initStorage() kv.Storage {
	storage, err := NewBoltClient("rr.db", 0777, nil, "rr", time.Second)
	if err != nil {
		panic(err)
	}
	return storage
}

func cleanup(t *testing.T, path string) {
	err := os.RemoveAll(path)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStorage_Has(t *testing.T) {
	s := initStorage()
	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}
		cleanup(t, "rr.db")
	}()

	v, err := s.Has(context.Background(), "key")
	assert.NoError(t, err)
	assert.False(t, v["key"])
}

func TestStorage_Has_Set_Has(t *testing.T) {
	s := initStorage()
	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}
		cleanup(t, "rr.db")
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
}

func TestConcurrentReadWriteTransactions(t *testing.T) {
	s := initStorage()
	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}
		cleanup(t, "rr.db")
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
			err = s.Delete(ctx, "key" + strconv.Itoa(i))
			assert.NoError(t, err)
			m.Unlock()
		}
	}(s)

	wg.Wait()
}

func TestStorage_Has_Set_MGet(t *testing.T) {
	s := initStorage()
	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}
		cleanup(t, "rr.db")
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

	res, err := s.MGet(ctx, "key", "key2")
	assert.NoError(t, err)
	assert.Len(t, res, 2)
}

func TestStorage_Has_Set_Get(t *testing.T) {
	s := initStorage()
	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}
		cleanup(t, "rr.db")
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
		Value: "hello world2",
		TTL:   "",
	}))

	v, err = s.Has(ctx, "key", "key2")
	assert.NoError(t, err)
	// no such key
	assert.True(t, v["key"])
	assert.True(t, v["key2"])

	res, err := s.Get(ctx, "key")
	assert.NoError(t, err)

	if string(res) != "hello world" {
		t.Fatal("wrong value by key")
	}
}

func TestStorage_Set_Del_Get(t *testing.T) {
	s := initStorage()
	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}
		cleanup(t, "rr.db")
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

	// check that keys are present
	res, err := s.MGet(ctx, "key", "key2")
	assert.NoError(t, err)
	assert.Len(t, res, 2)

	assert.NoError(t, s.Delete(ctx, "key", "key2"))
	// check that keys are not present
	res, err = s.MGet(ctx, "key", "key2")
	assert.NoError(t, err)
	assert.Len(t, res, 0)
}

func TestStorage_Set_GetM(t *testing.T) {
	s := initStorage()
	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}
		cleanup(t, "rr.db")
	}()
	ctx := context.Background()

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

func TestNilAndWrongArgs(t *testing.T) {
	s := initStorage()
	ctx := context.Background()
	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}
		cleanup(t, "rr.db")
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

	assert.NoError(t, s.Set(ctx, kv.Item{
		Key:   "key",
		Value: "hello world",
		TTL:   "",
	}))

	assert.Error(t, s.Set(ctx, kv.Item{
		Key:   "key",
		Value: "hello world",
		TTL:   "asdf",
	}))

	_, err = s.Has(ctx, "key")
	assert.NoError(t, err)

	assert.Error(t, s.Set(ctx, kv.Item{}))

	err = s.Delete(ctx, "")
	assert.Error(t, err)

	err = s.Delete(ctx, "key", "")
	assert.Error(t, err)

	err = s.Delete(ctx, "key", "     ")
	assert.Error(t, err)

	err = s.Delete(ctx, "key")
	assert.NoError(t, err)
}

func TestStorage_MExpire_TTL(t *testing.T) {
	s := initStorage()
	ctx := context.Background()
	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}
		cleanup(t, "rr.db")
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

	time.Sleep(time.Second * 6)

	// ensure that storage is clean
	v, err = s.Has(ctx, "key", "key2")
	assert.NoError(t, err)
	assert.False(t, v["key"])
	assert.False(t, v["key2"])
}

func TestStorage_SetExpire_TTL(t *testing.T) {
	s := initStorage()
	ctx := context.Background()
	defer func() {
		if err := s.Close(); err != nil {
			panic(err)
		}
		cleanup(t, "rr.db")
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

	// remove a precision 4.02342342 -> 4
	keyTTL, err := strconv.Atoi(m["key"].(string)[0:1])
	if err != nil {
		t.Fatal(err)
	}

	// remove a precision 4.02342342 -> 4
	key2TTL, err := strconv.Atoi(m["key"].(string)[0:1])
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, keyTTL < 5)
	assert.True(t, key2TTL < 5)

	time.Sleep(time.Second * 4)

	// ensure that storage is clean
	v, err = s.Has(ctx, "key", "key2")
	assert.NoError(t, err)
	assert.False(t, v["key"])
	assert.False(t, v["key2"])
}

//func TestStorage_SetExpire_TTL(t *testing.T) {
//	s := initStorage()
//	ctx := context.Background()
//	defer func() {
//		if err := s.Close(); err != nil {
//			panic(err)
//		}
//		cleanup(t, "rr.db")
//	}()
//
//	// ensure that storage is clean
//	v, err := s.Has(ctx, "key", "key2")
//	assert.NoError(t, err)
//	assert.False(t, v["key"])
//	assert.False(t, v["key2"])
//
//	// set timeout to 5 sec
//	assert.NoError(t, s.Set(ctx, kv.Item{
//		Key:   "key",
//		Value: "value",
//		TTL:   5,
//	},
//		kv.Item{
//			Key:   "key2",
//			Value: "value",
//			TTL:   5,
//		}))
//
//	time.Sleep(time.Second * 2)
//	m, err := s.TTL(ctx, "key", "key2")
//	assert.NoError(t, err)
//
//	keyTTL := m["key"].(int64)
//	key2TTL := m["key2"].(int64)
//
//	tt := time.Now().Unix()
//
//	assert.True(t, keyTTL > tt)
//	assert.True(t, key2TTL > tt)
//
//	time.Sleep(time.Second * 9)
//
//	// ensure that storage is clean
//	v, err = s.Has(ctx, "key", "key2")
//	assert.NoError(t, err)
//	assert.False(t, v["key"])
//	assert.False(t, v["key2"])
//}

//func TestStorage_MExpire_TTL(t *testing.T) {
//	s := initStorage()
//	ctx := context.Background()
//	defer func() {
//		if err := s.Close(); err != nil {
//			panic(err)
//		}
//		cleanup(t, "rr.db")
//	}()
//
//	// ensure that storage is clean
//	v, err := s.Has(ctx, "key", "key2")
//	assert.NoError(t, err)
//	assert.False(t, v["key"])
//	assert.False(t, v["key2"])
//
//	assert.NoError(t, s.Set(ctx, kv.Item{
//		Key:   "key",
//		Value: "value",
//		TTL:   0,
//	},
//		kv.Item{
//			Key:   "key2",
//			Value: "value",
//			TTL:   0,
//		}))
//	// set timeout to 5 sec per key
//	assert.NoError(t, s.MExpire(ctx, 10, "key", "key2"))
//
//	time.Sleep(time.Second * 2)
//	m, err := s.TTL(ctx, "key", "key2")
//	assert.NoError(t, err)
//
//	keyTTL := m["key"].(int64)
//	key2TTL := m["key2"].(int64)
//
//	tt := time.Now().Unix()
//
//	assert.True(t, keyTTL > tt)
//	assert.True(t, key2TTL > tt)
//
//	time.Sleep(time.Second * 10)
//
//	// ensure that storage is clean
//	v, err = s.Has(ctx, "key", "key2")
//	assert.NoError(t, err)
//	assert.False(t, v["key"])
//	assert.False(t, v["key2"])
//}
