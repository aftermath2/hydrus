FROM debian:stable-slim

ARG USER_ID=10001
ARG GROUP_ID=10001
ARG BITCOIN_VERSION="28.1"
ENV PATH=/opt/bitcoin-${BITCOIN_VERSION}/bin:$PATH

RUN groupadd -f -g $GROUP_ID bitcoin && useradd -m -r -u $USER_ID -g $GROUP_ID bitcoin

RUN chown -R $USER_ID:$GROUP_ID "$(getent passwd bitcoin | cut -d: -f6)"

RUN apt-get update -y \
  && apt-get install -y curl wget \
  && apt-get clean \
  && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN SYS_ARCH="$(uname -m)" \
  && wget https://bitcoincore.org/bin/bitcoin-core-${BITCOIN_VERSION}/bitcoin-${BITCOIN_VERSION}-${SYS_ARCH}-linux-gnu.tar.gz \
  && tar -xzf bitcoin-${BITCOIN_VERSION}-${SYS_ARCH}-linux-gnu.tar.gz -C /opt \
  && rm bitcoin-${BITCOIN_VERSION}-${SYS_ARCH}-linux-gnu.tar.gz

COPY --chown=${USER_ID}:${GROUP_ID} bitcoin.conf /tmp/bitcoin.conf

COPY entrypoint.sh /entrypoint.sh

USER bitcoin

ENTRYPOINT ["/entrypoint.sh"]
