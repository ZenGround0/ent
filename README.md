This is a tool for testing out state tree migrations on lotus chain data
To get started you need data in a lotus directory at `.lotus`

`huorn migrate <state-cid>` does a migration and outputs the new state tree cid
`huorn validate <state-cid>` does a migration and runs long paranoid validation on the new state
`huorn roots <start-block-cid> <number-to-return>` outputs the given number of state roots walking back from the start block

All migrations are from specs actors v1 state to specs actors v2 state