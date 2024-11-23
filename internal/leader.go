package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/flowerinthenight/freyr/app"
	"github.com/flowerinthenight/hedge"
	"github.com/golang/glog"
)

var (
	CtrlPingPong = "CTRL_PING_PONG"

	leader = map[string]func(app *app.Data, e *cloudevents.Event) ([]byte, error){
		CtrlPingPong: doLeaderPingPong,
	}
)

func LeaderHandler(data interface{}, msg []byte) ([]byte, error) {
	app := data.(*app.Data)
	var e cloudevents.Event
	err := json.Unmarshal(msg, &e)
	if err != nil {
		glog.Errorf("Unmarshal failed: %v", err)
		return nil, err
	}

	if _, ok := leader[e.Type()]; !ok {
		return nil, fmt.Errorf("failed: unsupported type: %v", e.Type())
	}

	return leader[e.Type()](app, &e)
}

func doLeaderPingPong(app *app.Data, e *cloudevents.Event) ([]byte, error) {
	switch {
	case string(e.Data()) != "PING":
		return nil, fmt.Errorf("invalid message")
	default:
		return []byte("PONG"), nil
	}
}

type LeaderLive struct{ *app.Data }

func (l *LeaderLive) Run(ctx context.Context) {
	glog.Infof("start leader liveness broadcaster (every 5mins)")
	ticker := time.NewTicker(time.Minute * 5)
	var active int32
	do := func() {
		atomic.StoreInt32(&active, 1)
		defer atomic.StoreInt32(&active, 0)
		hl, _ := l.Hedge.HasLock()
		if !hl {
			return // leader's job only
		}

		// Broadcast leader liveness.
		b, _ := json.Marshal(NewEvent(
			hedge.KeyValue{}, // unused
			app.EventSource,
			CtrlBroadcastLeaderLiveness,
		))

		outs := l.Hedge.Broadcast(ctx, b)
		for i, out := range outs {
			if out.Error != nil {
				glog.Errorf("[dbg] leaderLive[%v] failed: %v", i, out.Error)
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
		}

		if atomic.LoadInt32(&active) == 1 {
			continue
		}

		go do()
	}
}
