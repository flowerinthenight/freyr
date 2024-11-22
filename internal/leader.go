package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/flowerinthenight/hedge"
	"github.com/flowerinthenight/hedged/app"
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

type LeaderNotify struct{ *app.Data }

func (l *LeaderNotify) Do(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var ldr int
		hl, _ := l.Hedge.HasLock()
		if hl {
			ldr = 1
		}

		l.SubLdrMutex.Lock()
		socket := l.SubLdrSocket
		l.SubLdrMutex.Unlock()

		func() {
			if socket == "" {
				return
			}

			conn, err := net.Dial("unix", socket)
			if err != nil {
				glog.Errorf("Dial failed: %v", err)
				return
			}

			defer conn.Close()
			msg := fmt.Sprintf("+%d", ldr)
			_, err = conn.Write([]byte(msg + app.CRLF))
			if err != nil {
				glog.Errorf("Write failed: %v", err)
				return
			}
		}()

		sec := l.SubLdrInterval.Load()
		if sec == 0 {
			sec = 1 // default 1s
		}

		time.Sleep(time.Second * time.Duration(sec))
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
