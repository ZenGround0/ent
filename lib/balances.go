package lib

import (
	"context"

	address "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	builtin0 "github.com/filecoin-project/specs-actors/actors/builtin"
	miner0 "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	states0 "github.com/filecoin-project/specs-actors/actors/states"
	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
)

type BalanceInfo struct {
	Balance           abi.TokenAmount
	LockedFunds       abi.TokenAmount
	InitialPledge     abi.TokenAmount
	PreCommitDeposits abi.TokenAmount
}

// V0TreeMinerBalancse returns a map of every miner's balance info
// at the provided state tree.  It is used for displaying and validating miner
// info.
func V0TreeMinerBalances(ctx context.Context, store cbornode.IpldStore, stateRootIn cid.Cid) (map[address.Address]BalanceInfo, error) {
	adtStore := adt.WrapStore(ctx, store)
	actorsIn, err := states0.LoadTree(adtStore, stateRootIn)
	if err != nil {
		return nil, err
	}
	balances := make(map[address.Address]BalanceInfo)

	err = actorsIn.ForEach(func(addr address.Address, a *states0.Actor) error {
		if !a.Code.Equals(builtin0.StorageMinerActorCodeID) {
			return nil
		}
		var inState miner0.State
		if err := store.Get(ctx, a.Head, &inState); err != nil {
			return err
		}
		balance := BalanceInfo{
			Balance:           a.Balance,
			LockedFunds:       inState.LockedFunds,
			InitialPledge:     inState.InitialPledgeRequirement,
			PreCommitDeposits: inState.PreCommitDeposits,
		}
		balances[addr] = balance
		return nil
	})
	return balances, err
}
