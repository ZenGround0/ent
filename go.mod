module github.com/zenground0/ent

go 1.13

require (
	github.com/dgraph-io/badger/v2 v2.2007.2
	github.com/filecoin-project/filecoin-ffi v0.30.4-0.20200910194244-f640612a1a1f // indirect
	github.com/filecoin-project/go-address v0.0.4
	github.com/filecoin-project/go-state-types v0.0.0-20200928172055-2df22083d8ab
	github.com/filecoin-project/lotus v0.9.0
	github.com/filecoin-project/specs-actors v0.9.12
	github.com/filecoin-project/specs-actors/v2 v2.0.2
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-badger2 v0.1.1-0.20200708190120-187fc06f714e
	github.com/ipfs/go-ipfs-blockstore v1.0.1
	github.com/ipfs/go-ipld-cbor v0.0.5-0.20200428170625-a0bd04d3cbdf
	github.com/mitchellh/go-homedir v1.1.0
	github.com/urfave/cli/v2 v2.2.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi
