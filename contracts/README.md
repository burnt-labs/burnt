## Contracts

Contracts in this directory are used for test. The source is maintained in
https://github.com/BurntFinance/cw-nfts

### Building
In the contract directory, run  
```RUSTFLAGS='-C link-arg=-s' cargo build --release --target wasm32-unknown-unknown```

### Optimizing
The maximum size for cosmwasm is quite small. To make the contracts fit the limit, use [binaryen](https://github.com/WebAssembly/binaryen)
```wasmopt -Os -o output.wasm input.wasm```