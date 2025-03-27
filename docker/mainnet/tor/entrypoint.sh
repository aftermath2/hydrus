#!/bin/sh
set -e

if [ ! -f "/etc/tor/torrc" ]; then
    cp /tmp/torrc /etc/tor/torrc
fi

chown -R tor:lnd "${TOR_DATA}"
chown -R :lnd /etc/tor
chown -R lnd:lnd /home/lnd

exec sudo -u tor /usr/bin/tor
