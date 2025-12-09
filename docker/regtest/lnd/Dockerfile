ARG LND_VERSION=v0.20.0-beta

FROM lightninglabs/lnd:${LND_VERSION}

ARG USER_ID=10001
ARG GROUP_ID=10001

RUN apk --no-cache --no-progress add sudo=~1

RUN addgroup -g "$GROUP_ID" lnd && \
	adduser --disabled-password --shell "/sbin/nologin" --uid "${USER_ID}" lnd --ingroup lnd

COPY lnd.conf /tmp/lnd.conf

COPY entrypoint.sh /entrypoint.sh

COPY healthcheck.sh /healthcheck.sh

ENTRYPOINT ["/entrypoint.sh"]

HEALTHCHECK --start-period=120s CMD /healthcheck.sh
