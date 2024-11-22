package app

import (
	"sync"
	"sync/atomic"

	"cloud.google.com/go/spanner"
	"github.com/flowerinthenight/hedge"
	"github.com/flowerinthenight/timedoff"
)

const (
	EventSource = "hedged"

	CRLF = "\r\n"
)

type Data struct {
	SpannerDb *spanner.Client
	Hedge     *hedge.Op

	// When active/ok, we have a live leader in the group.
	LeaderOk *timedoff.TimedOff

	SubLdrMutex    sync.Mutex
	SubLdrSocket   string
	SubLdrInterval atomic.Int64
}
