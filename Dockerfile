FROM golang:1.25-alpine3.23 AS builder

ARG USER_ID=10001
ARG GROUP_ID=10001 

RUN addgroup -g "$GROUP_ID" hydrus

RUN adduser \
    --disabled-password \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${USER_ID}" \
    hydrus --ingroup hydrus

WORKDIR /tmp/hydrus

COPY go.mod .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o main -ldflags="-s -w" .

# -----

FROM scratch

LABEL org.opencontainers.image.source=https://github.com/aftermath2/hydrus

COPY --from=builder /tmp/hydrus/main /usr/bin/hydrus

COPY --from=builder /etc/passwd /etc/passwd

USER hydrus

ENTRYPOINT ["/usr/bin/hydrus"]
