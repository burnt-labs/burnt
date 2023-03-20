# Burnt Chain

Burnt Chain is a unique platform that stands out from other blockchains due to its focus on revenue-generating decentralized applications and fee sharing with validators. To achieve this, Burnt Chain leverages CosmWasm, a smart contract platform focusing on security, performance, and interoperability, making it the only smart contract platform for public blockchains with heavy adoption outside of the EVM world. Additionally, Burnt Chain features a custom Cosmos module called "schedule," enabling developers to schedule deferred and recurring computation on-chain without the need to run a centralized service to execute their tasks.

**If you're a developer looking for build and architecture information, go to [Development](development).**

## Why Burnt Chain?

Burnt Chain is not only focused on making it easy for users and creators to interact with the platform, but it has also been designed to provide several benefits for validators and token holders.

Firstly, Burnt Chain is a platform focused on garnishing platform fees from real on-chain revenue. It redistributes these fees to the validators, which incentivizes them to secure the network and keep it running smoothly.

Secondly, Burnt Chain limits platform inflation by subsidizing validator rewards with the platform fees gathered by the chain. This means that during periods of high transaction volume, the platform may not inflate the token supply at all. This leads to long-term value for the Burnt token.

Lastly, in the future, the chain will automatically swap non-Burnt tokens gathered as fees for Burnt tokens on the Osmosis DEX and then distribute those Burnt tokens to validators as rewards. This will further incentivize validators to secure the network and contribute to the growth of the platform.

### What is Cosmos SDK?

The Cosmos SDK is an open-source framework for building multi-asset public Proof-of-Stake (PoS) blockchains, like the Cosmos Hub, as well as permissioned Proof-of-Authority (PoA) blockchains. Blockchains built with the Cosmos SDK are generally referred to as application-specific blockchains.

The goal of the Cosmos SDK is to allow developers to easily create custom blockchains from scratch that can natively interoperate with other blockchains. The Cosmos SDK is a capabilities-based system that allows developers to better reason about the security of interactions between modules. 

### Developing on Burnt

Developing on Burnt Chain offers several unique benefits for developers. Burnt Chain is a permissioned chain, meaning smart contracts deployed to the chain must be approved by staking token holders. This ensures that only high-quality applications are deployed to the chain.

Burnt Chain seeks to provide a home for revenue-generating applications, ideally those that involve the sale of digital assets. Decentralized applications ("Dapps") on Burnt must share platform fees with the validators, which incentivizes them to secure the network and keep it running smoothly.

Dapps on Burnt Chain will be able to leverage a suite of services exclusively available to the platform, including a simple way to buy digital assets (e.g. NFTs) on Burnt Chain by paying with a credit card. Additionally, the platform offers simple ways for users to manage their digital identities, including social wallets and a mobile wallet solution developers can embed into their mobile apps. These services make it easier for developers to create user-friendly applications on the platform.

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