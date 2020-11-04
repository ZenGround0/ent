package lib

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-hamt-ipld/v2"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	init2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/verifreg"
	states2 "github.com/filecoin-project/specs-actors/v2/actors/states"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	cbg "github.com/whyrusleeping/cbor-gen"

	"golang.org/x/xerrors"
)

func PrintHAMTSizes(ctx context.Context, store cbornode.IpldStore, tree *states2.Tree) error {
	// Init
	initActor, found, err := tree.GetActor(builtin.InitActorAddr)
	if !found {
		return xerrors.Errorf("init actor not found")
	}
	if err != nil {
		return err
	}
	var initState init2.State
	if err := store.Get(ctx, initActor.Head, &initState); err != nil {
		return err
	}
	if err := measureAndPrintHAMT(ctx, store, initState.AddressMap, "init.AddressMap"); err != nil {
		return err
	}

	// VerifReg
	verifRegActor, found, err := tree.GetActor(builtin.VerifiedRegistryActorAddr)
	if !found {
		return xerrors.Errorf("verified registry actor not found")
	}
	if err != nil {
		return err
	}
	var verifRegState verifreg.State
	if err := store.Get(ctx, verifRegActor.Head, &verifRegState); err != nil {
		return err
	}
	if err := measureAndPrintHAMT(ctx, store, verifRegState.Verifiers, "verifreg.Verifiers"); err != nil {
		return err
	}
	if err := measureAndPrintHAMT(ctx, store, verifRegState.VerifiedClients, "verifreg.VerifiedClients"); err != nil {
		return err
	}

	// Market
	marketActor, found, err := tree.GetActor(builtin.StorageMarketActorAddr)
	if !found {
		return xerrors.Errorf("market actor not found")
	}
	if err != nil {
		return err
	}
	var marketState market.State
	if err := store.Get(ctx, marketActor.Head, &marketState); err != nil {
		return err
	}
	if err := measureAndPrintHAMT(ctx, store, marketState.PendingProposals, "market.PendingProposals"); err != nil {
		return err
	}
	if err := measureAndPrintHAMT(ctx, store, marketState.EscrowTable, "market.EscrowTable"); err != nil {
		return err
	}
	if err := measureAndPrintHAMT(ctx, store, marketState.LockedTable, "market.LockedTable"); err != nil {
		return err
	}
	if err := measureAndPrintHAMT(ctx, store, marketState.DealOpsByEpoch, "market.DealOpsByEpoch"); err != nil {
		return err
	}

	// Power
	powerActor, found, err := tree.GetActor(builtin.StoragePowerActorAddr)
	if !found {
		return xerrors.Errorf("power actor not found")
	}
	if err != nil {
		return err
	}
	var powerState power.State
	err = store.Get(ctx, powerActor.Head, &powerState)
	if err != nil {
		return err
	}
	if err := measureAndPrintHAMT(ctx, store, powerState.CronEventQueue, "power.CronEventQueue"); err != nil {
		return err
	}
	if err := measureAndPrintHAMT(ctx, store, powerState.Claims, "power.Claims"); err != nil {
		return err
	}
	if powerState.ProofValidationBatch != nil {
		if err := measureAndPrintHAMT(ctx, store, *powerState.ProofValidationBatch, "power.ProofValidationBatch"); err != nil {
			return err
		}
	}

	return nil
}

func measureAndPrintHAMT(ctx context.Context, store cbornode.IpldStore, root cid.Cid, id string) error {
	var total int
	var avgDataSize float64
	var avgKeySize float64

	rootNode, err := hamt.LoadNode(ctx, store, root, adt.HamtOptions...)
	if err != nil {
		return err
	}

	err = rootNode.ForEach(ctx, func(k string, val interface{}) error {
		total++
		// cast value to cbg deferred
		d := val.(*cbg.Deferred)
		avgDataSize += float64(len(d.Raw))
		avgKeySize += float64(len([]byte(k)))
		return nil
	})
	if err != nil {
		return err
	}
	avgDataSize = avgDataSize / float64(total)
	avgKeySize = avgKeySize / float64(total)
	fmt.Printf("%s -- %d, %f, %f\n", id, total, avgDataSize, avgKeySize)
	return nil
}
