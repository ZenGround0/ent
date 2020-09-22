package lib

import (
	"context"

	lbstore "github.com/filecoin-project/lotus/lib/blockstore"
	block "github.com/ipfs/go-block-format"
	blocks "github.com/ipfs/go-block-format"
	cid "github.com/ipfs/go-cid"
	"github.com/ipfs/go-ipfs-blockstore"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/xerrors"

	lvm "github.com/filecoin-project/lotus/chain/vm"
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
	if has, err := rb.buffer.Has(c); err != nil {
		return false, err
	} else if has {
		return true, nil
	}
	if has, err := rb.read.Has(c); err != nil {
		return false, err
	} else if has {
		return true, nil
	}
	return rb.write.Has(c)
}

func (rb *BufferedBlockstore) Get(c cid.Cid) (blocks.Block, error) {
	if b, err := rb.buffer.Get(c); err == nil {
		return b, nil
	} else if err != blockstore.ErrNotFound {
		return nil, err
	}
	if b, err := rb.read.Get(c); err == nil {
		return b, nil
	} else if err != blockstore.ErrNotFound {
		return nil, err
	}
	return rb.write.Get(c)
}

func (rb *BufferedBlockstore) GetSize(c cid.Cid) (int, error) {
	if s, err := rb.buffer.GetSize(c); err == nil {
		return s, nil
	} else if err != blockstore.ErrNotFound {
		return 0, err
	}
	if s, err := rb.read.GetSize(c); err == nil {
		return s, nil
	} else if err != blockstore.ErrNotFound {
		return 0, err
	}
	return rb.write.GetSize(c)
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
	rb.write.HashOnRead(enabled)
}

func (rb *BufferedBlockstore) LoadToBuffer(c cid.Cid) error {
	return lvm.Copy(rb.read, rb.buffer, c)
}

func (rb *BufferedBlockstore) FlushFromBuffer(ctx context.Context, c cid.Cid) error {
	allCh, err := rb.buffer.AllKeysChan(ctx)
	if err != nil {
		return err
	}
	var batch []block.Block
	for c := range allCh {
		blk, err := rb.buffer.Get(c)
		if err != nil {
			return xerrors.Errorf("buffer get in flush", err)
		}
		batch = append(batch, blk)
		if len(batch) > 100 {
			if err := rb.write.PutMany(batch); err != nil {
				return xerrors.Errorf("batch put in flush: %w", err)
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := rb.write.PutMany(batch); err != nil {
			return xerrors.Errorf("batch put in flush: %w", err)
		}
	}
	return nil
}
