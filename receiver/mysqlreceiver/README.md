# MySQL Receiver

This receiver queries MySQL's global status and InnoDB tables.

Supported pipeline types: `metrics`

> :construction: This receiver is in **BETA**. Configuration fields and metric data model are subject to change.

## Prerequisites

This receiver supports MySQL version 8.0

Collecting most metrics requires the ability to execute `SHOW GLOBAL STATUS`. The `buffer_pool_size` metric requires access to the `information_schema.innodb_metrics` table. Please refer to [setup.sh](./testdata/integration/scripts/setup.sh) for an example of how to configure these permissions. 

## Configuration


The following settings are optional:
- `endpoint`: (default = `localhost:3306`)
- `username`: (default = `root`)
- `password`: The password to the username.
- `allow_native_passwords`: (default = `true`)
- `database`: The database name. If not specified, metrics will be collected for all databases.

- `collection_interval` (default = `10s`): This receiver collects metrics on an interval. This value must be a string readable by Golang's [time.ParseDuration](https://pkg.go.dev/time#ParseDuration). Valid time units are `ns`, `us` (or `µs`), `ms`, `s`, `m`, `h`.

- `transport`: (default = `tcp`): Defines the network to use for connecting to the server.

### Example Configuration

```yaml
receivers:
  mysql:
    endpoint: localhost:3306
    username: otel
    password: $MYSQL_PASSWORD
    database: otel
    collection_interval: 10s
```

The full list of settings exposed for this receiver are documented [here](./config.go) with detailed sample configurations [here](./testdata/config.yaml).

## Metrics

Details about the metrics produced by this receiver can be found in [metadata.yaml](./metadata.yaml)
