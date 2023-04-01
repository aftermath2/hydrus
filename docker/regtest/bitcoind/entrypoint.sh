#!/bin/bash
set -e

BITCOIN_HOME="/home/bitcoin/.bitcoin"

stop() {
    bitcoin-cli stop
	# Wait until the shutdown is done
	grep -q "Shutdown: done" <(tail -f $BITCOIN_HOME/regtest/debug.log)
}

# Trap SIGTERM
trap stop SIGTERM

cp /tmp/bitcoin.conf $BITCOIN_HOME/bitcoin.conf

bitcoind &

if [ ! -f "$BITCOIN_HOME/regtest/wallets/wallet.dat" ]; then
	bitcoin-cli -rpcwait createwallet ""
	bitcoin-cli generatetoaddress 101 $(bitcoin-cli getnewaddress)
fi

# LND gets stuck if there are no new blocks to sync
bitcoin-cli generatetoaddress 1 $(bitcoin-cli -rpcwait getnewaddress)

wait $!
