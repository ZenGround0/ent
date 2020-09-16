package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"

	"os"
	"sort"
	"strconv"
	"time"

	"github.com/filecoin-project/specs-actors/v2/actors/migration"
	cid "github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"github.com/zenground0/ent/lib"
	"golang.org/x/xerrors"
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
		},
		{
			Name:   "chain",
			Usage:  "migrate all state trees from given chain head to genesis",
			Action: runMigrateChainCmd,
		},
	},
}

var validateCmd = &cli.Command{
	Name:        "validate",
	Description: "validate a migration by checking lots of invariants",
	Action:      runValidateCmd,
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
		Commands: []*cli.Command{
			migrateCmd,
			validateCmd,
			rootsCmd,
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
	start := time.Now()
	stateRootOut, err := migration.MigrateStateTree(c.Context, store, stateRootIn)
	duration := time.Since(start)
	if err != nil {
		return err
	}
	fmt.Printf("%s => %s -- %v\n", stateRootIn, stateRootOut, duration)
	return nil
}

func runMigrateChainCmd(c *cli.Context) error {
	if !c.Args().Present() {
		return xerrors.Errorf("not enough args, need chain head to migrate")
	}
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
		val := iter.Val()
		start := time.Now()
		stateRootOut, err := migration.MigrateStateTree(c.Context, store, val.State)
		duration := time.Since(start)
		if err != nil {
			fmt.Printf("%d -- %s => %s !! %v\n", val.Height, val.State, stateRootOut, err)
		} else {
			fmt.Printf("%d -- %s => %s -- %v\n", val.Height, val.State, stateRootOut, duration)
		}

		if err := iter.Step(c.Context); err != nil {
			return err
		}
	}
	return nil
}

func runValidateCmd(c *cli.Context) error {
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
	stateRootOut, err := migration.MigrateStateTree(c.Context, store, stateRootIn)
	if err != nil {
		return err
	}
	fmt.Printf("Migrated State Root: %s\n", stateRootOut)
	// TODO when specs actors creates an entry point state validation function
	// call it here on the new state
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
