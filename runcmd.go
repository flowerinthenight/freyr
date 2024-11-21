package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/flowerinthenight/hedge"
	"github.com/flowerinthenight/hedged/app"
	"github.com/flowerinthenight/hedged/internal"
	"github.com/flowerinthenight/hedged/params"
	"github.com/flowerinthenight/timedoff"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "run",
		Short:  "Daemonize (run as service)",
		Long:   "Daemonize (run as service).",
		PreRun: func(cmd *cobra.Command, args []string) {},
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			quit, cancel := context.WithCancel(ctx)
			done := make(chan error)
			go run(quit, done)

			go func() {
				defer cancel()
				sigch := make(chan os.Signal, 1)
				signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
				glog.Infof("sigterm: %v", <-sigch)
			}()

			<-done
		},
		SilenceUsage: true,
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&params.DbString,
		"db",
		"",
		"Spanner DB connection URL, fmt: projects/{v}/instances/{v}/databases/{v}",
	)

	cmd.Flags().StringVar(&params.HostPort,
		"hostport",
		":8080",
		"TCP host:port for main comms (gRPC will be :port+1), fmt: [host]<:port>",
	)

	cmd.Flags().StringVar(&params.SocketFile,
		"socketfile",
		filepath.Join(os.TempDir(), "hedged.sock"),
		"Socket file for the API",
	)

	cmd.Flags().StringVar(&params.LockTable, "locktable", "hedged", "Spanner table for lock")
	cmd.Flags().StringVar(&params.LockName, "lockname", "hedged", "Lock name")
	cmd.Flags().StringVar(&params.LogTable, "logtable", "hedged_kv", "Spanner table for K/V storage and semaphore meta")
	cmd.Flags().IntVar(&params.SyncInterval, "syncinterval", 10, "Membership sync interval in seconds")
	return cmd
}

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

	host := os.Getenv("HEDGED_HOST")
	port := os.Getenv("HEDGED_PORT")
	hp := strings.Split(params.HostPort, ":")
	if len(hp) == 2 {
		if hp[0] != "" {
			host = hp[0]
		}

		if hp[1] != "" {
			port = hp[1]
		}
	}

	op := hedge.New(
		appdata.SpannerDb,
		host+":"+port,
		params.LockTable,
		params.LockName,
		params.LogTable,
		hedge.WithGroupSyncInterval(time.Second*time.Duration(params.SyncInterval)),
		hedge.WithLeaderHandler(appdata, internal.LeaderHandler),
		hedge.WithBroadcastHandler(appdata, internal.BroadcastHandler),
		// hedge.WithLogger(log.New(io.Discard, "", 0)),
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
