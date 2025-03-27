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

If you use `docker compose down` that will kill the containers and the next time you start the environment all nodes will be assigned different IP addresses and they won't be able to reach each other. To fix this, connect all peers again manually using `lncli connect <pub_key>@<address>:<port>`.

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
