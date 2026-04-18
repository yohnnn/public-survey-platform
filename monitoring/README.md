# Monitoring Stack

This project now includes a basic monitoring setup with Prometheus + Grafana.

## Components

- Prometheus: metrics collection and alert evaluation
- Grafana: dashboards and visualization
- cAdvisor: container CPU/memory metrics
- Node Exporter: host-level metrics
- Redis Exporter: Redis metrics
- Blackbox Exporter: HTTP/TCP health probes

## Endpoints

- Prometheus UI: http://127.0.0.1:9090
- Grafana UI: http://127.0.0.1:3001
  - Username: admin
  - Password: admin

## Preconfigured Dashboard

- Folder: Public Survey Platform
- Dashboard: PSP Overview

## Config Locations

- Prometheus scrape config: monitoring/prometheus/prometheus.yml
- Prometheus alert rules: monitoring/prometheus/alerts.yml
- Blackbox config: monitoring/blackbox/blackbox.yml
- Grafana provisioning:
  - monitoring/grafana/provisioning/datasources/prometheus.yml
  - monitoring/grafana/provisioning/dashboards/dashboards.yml
- Grafana dashboard JSON:
  - monitoring/grafana/dashboards/psp-overview.json

## Basic Alert Scenarios

1. Target down
- Alert: PrometheusTargetDown
- Trigger example:
  - Stop one exporter or service, e.g. `docker compose stop redis-exporter`

2. Health probe failed
- Alert: BlackboxProbeFailed
- Trigger example:
  - Stop `api-service` or break `/healthz` response path

3. High container CPU
- Alert: HighContainerCPUUsage
- Trigger example:
  - Run a CPU-heavy workload inside one service container for >5m

4. High container memory
- Alert: HighContainerMemoryUsage
- Trigger example:
  - Run memory-heavy workload in a container (>600 MiB for >5m)

5. Redis exporter down
- Alert: RedisExporterDown
- Trigger example:
  - Stop `redis-exporter`

## Start/Stop

Start full stack:

```bash
docker compose up -d --build
```

Start only monitoring layer:

```bash
docker compose up -d cadvisor node-exporter redis-exporter blackbox-exporter prometheus grafana
```

Stop monitoring layer:

```bash
docker compose stop cadvisor node-exporter redis-exporter blackbox-exporter prometheus grafana
```
