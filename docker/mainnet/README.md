## Hydrus mainnet testing environment

This environment is designed to test hydrus against the mainnet network graph.

It spins up a lightweight neutrino node that adds a peer and loads the graph to then execute hydrus on it.

Make sure the configuration has `dry-run` enabled:

```yml
agent.dry_run: true
```

### Setup

The first time the enviroment is spun out, the LND wallet must be initialized to start the daemon. For that, execute:

```console
lncli --tlscertpath /home/lnd/.lnd/tls.cert create
```

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
