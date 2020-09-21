package lib

import (
	"context"

	dgbadger "github.com/dgraph-io/badger/v2"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	lvm "github.com/filecoin-project/lotus/chain/vm"
	cid "github.com/ipfs/go-cid"
	datastore "github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"
	"github.com/ipfs/go-ipfs-blockstore"
	"github.com/ipfs/go-ipld-cbor"
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

func (c *Chain) PreLoadStateTree(ctx context.Context, stateRoot cid.Cid) error {
	bs, err := c.loadBufferedBstore(ctx)
	if err != nil {
		return err
	}

	// Because of the underlying redirect blockstore structure
	// this will read from slow lotus datastore and write to fast
	// in memory ent datastore.
	return lvm.Copy(bs, bs, stateRoot)
}

func (c *Chain) FlushBufferedState(ctx context.Context, stateRoot cid.Cid) error {
	bs, err := c.loadBufferedBstore(ctx)
	if err != nil {
		return err
	}
	return bs.FlushFromBuffer(stateRoot)
}

// ChainStateIterator moves from tip to genesis emiting parent state roots of blocks
type ChainStateIterator struct {
	bs        blockstore.Blockstore
	currBlock *types.BlockHeader
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

	return &ChainStateIterator{
		currBlock: blk,
		bs:        bs,
	}, nil
}

func (it *ChainStateIterator) Done() bool {
	if it.currBlock.Height == abi.ChainEpoch(0) {
		return true
	}
	return false
}

// Return the parent state root of the current block
func (it *ChainStateIterator) Val() IterVal {
	return IterVal{
		State:  it.currBlock.ParentStateRoot,
		Height: int64(it.currBlock.Height),
	}
}

// Moves iterator backwards towards genesis.  Noop at genesis
func (it *ChainStateIterator) Step(ctx context.Context) error {
	if it.Done() { // noop
		return nil
	}
	parents := it.currBlock.Parents
	// we don't care which, take the first one
	raw, err := it.bs.Get(parents[0])
	if err != nil {
		return err
	}
	nextBlock, err := types.DecodeBlock(raw.RawData())
	if err != nil {
		return err
	}
	it.currBlock = nextBlock
	return nil
}
