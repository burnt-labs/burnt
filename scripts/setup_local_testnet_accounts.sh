VALIDATOR_MNEMONIC="clinic tube choose fade collect fish original recipe pumpkin fantasy enrich sunny pattern regret blouse organ april carpet guitar skin work moon fatigue hurdle"
FAUCET_MNEMONIC="decorate corn happy degree artist trouble color mountain shadow hazard canal zone hunt unfold deny glove famous area arrow cup under sadness salute item"

VALIDATOR_KEY_NAME="${VALIDATOR_KEY_NAME:-validator}"
FAUCET_KEY_NAME="${FAUCET_KEY_NAME:-faucet}"

# Create keys locally if necessary
if [[ "$(burntd keys list --output json | jq "map(select(.name == \"$VALIDATOR_KEY_NAME\")) | length")" == "0" ]]; then
    echo "Validator key not present, creating..."
    echo $VALIDATOR_MNEMONIC | burntd keys add $VALIDATOR_KEY_NAME --recover
fi

if [[ "$(burntd keys list --output json | jq "map(select(.name == \"$FAUCET_KEY_NAME\")) | length")" == "0" ]]; then
    echo "Faucert key not present, creating..."
    echo $FAUCET_MNEMONIC | burntd keys add $FAUCET_KEY_NAME --recover
fi
