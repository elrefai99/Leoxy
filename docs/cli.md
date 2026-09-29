# Leoxy CLI

The Leoxy command line interface starts the reverse proxy, manages it as a background process, and prints the executable version.

## Build

From the repository root, build the executable:

```sh
go build -o leoxy ./cli
```

On Windows, build `leoxy.exe`:

```powershell
go build -o leoxy.exe ./cli
```

Run the commands from the directory containing the `leoxy/` configuration directory. The configuration is read from `leoxy/config.yaml` relative to the current working directory. If it does not exist, Leoxy creates a default configuration.

## Commands

### `leoxy`

Run the server in the foreground:

```sh
./leoxy
```

### `leoxy start`

Start the server as a background process:

```sh
./leoxy start
```

Leoxy appends process output to `leoxy/log/leoxy.log` and stores the process ID in `leoxy/keys/leoxy.pid`.

### `leoxy stop`

Stop the background process started with `leoxy start`:

```sh
./leoxy stop
```

The stop command reads the process ID from `leoxy/keys/leoxy.pid` and removes the PID file after stopping the process.

### `leoxy version`

Print the executable version:

```sh
./leoxy version
```

### Help

Cobra provides help for the root command and each visible subcommand:

```sh
./leoxy --help
./leoxy start --help
./leoxy stop --help
./leoxy version --help
```

Use `-h` instead of `--help` for the short flag.

## Configuration and environment

Configure the listener and upstreams in `leoxy/config.yaml`. See [the configuration guide](../README.md#configuration) for the available YAML settings and security options.

`PORT` overrides `server.port`. The `RATE_LIMIT`, `JWT_SECRET`, `JWT_ISSUER`, `JWT_AUDIENCE`, `API_KEYS`, and `REDIS_PASSWORD` environment variables override their corresponding global security settings. Keep credentials in the environment rather than committing them to the YAML file.

The `run` subcommand is an internal command used by `leoxy start`; it is hidden from CLI help.
