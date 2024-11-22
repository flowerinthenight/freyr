[![build](https://github.com/flowerinthenight/hedged/actions/workflows/main.yml/badge.svg)](https://github.com/flowerinthenight/hedged/actions/workflows/main.yml)

## Overview

A long-running service based on [hedge](https://github.com/flowerinthenight/hedge).

## API Reference

#### SUBLDR

Subscribe to leader notifications. Notifications will be sent to the provided `/path/to/socket` file. If `interval-in-seconds` in not provided, it will default to 1s. Minimum interval is 1s.

``` sh
# Command:
$SUBLDR <path/to/socket> [interval-in-seconds]

# Example:
$SUBLDR /tmp/hedged-notify.sock 2
```

#### UNSUBLDR

Unsubscribe from leader notifications.

``` sh
# Command:
$SUBLDR
```
