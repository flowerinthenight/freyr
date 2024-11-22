## Overview

A generic daemon based on [hedge](https://github.com/flowerinthenight/hedge).

## API Reference

#### SUBLDR

Subscribe to leader notifications. If `interval-in-seconds` in not provided, it will default to 1s.

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
