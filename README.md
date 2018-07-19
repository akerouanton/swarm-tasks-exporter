# Swarm tasks exporter

This little Prometheus exporter provides two metrics to better monitor Swarm
tasks and their state.

##### Summary

* [How it works](#how-it-works)
* [Metrics](#metrics)
* [Install](#install)
* [Configure](#configure)

## How it works?

In its current state, the exporter does:

* Watch swarm events about service create/update/remove, to update the number
  of desired replicas for replicated services.
* Watch swarm events about node create/remove, to update the number of desired
  replicas for global services.
* Regularly poll task list to update the gauge of service tasks segmented
  by state.

## Metrics

* `swarm_service_desired_replicas`: Gauge of how many replicas is desired, for
  every Swarm service (labels: `stack`, `service`, `service_mode`).
* `swarm_service_replicas_state`: Gauge of tasks state (labels: `stack`,
  `service`, `service_mode`, `state`).

## Install

This exporter is available on Docker Hub: [`akerouanton/swarm-tasks-exporter`](https://hub.docker.com/r/akerouanton/swarm-tasks-exporter/):

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock:ro akerouanton/swarm-tasks-exporter
```

Or, with docker-compose (or Swarm):

```yaml
services:
  tasks_exporter:
    image: akerouanton/swarm-tasks-exporter:0.1.0
    command: -log-level error
    volumes:
      - '/var/run/docker.sock:/var/run/docker.sock:ro'
    networks:
      - monit_prometheus
    deploy:
      replicas: 1
      restart_policy:
        condition: on-failure
      placement:
        constraints:
          - node.role == manager     
```

As you can see, when you want to deploy it to a Swarm cluster, it has to be
scheduled on a manager node, or it won't be able to access cluster events.

## Configure

You can use following flags to configure the exporter:

* `-listen-addr <ip:port>`: IP address and port to listen to (default 0.0.0.0:8888).
* `-poll-delay`: Delay in seconds between two polls (default 10s).
* `-log-format`: How log should be formatted. Either json or text (default text).
* `-log-level`: What's the minimum level of logs. Either debug, info, warn,
  error, fatal or panic (default info).

Moreover, this exporter supports the same env vars as the docker client:

* `DOCKER_HOST` to set the url to the docker server
* `DOCKER_API_VERSION` to set the version of the API to reach, leave empty for
  latest.
* `DOCKER_CERT_PATH` to load the TLS certificates from.
* `DOCKER_TLS_VERIFY` to enable or disable TLS verification, off by default.
