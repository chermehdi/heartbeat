# Heartbeat

The distributed simple registry server.

## API

Heartbeat exposes 2 types of APIs:

1. REST API to talk to the UI.
1. an RPC API to talk to the other nodes in the cluster.

### REST API description

- `GET /services`

This request returns the list of all available services along with their
available instances

```json
[
  {
    "name": "service-a",
    "uptime": 12312321,
    "instances": [
      "host1:port1",
      "host2:port2",
    ]
  },
  ...
]
```

- `GET /config`

This request returns the list of all the available key-values stored in the
heartbeat server

```json
[
  {
    "key": "build-id",
    "value": "12AEDE234",
  },
  ...
]
```

- `PUT /config`

```json
{
  "key": "key",
  "value": "value"
}
```

This creates the config with the key and value in the heartbeat server, the
server returns an `OK` status only if the key-value pair has been successfully
persisted by the **majority** of nodes of the heartbeat server.
