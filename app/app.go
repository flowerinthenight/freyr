package app

import (
	"sync"

	"cloud.google.com/go/spanner"
	"github.com/flowerinthenight/hedge"
	"github.com/flowerinthenight/timedoff"
)

const (
	EventSource = "hedged"
)

type App struct {
	*sync.Mutex
	SpannerDb *spanner.Client
	Hedge     *hedge.Op

	// When active/ok, we have a live leader in the group.
	LeaderOk *timedoff.TimedOff
}
