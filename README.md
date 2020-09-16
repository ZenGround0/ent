![ent logo](assets/old-trees.jpeg)
## ent

This is a tool for testing out state tree migrations on lotus chain data

## Usage

To get started you need data in a lotus directory at `~/.lotus`

- `ent migrate one <state-cid>` does a migration and outputs the new state tree cid
- `ent migrate chain <start-block-cid>` does a migration on all states between start header and genesis
- `ent validate <state-cid>` does a migration and runs long paranoid validation on the new state
- `ent roots <start-block-cid> <number-to-return>` outputs the given number of state roots walking back from the start block

All migrations are from specs actors v1 state to specs actors v2 state
