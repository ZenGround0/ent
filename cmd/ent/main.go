package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	migration0 "github.com/filecoin-project/specs-actors/actors/migration/nv3"
	adt0 "github.com/filecoin-project/specs-actors/actors/util/adt"
	migration2 "github.com/filecoin-project/specs-actors/v2/actors/migration"
	states2 "github.com/filecoin-project/specs-actors/v2/actors/states"
	cid "github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/zenground0/ent/lib"
)

var rootsCmd = &cli.Command{
	Name:        "roots",
	Description: "provide state tree root cids for migrating",
	Action:      runRootsCmd,
}

var migrateCmd = &cli.Command{
	Name:        "migrate",
	Description: "migrate a filecoin v1 state root to v2",
	Subcommands: []*cli.Command{
		{
			Name:   "one",
			Usage:  "migrate a single state tree",
			Action: runMigrateOneCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "preload"},
			},
		},
		{
			Name:   "chain",
			Usage:  "migrate all state trees from given chain head to genesis",
			Action: runMigrateChainCmd,
			Flags: []cli.Flag{
				&cli.IntFlag{Name: "skip", Aliases: []string{"k"}},
			},
		},
		{
			Name:   "v0",
			Usage:  "DEPRECATED run a v0 migration on the parent state of the provided header",
			Action: runMigrateV0Cmd,
		},
	},
}

var validateCmd = &cli.Command{
	Name:        "validate",
	Description: "validate a migration by checking lots of invariants",
	Action:      runValidateCmd,
}

var debtsCmd = &cli.Command{
	Name:        "debts",
	Description: "display all miner actors in debt and total burnt funds",
	Action:      runDebtsCmd,
}

func main() {
	// pprof server
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	app := &cli.App{
		Name:        "ent",
		Usage:       "Test filecoin state tree migrations by running them",
		Description: "Test filecoin state tree migrations by running them",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "cpuprofile",
				Usage: "run cpuprofile and write results to provided file path",
			},
		},
		Commands: []*cli.Command{
			migrateCmd,
			validateCmd,
			rootsCmd,
			debtsCmd,
		},
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	for _, c := range app.Commands {
		sort.Sort(cli.FlagsByName(c.Flags))
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func runMigrateOneCmd(c *cli.Context) error {
	if c.Args().Len() != 2 {
		return xerrors.Errorf("not enough args, need state root to migrate and height")
	}
	cleanUp, err := cpuProfile(c)
	if err != nil {
		return err
	}
	defer cleanUp()
	stateRootIn, err := cid.Decode(c.Args().First())
	if err != nil {
		return err
	}
	hRaw, err := strconv.Atoi(c.Args().Get(1))
	if err != nil {
		return err
	}
	height := abi.ChainEpoch(int64(hRaw))
	preloadStateRoot := cid.Undef
	preloadStr := c.String("preload")
	if preloadStr != "" {
		preloadStateRoot, err = cid.Decode(preloadStr)
		if err != nil {
			return err
		}
		fmt.Printf("successful preload :%s\n", preloadStateRoot)
	}

	chn := lib.Chain{}
	if !preloadStateRoot.Equals(cid.Undef) {
		fmt.Printf("start preload of %s\n", preloadStateRoot)
		loadStart := time.Now()
		err = chn.LoadToReadOnlyBuffer(c.Context, preloadStateRoot)
		loadDuration := time.Since(loadStart)
		if err != nil {
			return err
		}
		fmt.Printf("%s preload time: %v\n", stateRootIn, loadDuration)
	}
	store, err := chn.LoadCborStore(c.Context)
	if err != nil {
		return err
	}
	start := time.Now()
	stateRootOut, err := migration2.MigrateStateTree(c.Context, store, stateRootIn, height, migration2.DefaultConfig())
	duration := time.Since(start)
	if err != nil {
		return err
	}
	fmt.Printf("%s => %s -- %v\n", stateRootIn, stateRootOut, duration)
	writeStart := time.Now()
	if err := chn.FlushBufferedState(c.Context, stateRootOut); err != nil {
		return xerrors.Errorf("failed to flush state tree to disk: %w\n", err)
	}
	writeDuration := time.Since(writeStart)
	fmt.Printf("%s buffer flush time: %v\n", stateRootOut, writeDuration)
	return nil
}

func runMigrateChainCmd(c *cli.Context) error {
	if !c.Args().Present() {
		return xerrors.Errorf("not enough args, need chain head to migrate")
	}
	cleanUp, err := cpuProfile(c)
	if err != nil {
		return err
	}
	defer cleanUp()
	bcid, err := cid.Decode(c.Args().First())
	if err != nil {
		return err
	}
	chn := lib.Chain{}
	iter, err := chn.NewChainStateIterator(c.Context, bcid)
	if err != nil {
		return err
	}
	store, err := chn.LoadCborStore(c.Context)
	if err != nil {
		return err
	}
	k := c.Int("skip")
	for !iter.Done() {
		val := iter.Val()
		if k == 0 || val.Height%int64(k) == int64(0) { // skip every k epochs
			start := time.Now()
			// The migration operates on the parent state computed at epoch k and epoch k
			// In the case of > 1 null blocks this won't exactly match the state that lotus
			// migrates because we don't apply cron transitions first.
			height := val.Height - int64(1)
			stateRootOut, err := migration2.MigrateStateTree(c.Context, store, val.State, abi.ChainEpoch(height), migration2.DefaultConfig())
			duration := time.Since(start)
			if err != nil {
				fmt.Printf("%d -- %s => %s !! %v\n", val.Height, val.State, stateRootOut, err)
			} else {
				fmt.Printf("%d -- %s => %s -- %v\n", val.Height, val.State, stateRootOut, duration)
			}
			writeStart := time.Now()
			if err := chn.FlushBufferedState(c.Context, stateRootOut); err != nil {
				fmt.Printf("%s buffer flush failed: %s\n", err, stateRootOut, err)
			}
			writeDuration := time.Since(writeStart)
			fmt.Printf("%s buffer flush time: %v\n", stateRootOut, writeDuration)
		}

		if err := iter.Step(c.Context); err != nil {
			return err
		}
	}
	return nil
}

func runMigrateV0Cmd(c *cli.Context) error {
	if !c.Args().Present() {
		return xerrors.Errorf("not enough args, need header cid to migrate")
	}
	cleanUp, err := cpuProfile(c)
	if err != nil {
		return err
	}
	defer cleanUp()
	bcid, err := cid.Decode(c.Args().First())
	if err != nil {
		return err
	}
	chn := lib.Chain{}
	iter, err := chn.NewChainStateIterator(c.Context, bcid)
	if err != nil {
		return err
	}
	store, err := chn.LoadCborStore(c.Context)
	if err != nil {
		return err
	}

	for !iter.Done() {
		v := iter.Val()
		stateRootIn := v.State
		epoch := abi.ChainEpoch(v.Height)
		start := time.Now()
		stateRootOut, err := migration0.MigrateStateTree(c.Context, store, stateRootIn, epoch)
		duration := time.Since(start)
		if err != nil {
			return err
		}
		fmt.Printf("%d: %s => %s -- %v\n", v.Height, stateRootIn, stateRootOut, duration)
		writeStart := time.Now()
		if err := chn.FlushBufferedState(c.Context, stateRootOut); err != nil {
			return xerrors.Errorf("failed to flush state tree to disk: %w\n", err)
		}
		writeDuration := time.Since(writeStart)
		fmt.Printf("%s buffer flush time: %v\n", stateRootOut, writeDuration)

		if err := iter.Step(c.Context); err != nil {
			return err
		}
	}
	return nil
}

func runValidateCmd(c *cli.Context) error {
	if c.Args().Len() != 2 {
		return xerrors.Errorf("wrong numberof args, need state root to migrate and height")
	}
	cleanUp, err := cpuProfile(c)
	if err != nil {
		return err
	}
	defer cleanUp()

	stateRootIn, err := cid.Decode(c.Args().First())
	if err != nil {
		return err
	}
	hRaw, err := strconv.Atoi(c.Args().Get(1))
	if err != nil {
		return err
	}
	height := abi.ChainEpoch(int64(hRaw))
	chn := lib.Chain{}
	store, err := chn.LoadCborStore(c.Context)
	if err != nil {
		return err
	}

	start := time.Now()
	stateRootOut, err := migration2.MigrateStateTree(c.Context, store, stateRootIn, height, migration2.DefaultConfig())
	duration := time.Since(start)
	if err != nil {
		return err
	}

	fmt.Printf("Migration: %s => %s -- %v\n", stateRootIn, stateRootOut, duration)

	adtStore := adt0.WrapStore(c.Context, store)
	actorsOut, err := states2.LoadTree(adtStore, stateRootOut)
	if err != nil {
		return err
	}
	expectedBalance, err := migration2.InputTreeBalance(c.Context, store, stateRootIn)
	if err != nil {
		return err
	}
	start = time.Now()
	acc, err := states2.CheckStateInvariants(actorsOut, expectedBalance)
	duration = time.Since(start)
	if err != nil {
		return err
	}
	if acc.IsEmpty() {
		fmt.Printf("Validation: %s -- no errors -- %v\n", stateRootOut, duration)
	} else {
		fmt.Printf("Validation: %s -- with errors -- %v\n%s\n", stateRootOut, duration, strings.Join(acc.Messages(), "\n"))
	}

	return nil
}

func runRootsCmd(c *cli.Context) error {
	if c.Args().Len() < 2 {
		return xerrors.Errorf("not enough args, need chain tip and number of states to fetch")
	}

	bcid, err := cid.Decode(c.Args().First())
	if err != nil {
		return err
	}
	num, err := strconv.Atoi(c.Args().Get(1))
	if err != nil {
		return err
	}
	// Read roots from lotus datastore
	roots := make([]cid.Cid, num)
	chn := lib.Chain{}
	iter, err := chn.NewChainStateIterator(c.Context, bcid)
	if err != nil {
		return err
	}
	for i := 0; !iter.Done() && i < num; i++ {
		roots[i] = iter.Val().State
		if err := iter.Step(c.Context); err != nil {
			return err
		}
	}
	// Output roots
	for _, root := range roots {
		fmt.Printf("%s\n", root)
	}
	return nil
}

func runDebtsCmd(c *cli.Context) error {
	if !c.Args().Present() {
		return xerrors.Errorf("not enough args, need state root to migrate")
	}
	stateRootIn, err := cid.Decode(c.Args().First())
	if err != nil {
		return err
	}
	chn := lib.Chain{}
	store, err := chn.LoadCborStore(c.Context)
	if err != nil {
		return err
	}

	bf, err := migration2.InputTreeBurntFunds(c.Context, store, stateRootIn)
	if err != nil {
		return err
	}

	available, err := migration2.InputTreeMinerAvailableBalance(c.Context, store, stateRootIn)
	if err != nil {
		return err
	}
	// filter out positive balances
	totalDebt := big.Zero()
	for addr, balance := range available {
		if balance.LessThan(big.Zero()) {
			debt := balance.Neg()
			fmt.Printf("miner %s: %s\n", addr, debt)
			totalDebt = big.Add(totalDebt, debt)
		}
	}
	fmt.Printf("burnt funds balance: %s\n", bf)
	fmt.Printf("total debt:          %s\n", totalDebt)
	return nil
}

func cpuProfile(c *cli.Context) (func(), error) {
	val := c.String("cpuprofile")
	if val == "" { // flag not set do nothing and defer nothing
		return func() {}, nil
	}

	// val is output path of cpuprofile file
	f, err := os.Create(val)
	if err != nil {
		return nil, err
	}
	err = pprof.StartCPUProfile(f)
	if err != nil {
		return nil, err
	}

	return func() {
		pprof.StopCPUProfile()
		err := f.Close()
		if err != nil {
			fmt.Printf("failed to close cpuprofile file %s: %s\n", val, err)
		}
	}, nil
}
