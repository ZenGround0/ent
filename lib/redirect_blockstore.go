package lib

import (
	"context"

	lbstore "github.com/filecoin-project/lotus/lib/blockstore"
	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs-blockstore"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"
)

// BufferedBlockstore pushes all writes to an in memory cache blockstore and reads
// through this cache into a slower lotus blockstore backed by an on disk repo.
// It has an extra method for flushing data from the in memory blockstore to a
// third on disk badger datastore backed blockstore.
type BufferedBlockstore struct {
	buffer blockstore.Blockstore
	read   blockstore.Blockstore
	write  blockstore.Blockstore
}

func NewBufferedBlockstore(readLotusPath, writeEntPath string) (*BufferedBlockstore, error) {
	// load lotus chain datastore
	lotusExpPath, err := homedir.Expand(lotusPath)
	if err != nil {
		return nil, err
	}
	lotusDS, err := chainBadgerDs(lotusExpPath)
	if err != nil {
		return nil, err
	}
	entExpPath, err := homedir.Expand(entPath)
	if err != nil {
		return nil, err
	}
	entDS, err := chainBadgerDs(entExpPath)
	if err != nil {
		return nil, err
	}

	return &BufferedBlockstore{
		buffer: lbstore.NewTemporarySync(),
		read:   blockstore.NewBlockstore(lotusDS),
		write:  blockstore.NewBlockstore(entDS),
	}, nil
}

func (rb *BufferedBlockstore) DeleteBlock(c cid.Cid) error {
	return xerrors.Errorf("buffered block store can't delete blocks")
}

func (rb *BufferedBlockstore) Has(c cid.Cid) (bool, error) {
	rwHas, err := rb.buffer.Has(c)
	if err != nil {
		return false, err
	}
	if rwHas {
		return true, err
	}
	rHas, err := rb.read.Has(c)
	if err != nil {
		return false, err
	}
	return rHas, nil
}

func (rb *BufferedBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	b, err := rb.buffer.Get(c)
	if err == nil {
		return b, nil
	}
	if err != blockstore.ErrNotFound {
		return nil, err
	}
	return rb.read.Get(c)
}

func (rb *BufferedBlockstore) GetSize(c cid.Cid) (int, error) {
	s, err := rb.buffer.GetSize(c)
	if err == nil {
		return s, nil
	}
	if err != blockstore.ErrNotFound {
		return 0, err
	}
	return rb.read.GetSize(c)
}

func (rb *BufferedBlockstore) Put(b blocks.Block) error {
	return rb.buffer.Put(b)
}

func (rb *BufferedBlockstore) PutMany(bs []blocks.Block) error {
	return rb.buffer.PutMany(bs)
}

func (rb *BufferedBlockstore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	// shouldn't call this
	return nil, xerrors.Errorf("redirect block store doesn't support operation")
}

func (rb *BufferedBlockstore) HashOnRead(enabled bool) {
	rb.buffer.HashOnRead(enabled)
	rb.read.HashOnRead(enabled)
}
