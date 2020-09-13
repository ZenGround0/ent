package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"

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
	Action:      runMigrateCmd,
}

var validateCmd = &cli.Command{
	Name:        "validate",
	Description: "validate a migration by checking lots of invariants",
	Action:      runValidateCmd,
}

func main() {
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

func runMigrateCmd(c *cli.Context) error {
	if !c.Args().Present() {
		return xerrors.Errorf("not enough args, need state root to migrate")
	}
	stateRootIn, err := cid.Decode(c.Args().First())
	if err != nil {
		return err
	}
	store, err := lib.LoadCborStore(c.Context)
	if err != nil {
		return err
	}
	stateRootOut, err := migration.MigrateStateTree(c.Context, store, stateRootIn)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", stateRootOut)
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
	store, err := lib.LoadCborStore(c.Context)
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
	iter, err := lib.NewChainStateIterator(c.Context, bcid)
	if err != nil {
		return err
	}
	for i := 0; !iter.Done() && i < num; i++ {
		roots = append(roots, iter.Val())
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
