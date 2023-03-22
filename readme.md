# Burnt Chain

Burnt Chain is a unique platform that stands out from other blockchains due to its focus on revenue-generating decentralized applications and fee sharing with validators. To achieve this, Burnt Chain leverages CosmWasm, a smart contract platform focusing on security, performance, and interoperability, making it the only smart contract platform for public blockchains with heavy adoption outside of the EVM world. Burnt Chain features a custom Cosmos module called "schedule," enabling developers to schedule deferred and recurring computation on-chain without the need to run a centralized service to execute their tasks.

## Development

Burnt Chain is a golang project built on the [cosmos-sdk](https://github.com/cosmos/cosmos-sdk) with the help of [Ignite](https://ignite.com/).

### Build and Install

The `burntd` binary can be built via our Makefile with

```bash
make build
```

which will output a binary at `build/burntd`. To install `burntd` into your GOPATH, you can use

```bash
make install
```

You will need to ensure that the output of `echo $(go env GOPATH)/bin` is in your PATH.

### Burnt-specific Logic

Burnt Chain introduces a few custom features in the form of cosmos modules. These can be found in two places:

- First, in this repository under `x/schedule` you will find the code for our `schedule` module for scheduled and recurring computation. This module follows the pre cosmos-sdk v0.47.0 conventions.
- Second, in our [fork of cosmos-sdk](https://github.com/burnt-labs/cosmos-sdk/tree/mint) (on the `mint` branch) you will find our modified mint module under `x/mint`.
- Finally, the Burnt Chain is assembled of all of its component modules and configurations under the `app/` directory in this repo, with `app/app.go` as the primary entrypoint.

## Contributing

Contributing to the Burnt Chain codebase is highly encouraged. If a user has an issue running Burnt Chain, they can create an issue on our GitHub page. This will allow the team to investigate the issue and work on resolving it.

If a user would like to implement a new feature for Burnt Chain or contribute to the codebase, they can contact the team in Discord or create an issue on GitHub to discuss the costs and benefits. This will help ensure that the proposed feature aligns with the platform's vision and goals.

The Burnt Chain team welcomes contributions from the community and is committed to building a strong ecosystem of developers and users around the platform.
