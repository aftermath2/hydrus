FROM debian:stable-slim

ARG BITCOIN_VERSION="28.1"
ENV PATH=/opt/bitcoin-${BITCOIN_VERSION}/bin:$PATH

RUN apt-get update -y \
  && apt-get install -y curl wget \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN SYS_ARCH="$(uname -m)" \
  && wget https://bitcoincore.org/bin/bitcoin-core-${BITCOIN_VERSION}/bitcoin-${BITCOIN_VERSION}-${SYS_ARCH}-linux-gnu.tar.gz \
  && tar -xzf bitcoin-${BITCOIN_VERSION}-${SYS_ARCH}-linux-gnu.tar.gz -C /opt \
  && rm bitcoin-${BITCOIN_VERSION}-${SYS_ARCH}-linux-gnu.tar.gz

COPY bitcoin.conf /tmp/bitcoin.conf

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
