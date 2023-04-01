#!/bin/bash
set -e

LND_HOME="/home/lnd/.lnd"

echo 'alias lncli="lncli --network regtest"' >> ~/.bashrc

HOSTNAME=$(hostname)

sed -i "s:HOSTNAME:$HOSTNAME:g" /tmp/lnd.conf

mkdir -p $LND_HOME

chown -R lnd $LND_HOME

cp /tmp/lnd.conf $LND_HOME/lnd.conf

exec sudo -u lnd lnd
