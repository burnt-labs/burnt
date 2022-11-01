#!/bin/sh

if [[ ! -f /burnt/data/priv_validator_state.json ]]; then
    mv /burnt/config /burnt-config
    burntd init fogo --chain-id burnt-local-testnet --home /burnt
    rm -r /burnt/config
    mv /burnt-config /burnt/config
fi

burntd start --home /burnt
