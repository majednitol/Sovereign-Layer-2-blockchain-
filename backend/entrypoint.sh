#!/bin/sh

if [ "$1" = "faucet" ]; then
    echo "Starting Faucet service..."
    CHAIN_HOME="${CHAIN_HOME:-/root/.sovereign}"
    # Ensure faucet key exists in keyring-test
    if ! chaind keys show faucet --keyring-backend test --home "${CHAIN_HOME}" >/dev/null 2>&1; then
        echo "Faucet key not found in keyring. Recovering from mnemonic..."
        echo "test test test test test test test test test test test junk" | chaind keys add faucet --recover --keyring-backend test --home "${CHAIN_HOME}"
    fi
    exec /app/faucet
fi

# Start ingestion service in the background
echo "Starting Ingestion service..."
/app/ingestion &
INGESTION_PID=$!

# Start projection service in the background
echo "Starting Projection service..."
/app/projection &
PROJECTION_PID=$!

# Start api gateway service in the foreground
echo "Starting API Gateway service..."
/app/api &
API_PID=$!

# Cleanup handler for graceful shutdown
cleanup() {
    echo "SIGTERM received. Stopping all services..."
    kill -TERM "$INGESTION_PID" 2>/dev/null
    kill -TERM "$PROJECTION_PID" 2>/dev/null
    kill -TERM "$API_PID" 2>/dev/null
    
    wait "$INGESTION_PID"
    wait "$PROJECTION_PID"
    wait "$API_PID"
    echo "All services stopped."
    exit 0
}

trap cleanup INT TERM

# Wait for the API process to complete
wait "$API_PID"
cleanup
