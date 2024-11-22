[![build](https://github.com/flowerinthenight/hedged/actions/workflows/main.yml/badge.svg)](https://github.com/flowerinthenight/hedged/actions/workflows/main.yml)
[![Docker Repository on Quay](https://quay.io/repository/flowerinthenight/hedged/status "Docker Repository on Quay")](https://quay.io/repository/flowerinthenight/hedged)

## Overview

A long-running service based on [hedge](https://github.com/flowerinthenight/hedge).

Sample run:

``` sh
# Build the binary:
$ go build -v

# Run 1st instance:
$ ./hedged run --logtostderr \
  --db projects/{v}/instances/{v}/databases/{v} \
  --host-port :8080 \
  --lock-table mylocktable \
  --log-table mylocktable_log \
  --socket-file /tmp/hedged-8080.sock

# Run 2nd instance (different terminal):
$ ./hedged run --logtostderr \
  --db projects/{v}/instances/{v}/databases/{v} \
  --host-port :8082 \
  --lock-table mylocktable \
  --log-table mylocktable_log \
  --socket-file /tmp/hedged-8082.sock

# Run a sink reader for notifications from 1st instance (different terminal):
$ ./hedged sink --logtostderr /tmp/hedged-notify-8080.sock

# Run a sink reader for notifications from 2nd instance (different terminal):
$ ./hedged sink --logtostderr /tmp/hedged-notify-8082.sock

# Subscribe to notifications from 1st instance thru API:
$ ./hedged api SUBLDR /tmp/hedged-notify-8080.sock --socket-file /tmp/hedged-8080.sock

# Subscribe to notifications from 2nd instance thru API:
$ ./hedged api SUBLDR /tmp/hedged-notify-8082.sock --socket-file /tmp/hedged-8082.sock
```

## API Reference

The API protocol uses a subset of Redis' [RESP](https://redis.io/docs/latest/develop/reference/protocol-spec/) wire protocol; specifically, **Simple strings**, **Simple errors**, and **Bulk strings**.

#### SUBLDR

Subscribe to leader notifications. Notifications will be sent to the provided `/path/to/socket` file.

``` sh
# Command:
$SUBLDR <path/to/socket>

# Example:
$SUBLDR /tmp/hedged-notify.sock
```

#### UNSUBLDR

Unsubscribe from leader notifications.

``` sh
# Command:
$SUBLDR
```
