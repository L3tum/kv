package kv

import (
	"context"

	"github.com/spiral/kv/buffer/data"
	"golang.org/x/sync/errgroup"
)

type RpcServer struct {
	svc *Service
}

type Data struct {
	Storage string   `json:"storage"`
	Keys    []string `json:"keys"`
	Timeout string   `json:"timeout"`
}

// data Data
func (r *RpcServer) Has(in []byte, res *map[string]bool) error {
	ctx := context.Background()
	dataRoot := data.GetRootAsData(in, 0)
	l := dataRoot.KeysLength()
	keys := make([]string, 0, l)

	for i := 0; i < l; i++ {
		// TODO make unsafe fast convert
		keys = append(keys, string(dataRoot.Keys(i)))
	}

	storage := string(dataRoot.Storage())

	ret, err := r.svc.Storages[storage].Has(ctx, keys...)
	if err != nil {
		return err
	}
	// fill the map
	*res = ret

	return nil
}

type SetData struct {
	Items    []Item   `json:"items"`
	Storages []string `json:"storages"`
}

// in SetData
func (r *RpcServer) Set(in []byte, ok *bool) error {
	ctx := context.Background()
	dataRoot := data.GetRootAsSetData(in, 0)

	items := make([]Item, 0, dataRoot.ItemsLength())
	it := &data.Item{}
	for i := 0; i < dataRoot.ItemsLength(); i++ {
		if !dataRoot.Items(it, i) {
			continue
		}

		itc := Item{
			Key:   string(it.Key()),
			Value: string(it.Value()),
			TTL:   string(it.Timeout()),
		}

		items[i] = itc
	}

	errg := &errgroup.Group{}
	for i := 0; i < dataRoot.StoragesLength(); i++ {
		errg.Go(func() error {
			err := r.svc.Storages[string(dataRoot.Storages(i))].Set(ctx, items...)
			if err != nil {
				return err
			}
			return nil
		})
	}

	return errg.Wait()
}

// in Data
func (r *RpcServer) Get(in []byte, res *[]byte) error {
	ctx := context.Background()
	dataRoot := data.GetRootAsData(in, 0)
	l := dataRoot.KeysLength()
	keys := make([]string, 0, l)

	for i := 0; i < l; i++ {
		// TODO make unsafe fast convert
		keys = append(keys, string(dataRoot.Keys(i)))
	}

	storage := string(dataRoot.Storage())

	ret, err := r.svc.Storages[storage].Get(ctx, keys[0])
	if err != nil {
		return err
	}
	// value by key
	*res = ret

	return nil
}

// in Data
func (r *RpcServer) MGet(in []byte, res *map[string]interface{}) error {
	ctx := context.Background()
	dataRoot := data.GetRootAsData(in, 0)
	l := dataRoot.KeysLength()
	keys := make([]string, 0, l)

	for i := 0; i < l; i++ {
		// TODO make unsafe fast convert
		keys = append(keys, string(dataRoot.Keys(i)))
	}

	storage := string(dataRoot.Storage())

	ret, err := r.svc.Storages[storage].MGet(ctx, keys...)
	if err != nil {
		return err
	}
	// return the map
	*res = ret

	return nil
}

// in Data
func (r *RpcServer) MExpire(in []byte, ok *bool) error {
	ctx := context.Background()
	dataRoot := data.GetRootAsData(in, 0)
	l := dataRoot.KeysLength()

	// when unmarshalling the keys, simultaneously, fill up the slice with items
	it := make([]Item, 0, l)

	for i := 0; i < l; i++ {
		it = append(it, Item{
			Key: string(dataRoot.Keys(i)),
			// we set up timeout on the keys, so, value here is redundant
			Value: "",
			TTL:   string(dataRoot.Timeout()),
		})
	}

	err := r.svc.Storages[string(dataRoot.Storage())].MExpire(ctx, it...)
	if err != nil {
		return err
	}
	// return the map
	*ok = true

	return nil
}

// in Data
func (r *RpcServer) TTL(in []byte, res *map[string]interface{}) error {
	ctx := context.Background()
	dataRoot := data.GetRootAsData(in, 0)
	l := dataRoot.KeysLength()
	keys := make([]string, 0, l)

	for i := 0; i < l; i++ {
		// TODO make unsafe fast convert
		keys = append(keys, string(dataRoot.Keys(i)))
	}

	storage := string(dataRoot.Storage())

	ret, err := r.svc.Storages[storage].TTL(ctx, keys...)
	if err != nil {
		return err
	}
	// return the map
	*res = ret

	return nil
}

// in Data
func (r *RpcServer) Delete(in []byte, ok *bool) error {
	ctx := context.Background()
	dataRoot := data.GetRootAsData(in, 0)
	l := dataRoot.KeysLength()
	keys := make([]string, 0, l)

	for i := 0; i < l; i++ {
		// TODO make unsafe fast convert
		keys = append(keys, string(dataRoot.Keys(i)))
	}

	storage := string(dataRoot.Storage())

	err := r.svc.Storages[storage].Delete(ctx, keys...)
	if err != nil {
		return err
	}
	// return true
	*ok = true

	return nil
}

// in string, storages
func (r *RpcServer) Close(storage string, ok *bool) error {
	err := r.svc.Storages[storage].Close()
	if err != nil {
		return err
	}
	// return true
	*ok = true

	return nil
}
