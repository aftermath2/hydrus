#!/bin/bash
set -e

echo 'alias lncli="lncli --network regtest"' >> ~/.bashrc

HOSTNAME=$(hostname)

sed -i "s:HOSTNAME:$HOSTNAME:g" /tmp/lnd.conf

cp /tmp/lnd.conf /home/lnd/.lnd/lnd.conf

exec lnd
