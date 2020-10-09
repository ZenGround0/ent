package lib

import (
	"context"

	dgbadger "github.com/dgraph-io/badger/v2"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	cid "github.com/ipfs/go-cid"
	datastore "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	"github.com/ipfs/go-ipfs-blockstore"
	"github.com/ipfs/go-ipld-cbor"
	"golang.org/x/xerrors"
)

var lotusPath = "~/.lotus/datastore/chain"

// currently unused but can be used for persisting data
var entPath = "~/.ent/datastore/chain"

type Chain struct {
	cachedBs *BufferedBlockstore
}

// Lifted from lotus/node/repo/fsrepo_ds.go
func chainBadgerDs(path string) (datastore.Batching, error) {
	opts := badger.DefaultOptions
	opts.GcInterval = 0 // disable GC for chain datastore

	opts.Options = dgbadger.DefaultOptions("").WithTruncate(true).
		WithValueThreshold(1 << 10)

	return badger.NewDatastore(path, &opts)
}

func (c *Chain) loadBufferedBstore(ctx context.Context) (*BufferedBlockstore, error) {
	if c.cachedBs != nil {
		return c.cachedBs, nil
	}
	var err error
	c.cachedBs, err = NewBufferedBlockstore(lotusPath, entPath)
	return c.cachedBs, err
}

// LoadCborStore loads the ~/.lotus chain datastore for chain traversal and state loading
func (c *Chain) LoadCborStore(ctx context.Context) (cbornode.IpldStore, error) {
	bs, err := c.loadBufferedBstore(ctx)
	if err != nil {
		return nil, err
	}
	return cbornode.NewCborStore(bs), nil
}

func (c *Chain) LoadToReadOnlyBuffer(ctx context.Context, stateRoot cid.Cid) error {
	bs, err := c.loadBufferedBstore(ctx)
	if err != nil {
		return err
	}
	return bs.LoadToReadOnlyBuffer(ctx, stateRoot)
}

func (c *Chain) FlushBufferedState(ctx context.Context, stateRoot cid.Cid) error {
	bs, err := c.loadBufferedBstore(ctx)
	if err != nil {
		return err
	}
	return bs.FlushFromBuffer(ctx, stateRoot)
}

// ChainStateIterator moves from tip to genesis emiting parent state roots of blocks
type ChainStateIterator struct {
	bs         blockstore.Blockstore
	currBlock  *types.BlockHeader
	currParent *types.BlockHeader
}

type IterVal struct {
	Height int64
	State  cid.Cid
}

func (c *Chain) NewChainStateIterator(ctx context.Context, tipCid cid.Cid) (*ChainStateIterator, error) {
	bs, err := c.loadBufferedBstore(ctx)
	if err != nil {
		return nil, err
	}
	// get starting block
	raw, err := bs.Get(tipCid)
	if err != nil {
		return nil, err
	}

	blk, err := types.DecodeBlock(raw.RawData())
	if err != nil {
		return nil, err
	}
	parent, err := getParent(blk, bs)
	if err != nil {
		return nil, err
	}

	return &ChainStateIterator{
		currBlock:  blk,
		currParent: parent,
		bs:         bs,
	}, nil
}

func (it *ChainStateIterator) Done() bool {
	if it.currParent.Height == abi.ChainEpoch(0) {
		return true
	}
	return false
}

// Return the parent state root and parent height of the current block
func (it *ChainStateIterator) Val() IterVal {
	return IterVal{
		State:  it.currBlock.ParentStateRoot,
		Height: int64(it.currParent.Height),
	}
}

// Moves iterator backwards towards genesis.  Noop at genesis
func (it *ChainStateIterator) Step(ctx context.Context) error {
	if it.Done() { // noop
		return nil
	}
	var err error
	parent := it.currParent
	it.currParent, err = getParent(it.currParent, it.bs)
	if err != nil {
		return err
	}
	it.currBlock = parent
	return err
}

func getParent(blk *types.BlockHeader, bs blockstore.Blockstore) (*types.BlockHeader, error) {
	parents := blk.Parents
	if len(parents) == 0 {
		return nil, xerrors.Errorf("cannot get parent from input blk, is it genesis?")
	}
	// we don't care which, take the first one
	raw, err := bs.Get(parents[0])
	if err != nil {
		return nil, err
	}
	return types.DecodeBlock(raw.RawData())
}
