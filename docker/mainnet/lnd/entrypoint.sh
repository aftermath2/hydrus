#!/bin/bash
set -e

LND_HOME="/home/lnd/.lnd"
SPEEDLOADER_URL="https://egs.lnze.us"
LOCAL_GRAPH_PATH="$LND_HOME/data/graph/mainnet/channel.db"

cp /tmp/lnd.conf $LND_HOME/lnd.conf

echo ${AUTO_UNLOCK_PWD} > /tmp/pwd

LOCAL_GRAPH_SUM=$(md5sum $LOCAL_GRAPH_PATH | cut -d ' ' -f 1)

EXTERNAL_GRAPH_SUM=$(curl -s $SPEEDLOADER_URL/mainnet/graph/MD5SUMS | cut -d ' ' -f 1)

if [ "$LOCAL_GRAPH_SUM" != "$EXTERNAL_GRAPH_SUM" ]; then
  	# Download external graph to avoid constructing it ourselves, which takes a considerable amount of time
	echo "Downloading graph from external service"

	curl -s -o $LOCAL_GRAPH_PATH $SPEEDLOADER_URL/mainnet/graph/graph-001d.db
fi

exec lnd
