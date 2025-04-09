## Installation

To install Hydrus, follow one of these options:

1. Download the pre-compiled binaries from the [releases](https://github.com/aftermath2/hydrus/releases) page.
2. Compile it yourself from source code (Go required)

```bash
go build -o hydrus -ldflags="-s -w" .
# or
go install -ldflags="-s -w" .
```

3. Build the Docker image

```bash
docker build -t hydrus .
```

### Systemd

To run Hydrus in the background, the recommended way of doing it is with systemd. Follow the steps below to create and start a service.

1. Create a service, check out [hydrus.service](./hydrus.service) for a sample file.

2. Enable and start the service

```console
sudo systemctl daemon-reload
sudo systemctl enable hydrus.service
sudo systemctl start hydrus.service
```

> [!Note]
> To run the agent on a specific day/time, use `systemd-run --on-calendar=<date> systemctl start hydrus.service` or a timer (see [hydrus.timer](./hydrus.timer)).

### Cron

Scheduling tasks using cron is much simpler, but it has some disadvantages like no failover mechanisms or logs recording. To persist the logs, redirect the standard streams to a file.

To run hydrus `chanenls open` command using cron, edit the cron tab with `crontab -e` and paste the following line

```
0 20 * * 6 /usr/local/bin/hydrus channels open
```

This will instruct hydrus to look for new nodes to open channels with every Saturday at 20:00 (8PM).
