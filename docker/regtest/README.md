## Hydrus regtest development environment

This document outlines how to set up a development environment with 1 Bitcoin node, 8 LND nodes and 1 Hydrus instance.

### Commands

#### Build the environment

```console
docker compose build
```

#### Start the environment

```console
docker compose up -d
```

#### Stop the environment

```console
docker compose stop
```

#### Restart hydrus

```console
docker restart hydrus
```

#### Show a hydrus logs

```console
docker logs -f hydrus
```

#### Show a node's logs

```console
docker logs -f <hostname>
docker logs -f alice
```
