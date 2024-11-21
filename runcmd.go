package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/flowerinthenight/hedge"
	"github.com/flowerinthenight/hedged/app"
	"github.com/flowerinthenight/hedged/internal"
	"github.com/flowerinthenight/hedged/params"
	"github.com/flowerinthenight/timedoff"
	"github.com/golang/glog"
)

func run(ctx context.Context, done chan error) {
	db, err := spanner.NewClient(cctx(ctx), params.DbString)
	if err != nil {
		glog.Fatal(err)
	}

	defer db.Close()
	appdata := &app.App{
		Mutex:     &sync.Mutex{},
		SpannerDb: db,
		LeaderOk: timedoff.New(time.Minute*30, &timedoff.CallbackT{
			Callback: func(args interface{}) {
				glog.Errorf("failed: no leader for the past 30mins?")
			},
		}),
	}

	podIp := os.Getenv("K8S__MY_POD_IP") // via k8s downward API
	op := hedge.New(
		appdata.SpannerDb,
		podIp+":8080",
		"curmxdlock",
		"curmxd",
		"curmxd_kvstore",
		hedge.WithGroupSyncInterval(time.Second*10),
		hedge.WithLeaderHandler(appdata, internal.LeaderHandler),
		hedge.WithBroadcastHandler(appdata, internal.BroadcastHandler),
		hedge.WithLogger(log.New(io.Discard, "", 0)),
	)

	doneOp := make(chan error, 1)
	go op.Run(cctx(ctx), doneOp)
	appdata.Hedge = op

	// Attempt to wait for our leader before proceeding.
	func() {
		glog.Infof("attempt leader wait...")
		msg := internal.NewEvent([]byte("PING"), app.EventSource, internal.CtrlPingPong)
		b, _ := json.Marshal(msg)
		r, err := hedge.SendToLeader(ctx, appdata.Hedge, b)
		if err != nil {
			return
		}

		switch {
		case string(r) == "PONG":
			glog.Infof("confirm: leader active")
		default:
			glog.Errorf("failed: no leader?")
		}
	}()

	ll := internal.LeaderLive{App: appdata}
	go ll.Run(cctx(ctx)) // periodic leader liveness broadcaster

	<-ctx.Done()
	done <- nil
}
