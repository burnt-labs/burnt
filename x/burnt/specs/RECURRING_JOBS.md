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

- `Schedule(ScheduleReq)`
- `Remove(RemoveReq)`

With the corresponding requests:

```protobuf
message ScheduleReq {
  // Signer address
  required bytes signer = 1;
  // Contract address
  required bytes contract = 2;
  // The name of the message to send to the contract. It must have no body and
  // the contract should return no result upon receipt.
  required string message_name = 3;
  // [Optional] address to pay gas fees for the contract. This address must
  // have an active feegrant established on behalf of the signer.
  optional bytes payer = 4;
}


message RemoveReq {
  // Signer address
  required bytes signer = 1;
  // Contract address
  required bytes contract = 2;
  // The name of the message to send to the contract. It must have no body and
  // the contract should return no result upon receipt.
  required string message_name = 3;
}
```

The primary mode of interaction with the module will be via a commandline
interface which will expose the following transaction methods:

- `schedule add [--payer payer] <contract> <function> <schedule>` - Creates a
  `ScheduleReq` with the defined parameters and broadcasts it to the validators.
- `schedule remove <contract> <function>` - Creates a
  `RemoveReq`, removing the (signer, contract, function) tuple from the
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
[uint64 block number][signer address][contract address][message name]
```

Depending on the implementation, we may not need to store a value at the key,
as it contains all the information necessary to invoke the contract. If there is
a `payer` specified on the scheduling request, it will be the value.

When invoking contracts, we will leverage the capabilities of the `feegrant`
module. When a contract is registered for the first time, we will validate with
the `feegrant` module that there is an active grant from the payer to the
signer, rejecting the transaction if there is not. When invoking the contract, 
we do so via the `wasm` module directly with a context of our creation and then
we deduct the fees via the `feegrant` module if a `payer` is present.

On success, we take the next scheduled block from the result of the invocation
if present and delete the entry from the store and reinsert it under a new key
prefixed by this block. If there is no block returned, we simply delete it and
do not reschedule.