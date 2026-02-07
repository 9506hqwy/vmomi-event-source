# VMOMI Event Source

This repository provides event source for vSphere infrastructure.
It collects and push vSphere infrastructure events to Grafana Loki.

## Features

- Collects vSphere infrastructure events in realtime
- Pushes events to Grafana Loki

### Labels and Metadata

Expose event with follow labels.

| Label        | Description        |
| :----------- | :----------------- |
| severity     | Severity for event |
| service_name | Service name       |

Expose event with follow metadata.

| Name                       | Description                           |
| :------------------------- | :------------------------------------ |
| cluster                    | Cluster name for event source         |
| datacenter                 | Datacenter name for event source      |
| datastore                  | Datastore name for event source       |
| distributed_virtual_switch | DVS name for event source             |
| host                       | Host name for event source            |
| network                    | Network name for event source         |
| user                       | User name for event                   |
| vm                         | Virtual machine name for event source |
| event_type_id              | Internal kind for event               |

## Build

Build binary.

```sh
go build -o bin/vmomi-event-source ./cmd/vmomi-event-source
```

Or build container image.

```sh
docker build -t vmomi-event-source .
```

Add `Z` option at bind mount operation in *Dockerfile* if using podman with SELinux.

## Usage

Run application.

```sh
$ ./bin/vmomi-event-source loki collect -h
VMOMI Event Source Loki Collect

Usage:
  vmomi-event-source loki collect [flags]

Flags:
  -h, --help      help for collect
  -v, --version   version for collect

Global Flags:
      --log-level string           Log level. (default "INFO")
      --loki-no-verify-ssl         Skip SSL verification.
      --loki-service-name string   Loki service name. (default "vmomi-event-source")
      --loki-url string            Loki URL. (default "http://127.0.0.1:3100/loki/api/v1/push")
      --no-verify-ssl              Skip SSL verification.
      --password string            vSphere server password.
      --tenant string              Loki tenant.
      --timeout int                API call timeout seconds. (default 10)
      --url string                 vSphere server URL. (default "https://127.0.0.1/sdk")
      --user string                vSphere server username.
```

Set environment variable instead of arguments.

| Argument             | Environment Variable                    |
| :------------------- | :-------------------------------------- |
| --log-level          | VMOMI_EVENT_SOURCE_LOG_LEVEL            |
| --loki-no-verify-ssl | VMOMI_EVENT_SOURCE_LOKI_NO_VERIFY_SSL   |
| --loki-service-name  | VMOMI_EVENT_SOURCE_LOKI_SERVICE_NAME    |
| --loki-url           | VMOMI_EVENT_SOURCE_LOKI_URL             |
| --no-verify-ssl      | VMOMI_EVENT_SOURCE_TARGET_NO_VERIFY_SSL |
| --password           | VMOMI_EVENT_SOURCE_TARGET_PASSWORD      |
| --tenant             | VMOMI_EVENT_SOURCE_LOKI_TENANT          |
| --timeout            | VMOMI_EVENT_SOURCE_TARGET_TIMEOUT       |
| --url                | VMOMI_EVENT_SOURCE_TARGET_URL           |
| --user               | VMOMI_EVENT_SOURCE_TARGET_USER          |

Or run container.

```sh
docker run -d \
    -e VMOMI_EVENT_SOURCE_TARGET_URL=<URL> \
    -e VMOMI_EVENT_SOURCE_TARGET_USER=<USER> \
    -e VMOMI_EVENT_SOURCE_TARGET_PASSWORD=<PASSWORD> \
    -e VMOMI_EVENT_SOURCE_LOKI_URL=<URL> \
    vmomi-event-source loki collect
```

## TODO

- Push events between network disconnected.
