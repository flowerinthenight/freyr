package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
		Use:   "run",
		Short: "Daemonize (run as service)",
		Long:  "Daemonize (run as service).",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			quit, cancel := context.WithCancel(ctx)
			done := make(chan error)
			go run(quit, done)

			go func() {
				defer cancel()
				sigch := make(chan os.Signal, 1)
				signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
				<-sigch
			}()

			<-done
		},
		SilenceUsage: true,
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&params.DbString, "db", "", "Spanner DB connection URL, fmt: projects/{v}/instances/{v}/databases/{v}")
	cmd.Flags().StringVar(&params.HostPort, "host-port", ":8080", "TCP host:port for main comms (gRPC will be :port+1), fmt: [host]<:port>")
	cmd.Flags().StringVar(&params.SocketFile, "socket-file", filepath.Join(os.TempDir(), "hedged.sock"), "Socket file for the API")
	cmd.Flags().StringVar(&params.LockTable, "lock-table", "hedged", "Spanner table for lock")
	cmd.Flags().StringVar(&params.LockName, "lock-name", "hedged", "Lock name")
	cmd.Flags().StringVar(&params.LogTable, "log-table", "hedged_kv", "Spanner table for K/V storage and semaphore meta")
	cmd.Flags().Int64Var(&params.LeaderInterval, "leader-interval", 5000, "Membership sync interval in milliseconds")
	cmd.Flags().Int64Var(&params.SyncInterval, "sync-interval", 5000, "Membership sync interval in milliseconds")
	return cmd
}

func run(ctx context.Context, done chan error) {
	glog.Infof("starting hedged on %v", params.DbString)
	db, err := spanner.NewClient(cctx(ctx), params.DbString)
	if err != nil {
		glog.Fatal(err)
	}

	defer db.Close()
	appdata := &app.Data{
		SpannerDb: db,
		LeaderOk: timedoff.New(time.Minute*30, &timedoff.CallbackT{
			Callback: func(args interface{}) {
				glog.Errorf("failed: no leader for the past 30mins?")
				// TODO: Include in the leader notification
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
		hedge.WithDuration(params.LeaderInterval),
		hedge.WithGroupSyncInterval(time.Millisecond*time.Duration(params.SyncInterval)),
		hedge.WithLeaderHandler(appdata, internal.LeaderHandler),
		hedge.WithBroadcastHandler(appdata, internal.BroadcastHandler),
	)

	doneHedge := make(chan error, 1)
	go op.Run(cctx(ctx), doneHedge)
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
			glog.Infof("confirm leader active")
		default:
			glog.Errorf("failed: no leader?")
		}
	}()

	doneSock := make(chan error, 1)
	go internal.SocketListen(cctx(ctx), appdata, doneSock)

	ln := internal.LeaderNotify{Data: appdata}
	go ln.Do(cctx(ctx)) // subscribe leader notifications

	ll := internal.LeaderLive{Data: appdata}
	go ll.Run(cctx(ctx)) // periodic leader liveness broadcaster

	<-ctx.Done() // wait signal
	<-doneHedge  // wait for hedge
	<-doneSock   // wait for socket
	done <- nil
}
