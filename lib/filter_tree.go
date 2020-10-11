package lib

import (
	"context"

	addr "github.com/filecoin-project/go-address"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	states0 "github.com/filecoin-project/specs-actors/actors/states"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	builtin "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/states"
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
)

func FilterTreeV0(ctx context.Context, store cbornode.IpldStore, stateRootIn cid.Cid, optionalAddrs ...addr.Address) (cid.Cid, error) {
	adtStore := adt.WrapStore(ctx, store)
	addrsToKeep := make(map[addr.Address]struct{})
	for _, a := range optionalAddrs {
		addrsToKeep[a] = struct{}{}
	}
	// Add all singletons
	addrsToKeep[builtin0.VerifiedRegistryActorAddr] = struct{}{}
	addrsToKeep[builtin0.StorageMarketActorAddr] = struct{}{}
	addrsToKeep[builtin0.StoragePowerActorAddr] = struct{}{}
	addrsToKeep[builtin0.RewardActorAddr] = struct{}{}
	addrsToKeep[builtin0.SystemActorAddr] = struct{}{}
	addrsToKeep[builtin0.CronActorAddr] = struct{}{}
	addrsToKeep[builtin0.InitActorAddr] = struct{}{}
	addrsToKeep[builtin0.BurntFundsActorAddr] = struct{}{}

	actorsIn, err := states0.LoadTree(adtStore, stateRootIn)
	if err != nil {
		return cid.Undef, err
	}
	stateRootOut, err := adt.MakeEmptyMap(adtStore).Root()
	if err != nil {
		return cid.Undef, err
	}
	actorsOut, err := states0.LoadTree(adtStore, stateRootOut)
	if err != nil {
		return cid.Undef, err
	}

	err = actorsIn.ForEach(func(a addr.Address, actorIn *states0.Actor) error {
		_, keep := addrsToKeep[a]
		if !keep {
			return nil
		}
		return actorsOut.SetActor(a, actorIn)
	})
	if err != nil {
		return cid.Undef, err
	}
	return actorsOut.Flush()
}

func FilterTreeV2(ctx context.Context, store cbornode.IpldStore, stateRootIn cid.Cid, optionalAddrs ...addr.Address) (cid.Cid, error) {
	adtStore := adt.WrapStore(ctx, store)
	addrsToKeep := make(map[addr.Address]struct{})
	for _, a := range optionalAddrs {
		addrsToKeep[a] = struct{}{}
	}
	// Add all singletons
	addrsToKeep[builtin.VerifiedRegistryActorAddr] = struct{}{}
	addrsToKeep[builtin.StorageMarketActorAddr] = struct{}{}
	addrsToKeep[builtin.StoragePowerActorAddr] = struct{}{}
	addrsToKeep[builtin.RewardActorAddr] = struct{}{}
	addrsToKeep[builtin.SystemActorAddr] = struct{}{}
	addrsToKeep[builtin.CronActorAddr] = struct{}{}
	addrsToKeep[builtin.InitActorAddr] = struct{}{}
	addrsToKeep[builtin.BurntFundsActorAddr] = struct{}{}

	actorsIn, err := states.LoadTree(adtStore, stateRootIn)
	if err != nil {
		return cid.Undef, err
	}
	stateRootOut, err := adt.MakeEmptyMap(adtStore).Root()
	if err != nil {
		return cid.Undef, err
	}
	actorsOut, err := states.LoadTree(adtStore, stateRootOut)
	if err != nil {
		return cid.Undef, err
	}

	err = actorsIn.ForEach(func(a addr.Address, actorIn *states.Actor) error {
		_, keep := addrsToKeep[a]
		if !keep {
			return nil
		}
		return actorsOut.SetActor(a, actorIn)
	})
	if err != nil {
		return cid.Undef, err
	}
	return actorsOut.Flush()
}
