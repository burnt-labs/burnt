# Recurring Contract Execution in CosmWasm

## Objective

To enable the automated scheduling and execution of smart contracts via a
validator-run module.

In order to use the module, a programmer must first deploy an instance of a
contract that they would like to receive scheduled callbacks. This contract
must have at least one method, to be designated as a callback, that accepts no
arguments and returns an uint64 representing the next block it should be invoked
on. If the block designated is less than or equal to the current block, the
invocation should fail.

Programmers may interact with the module by sending the following message to
the module:

- `AddSchedule(MsgAddSchedule)`
- `RemoveSchedule(MsgRemoveSchedule)`

With the corresponding requests:

```protobuf
message MsgAddSchedule {
  // Signer address
  required string signer = 1;
  // Contract address
  required string contract = 2;
  // The body of the message to send to the contract.
  required bytes call_body = 3;
  // The block height at which the call should be made
  optional bytes block_height = 4;
}


message MsgRemoveSchedule {
  // Signer address
  required bytes signer = 1;
  // Contract address
  required bytes contract = 2;
}
```

The primary mode of interaction with the module will be via a commandline
interface which will expose the following transaction methods:

- `schedule add <contract> <call_body> <block_height>` - Creates a
  `MsgAddSchedule` with the defined parameters and broadcasts it to the validators.
- `schedule remove <contract>` - Creates a
  `MsgRemoveSchedule`, removing the (signer, contract, function) tuple from the
  scheduler.

### Example Contract

A simple example of how a programmer might implement a scheduled callback.

```rust
fn scheduled_callbacks(
  deps: DepsMut,
  env: Env,
  info: MessageInfo,
  msg: ExecuteMsg,
) -> Result<Response, ContractError> {
  // Do something
  // ...
  // Schedule next execution on block 1337
  Ok(1337)
}

pub fn execute(
  deps: DepsMut,
  env: Env,
  info: MessageInfo,
  msg: ExecuteMsg,
) -> Result<Response, ContractError> {
  match msg {
    // ...
    ExecuteMsg::ScheduledCallback => scheduled_callback(deps, env, info, amount)
  }
}
```

## Implementation

In order to realize automated code execution without exposing ourselves to
massive cost, we will implement a Cosmos module that processes the messages
defined above. Its keeper will manage a store to register the callbacks along
with the relevant metadata.

The module will implement and `EndBlocker` that, every block, reads the store
for registered callbacks due to execute on this block. To make this efficient,
the callbacks registered in the store will be written under keys prefixed by the
block they should next execute on:

```
[uint64 block number][signer address][contract address]
```

Depending on the implementation, we may not need to store a value at the key,
as it contains all the information necessary to invoke the contract. 

When invoking contracts, we will debit the balance of the contract to pay for
the execution fees. We expect the usage of proxy contracts to be used heavily
for this, so that the balance for scheduled execution doesn't interfere with the
balance of the target contract for business uses. There is nothing, however,
stopping a contract from being scheduled directly if it chooses.

On success, we take the next scheduled block from the result of the invocation
if present and delete the entry from the store and reinsert it under a new key
prefixed by this block. If there is no block returned, we simply delete it and 
not reschedule.

## Outstanding Questions

Should we charge more for events scheduled further in the future?

Should we pay for storage of registered callbacks over time?
