package lib

import (
	"context"
	"fmt"
	"os"

	address "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/actors/adt"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/v2/actors/states"
)

type SectorInfo struct {
	Sector *miner.SectorOnChainInfo
	Status string
}

const channelBufferSize = 100

// V0TreeMinerBalancse returns a map of every miner's balance info
// at the provided state tree.  It is used for displaying and validating miner
// info.
func ExportSectors(ctx context.Context, store adt.Store, actorsIn *states.Tree) (chan *SectorInfo, error) {
	out := make(chan *SectorInfo, channelBufferSize)

	go func() {
		defer close(out)

		err := actorsIn.ForEach(func(addr address.Address, a *states.Actor) error {
			if !a.Code.Equals(builtin.StorageMinerActorCodeID) {
				return nil
			}
			_, _ = fmt.Fprintf(os.Stderr, "Miner %v\n", addr)
			var st miner.State
			if err := store.Get(ctx, a.Head, &st); err != nil {
				return err
			}

			sectors, err := miner.LoadSectors(store, st.Sectors)
			if err != nil {
				return err
			}

			deadlines, err := st.LoadDeadlines(store)
			if err != nil {
				return err
			}
			if err = deadlines.ForEach(store, func(dlIdx uint64, dl *miner.Deadline) error {
				partitions, err := dl.PartitionsArray(store)
				if err != nil {
					return err
				}
				var partition miner.Partition
				if err = partitions.ForEach(&partition, func(i int64) error {
					unproven, err := partition.Unproven.AllMap(1 << 20)
					if err != nil {
						return err
					}
					faults, err := partition.Faults.AllMap(1 << 20)
					if err != nil {
						return err
					}
					recovering, err := partition.Recoveries.AllMap(1 << 20)
					if err != nil {
						return err
					}
					terminated, err := partition.Terminated.AllMap(1 << 20)
					if err != nil {
						return err
					}

					if err = partition.Sectors.ForEach(func(sno uint64) error {
						status := "active"
						if unproven[sno] {
							status = "unproven"
						} else if faults[sno] {
							status = "faulty"
						} else if recovering[sno] {
							status = "recovering"
						} else if terminated[sno] {
							status = "terminated"
						}
						sector, err := sectors.MustGet(abi.SectorNumber(sno))
						if err != nil {
							return nil
						}

						out <- &SectorInfo{
							Sector: sector,
							Status: status,
						}
						return nil
					}); err != nil {
						return err
					}
					return nil
				}); err != nil {
					return err
				}
				return nil
			}); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
	}()

	return out, nil
}
