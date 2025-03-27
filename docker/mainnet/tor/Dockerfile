FROM alpine:3.21

ARG USER_ID=10001
ARG GROUP_ID=10001
ENV TOR_DATA=/var/lib/tor

RUN apk --no-cache --no-progress add tor sudo shadow

RUN useradd -u $USER_ID --shell /bin/sh -m lnd && \
  usermod -g lnd tor

COPY entrypoint.sh /entrypoint.sh

COPY torrc /tmp/torrc

ENTRYPOINT [ "/entrypoint.sh" ]
