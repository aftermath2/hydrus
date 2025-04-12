ARG LND_VERSION=v0.18.5-beta

FROM lightninglabs/lnd:${LND_VERSION}

ARG USER_ID=10001
ARG GROUP_ID=10001

RUN addgroup -g "$GROUP_ID" lnd && \
	adduser --disabled-password --shell "/sbin/nologin" --uid "${USER_ID}" lnd --ingroup lnd

COPY --chown=${USER_ID}:${GROUP_ID} lnd.conf /tmp/lnd.conf

COPY entrypoint.sh /entrypoint.sh

COPY healthcheck.sh /healthcheck.sh

USER lnd

ENTRYPOINT ["/entrypoint.sh"]

HEALTHCHECK --interval=5s --timeout=20s --start-period=5s --retries=12 CMD /healthcheck.sh
