## Installation

To install Hydrus, follow one of these options:

1. Download the pre-compiled binaries from the [releases](https://github.com/aftermath2/hydrus/releases) page.
2. Compile it yourself from source code (Go required)

```
go build -o hydrus -ldflags="-s -w" .
```

3. Build the Docker image

```
docker build -t hydrus .
```

## Scheduling

Hydrus is designed to be executed on a regular basis to keep your node connected to the best peers and close channels that are not performing well.

The interval between executions depends on what the user decides, but it is recommended to not run hydrus more than once a week. This is because there isn't much time for the scores to change in a short timeframe, you will end up consuming unnecessary resources and spending more in fees as the channels opening batches will be smaller.

Good timeframes are probably weekly, semi-monthly or monthly.

### Systemd (recommended)

1. Create a service, check out [hydrus.service](./hydrus.service) for a sample file.

2. Create the timer for the service. It is suggested to execute Hydrus on the weekends, as fees are typically lower during this period.

Check out [hydrus.timer](./hydrus.timer) for a sample file.

3. Enable and start timer

```console
sudo systemctl daemon-reload
sudo systemctl enable --now hydrus.timer
sudo systemctl start hydrus.timer
```

### Cron

Scheduling tasks using cron is much simpler, but it has some disadvantages like no failover mechanisms or logs recording. To persist the logs, redirect the standard streams to a file.

To run hydrus using cron, edit the cron tab with `crontab -e` and paste the following line

```
0 20 * * 6 /usr/local/bin/hydrus
```

This will execute hydrus every Saturday at 20:00 (8PM).
