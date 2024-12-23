[![build](https://github.com/flowerinthenight/freyr/actions/workflows/main.yml/badge.svg)](https://github.com/flowerinthenight/freyr/actions/workflows/main.yml)
[![Docker Repository on Quay](https://quay.io/repository/flowerinthenight/freyr/status "Docker Repository on Quay")](https://quay.io/repository/flowerinthenight/freyr)

## Overview

Companion service to [freya](https://github.com/flowerinthenight/freya/).

Sample run:

``` sh
# Build the binary:
$ go build -v

# Run 1st instance:
$ ./freyr run --logtostderr \
  --db projects/{v}/instances/{v}/databases/{v} \
  --host-port :8080 \
  --lock-table mylocktable \
  --log-table mylocktable_log \
  --socket-file /tmp/freyr-8080.sock

# Run 2nd instance (different terminal):
$ ./freyr run --logtostderr \
  --db projects/{v}/instances/{v}/databases/{v} \
  --host-port :8082 \
  --lock-table mylocktable \
  --log-table mylocktable_log \
  --socket-file /tmp/freyr-8082.sock

# Run a sink reader for notifications from 1st instance (different terminal):
$ ./freyr sink --logtostderr /tmp/freyr-notify-8080.sock

# Run a sink reader for notifications from 2nd instance (different terminal):
$ ./freyr sink --logtostderr /tmp/freyr-notify-8082.sock

# Subscribe to notifications from 1st instance thru API:
$ ./freyr api SUBLDR /tmp/freyr-notify-8080.sock --socket-file /tmp/freyr-8080.sock

# Subscribe to notifications from 2nd instance thru API:
$ ./freyr api SUBLDR /tmp/freyr-notify-8082.sock --socket-file /tmp/freyr-8082.sock
```

## API Reference

The API protocol uses a subset of Redis' [RESP](https://redis.io/docs/latest/develop/reference/protocol-spec/) wire protocol; specifically, [Simple strings](https://redis.io/docs/latest/develop/reference/protocol-spec/#simple-strings), [Simple errors](https://redis.io/docs/latest/develop/reference/protocol-spec/#simple-errors), and [Bulk strings](https://redis.io/docs/latest/develop/reference/protocol-spec/#bulk-strings). Commands use the `$<length>\r\n<data>\r\n` format, return values use the `+<data>\r\n` format, and error messages use the `-Error message\r\n` format.

#### SUBLDR

Subscribe to leader notifications. Notifications will be sent to the provided `/path/to/socket` file.

``` sh
# Command:
SUBLDR <path/to/socket>

# Example:
SUBLDR /tmp/freyr-notify.sock
```

#### UNSUBLDR

Unsubscribe from leader notifications.

``` sh
# Command:
SUBLDR
```
