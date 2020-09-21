package lib

import (
	"context"

	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs-blockstore"
	"golang.org/x/xerrors"
)

// RedirectBlockstore pushes all writes to one blockstore and reads from both.
//
// The motivation is to read from a large immutable blockstore while writing
// new data to a lightweight store such that data in the lightweight store
// can still point to data in the immutable store.
type RedirectBlockstore struct {
	readWriteStore blockstore.Blockstore
	readStore      blockstore.Blockstore
}

func NewRedirectBlockstore(readWriteStore, readStore blockstore.Blockstore) *RedirectBlockstore {
	return &RedirectBlockstore{
		readWriteStore: readWriteStore,
		readStore:      readStore,
	}
}

func (rb *RedirectBlockstore) DeleteBlock(c cid.Cid) error {
	return xerrors.Errorf("redirect block store can't delete blocks")
}

func (rb *RedirectBlockstore) Has(c cid.Cid) (bool, error) {
	rwHas, err := rb.readWriteStore.Has(c)
	if err != nil {
		return false, err
	}
	if rwHas {
		return true, err
	}
	rHas, err := rb.readStore.Has(c)
	if err != nil {
		return false, err
	}
	return rHas, nil
}

func (rb *RedirectBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	b, err := rb.readWriteStore.Get(c)
	if err == nil {
		return b, nil
	}
	if err != blockstore.ErrNotFound {
		return nil, err
	}
	return rb.readStore.Get(c)
}

func (rb *RedirectBlockstore) GetSize(c cid.Cid) (int, error) {
	s, err := rb.readWriteStore.GetSize(c)
	if err == nil {
		return s, nil
	}
	if err != blockstore.ErrNotFound {
		return 0, err
	}
	return rb.readStore.GetSize(c)
}

func (rb *RedirectBlockstore) Put(b blocks.Block) error {
	return rb.readWriteStore.Put(b)
}

func (rb *RedirectBlockstore) PutMany(bs []blocks.Block) error {
	return rb.readWriteStore.PutMany(bs)
}

func (rb *RedirectBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	// shouldn't call this
	return nil, xerrors.Errorf("redirect block store doesn't support operation")
}

func (rb *RedirectBlockstore) HashOnRead(enabled bool) {
	rb.readWriteStore.HashOnRead(enabled)
	rb.readStore.HashOnRead(enabled)
}
