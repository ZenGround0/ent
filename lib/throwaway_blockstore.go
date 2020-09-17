package lib

import (
	"context"

	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"golang.org/x/xerrors"
)

type ThrowawayBlockstore struct{}

func NewThrowawayBlockstore() *ThrowawayBlockstore {
	return &ThrowawayBlockstore{}
}

func (tb *ThrowawayBlockstore) DeleteBlock(c cid.Cid) error {
	return nil
}

func (tb *ThrowawayBlockstore) Has(c cid.Cid) (bool, error) {
	return false, nil
}

func (tb *ThrowawayBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	return nil, blockstore.ErrNotFound
}

func (tb *ThrowawayBlockstore) GetSize(c cid.Cid) (int, error) {
	return 0, blockstore.ErrNotFound
}

// Don't error just do nothing
func (tb *ThrowawayBlockstore) Put(b blocks.Block) error {
	return nil
}

func (tb *ThrowawayBlockstore) PutMany(bs []blocks.Block) error {
	return nil
}

func (tb *ThrowawayBlockstore) AllKeysChan(_ context.Context) (<-chan cid.Cid, error) {
	// shouldn't call this
	return nil, xerrors.Errorf("throwaway block store doesn't support operation")
}

// noop=
func (tb *ThrowawayBlockstore) HashOnRead(enabled bool) {
	return
}
