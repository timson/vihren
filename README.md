# Vihren

Lightweight continuous profiling for Intel gProfiler.

> [!NOTE]  
> **Vihren** is the highest peak in the Pirin Mountains of Bulgaria.

## Motivation
[Intel gProfiler](https://github.com/intel/gprofiler) is a great continuous profiling agent and is easy to run in local environments 
or Kubernetes. However, its backend, Intel Performance Studio, requires additional infrastructure
such as a database, S3, SQS, and other services.

Vihren is a lightweight backend for Intel gProfiler, designed for simpler deployments.
It is based on [Intel Performance Studio](https://github.com/intel/gprofiler-performance-studio),
but replaces external dependencies with a local queue
and embedded `chdb`, making it easier to run in homelabs, local environments,
and small Kubernetes setups.

## Features

- **Embedded ClickHouse** — no external database to manage; data is stored locally via [chdb](https://github.com/chdb-io/chdb-go)
- **Multi-resolution aggregation** — raw, hourly, and daily rollups with configurable retention
- **Docker & Helm ready** — single container deployment, Helm chart included
- **Built for Intel gProfiler** — works with the [Intel gProfiler](https://github.com/intel/gprofiler) agent out of the box

## Quick Start

```sh
docker run -p 8080:8080 ghcr.io/timson/vihren
```

Open [http://localhost:8080/ui](http://localhost:8080/ui) in your browser.

Point a [gProfiler](https://github.com/intel/gprofiler) agent at the server:

```sh
docker run --name granulate-gprofiler -d --restart=on-failure:10 --pid=host --userns=host 
--privileged intel/gprofiler:latest -cu --token="1234" --service-name=<server_name>
--server-host=http://<ip:port> --glogger-server=http://<ip:port> --no-verify
```

## Configuration

All settings are configured via environment variables with the `VIHREN_` prefix.

| Variable | Default | Description |
|---|---|---|
| `VIHREN_SERVER_PORT` | `8080` | HTTP server port |
| `VIHREN_DB_FILENAME` | `flamedb` | chdb session directory (data storage path) |
| `VIHREN_DB_RAWRETENTIONDAYS` | `7` | Raw data retention in days |
| `VIHREN_DB_MINUTERETENTIONDAYS` | `365` | Minute aggregation retention in days |
| `VIHREN_DB_HOURLYRETENTIONDAYS` | `90` | Hourly aggregation retention in days |
| `VIHREN_DB_DAILYRETENTIONDAYS` | `365` | Daily aggregation retention in days |
| `VIHREN_DB_MINSTACKROWS` | `100000` | Minimum stack rows per query |
| `VIHREN_INDEXER_WORKERS` | `2` | Number of indexer worker goroutines |

## License

Apache License 2.0 — see [LICENSE](LICENSE).

This project includes code from Intel's [gprofiler-performance-studio](https://github.com/intel/gprofiler-performance-studio). See [NOTICE](NOTICE) for details.
