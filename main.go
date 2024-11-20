package main

import (
	"context"
	"encoding/json"
	goflag "flag"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/flowerinthenight/hedge"
	"github.com/flowerinthenight/hedged/app"
	"github.com/flowerinthenight/hedged/params"
	"github.com/flowerinthenight/timedoff"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var version = "?"

var (
	rootcmd = &cobra.Command{
		Use:   "hedged",
		Short: "A generic daemon based on https://flowerinthenight/hedge/",
		Long:  `A generic daemon based on https://flowerinthenight/hedge/.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			goflag.Parse() // for cobra + glog flags
		},
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
	}

	dbstr string

	cctx = func(p context.Context) context.Context {
		return context.WithValue(p, struct{}{}, nil)
	}
)

func run(ctx context.Context, done chan error) {
	db, err := spanner.NewClient(cctx(ctx), params.DbString)
	if err != nil {
		glog.Fatal(err)
	}

	defer db.Close()
	app := &app.App{
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
		app.SpannerDb,
		podIp+":8080",
		"curmxdlock",
		"curmxd",
		"curmxd_kvstore",
		hedge.WithGroupSyncInterval(time.Second*10),
		hedge.WithLeaderHandler(app, leaderHandler),
		hedge.WithBroadcastHandler(app, broadcastHandler),
		hedge.WithLogger(log.New(io.Discard, "", 0)),
	)

	doneOp := make(chan error, 1)
	go op.Run(cctx(ctx), doneOp)
	app.Hedge = op

	// Attempt to wait for our leader before proceeding.
	func() {
		glog.Infof("attempt leader wait...")
		msg := newEvent([]byte("PING"), "hedged", "CtrlPingPong")
		b, _ := json.Marshal(msg)
		r, err := hedge.SendToLeader(ctx, app.Hedge, b)
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

	ll := leaderLive{app}
	go ll.Run(cctx(ctx)) // periodic leader liveness broadcaster

	<-ctx.Done()
	done <- nil
}

func testCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Anything, throw-away code",
		Long:  `Anything, throw-away code.`,
		Run: func(cmd *cobra.Command, args []string) {
			slog.Info("test:")
		},
		SilenceUsage: true,
	}

	cmd.Flags().SortFlags = false
	return cmd
}

func init() {
	rootcmd.Flags().SortFlags = false
	rootcmd.PersistentFlags().StringVar(&params.DbString,
		"db",
		"",
		"Spanner DB string, fmt: projects/{v}/instances/{v}/databases/{v}",
	)

	rootcmd.AddCommand(
		testCmd(),
	)

	// For cobra + glog flags.
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

func main() {
	cobra.EnableCommandSorting = false
	rootcmd.Execute()
}
