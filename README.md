![ent logo](assets/old-trees.jpeg)
## ent

This is a tool for testing out state tree migrations on lotus chain data

## Usage

To get started you need data in a lotus directory at `~/.lotus`

- `ent migrate one <state-cid> <state-epoch>` does a migration and outputs the new state tree cid
- `ent migrate chain <start-block-cid>` does a migration on all states between start header and genesis
- `ent validate <state-cid> <state-epoch>` runs long paranoid validation on the new state

`ent migrate one` and `ent migrate chain` take a `--validate` command for running a validation after a migratino
For a migration directly comparable to a filecoin protocol migration over the input `<state-cid>` provide a `<state-epoch>` equal to the epoch the state was created in. In other words use the height of the parent tipset of a header containing `<state-cid>`.

Migrations are from specs actors v1 state to specs actors v2 state
