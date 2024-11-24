package subcmds

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/flowerinthenight/freyr/app"
	"github.com/flowerinthenight/freyr/internal"
	"github.com/flowerinthenight/freyr/params"
	"github.com/flowerinthenight/hedge"
	"github.com/flowerinthenight/timedoff"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var (
	cctx = func(p context.Context) context.Context {
		return context.WithValue(p, struct{}{}, nil)
	}
)

func RunCmd() *cobra.Command {
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
	cmd.Flags().StringVar(&params.DbString, "db", "", "Spanner DB connection URL (hedge), fmt: projects/{v}/instances/{v}/databases/{v}")
	cmd.Flags().StringVar(&params.HostPort, "host-port", ":8080", "TCP host:port for hedge's main comms (gRPC will be :port+1), fmt: [host]<:port>")
	cmd.Flags().StringVar(&params.SocketFile, "socket-file", filepath.Join(os.TempDir(), "freyr.sock"), "Socket file for the API")
	cmd.Flags().StringVar(&params.LockTable, "lock-table", "freyr", "Spanner table for hedge lock")
	cmd.Flags().StringVar(&params.LockName, "lock-name", "freyr", "Lock name for hedge lock")
	cmd.Flags().StringVar(&params.LogTable, "log-table", "freyr_kv", "Spanner table for K/V storage and semaphore meta")
	cmd.Flags().Int64Var(&params.LeaderInterval, "leader-interval", 3000, "Leader check interval in milliseconds")
	cmd.Flags().Int64Var(&params.SyncInterval, "sync-interval", 3000, "Membership sync interval in milliseconds")
	return cmd
}

func run(ctx context.Context, done chan error) {
	glog.Infof("starting freyr on %v", params.DbString)
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

	host := os.Getenv("FREYR_HOST")
	port := os.Getenv("FREYR_PORT")
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
		hedge.WithLeaderCallback(appdata, func(d interface{}, m []byte) {
			ad := d.(*app.Data)
			ad.SubLdrMutex.Lock()
			socket := ad.SubLdrSocket
			ad.SubLdrMutex.Unlock()

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

				mm := strings.Split(string(m), " ")
				tmembers := ad.Hedge.Members()
				members := []string{mm[1]}
				for _, v := range tmembers {
					if v != mm[1] {
						members = append(members, v)
					}
				}

				val, _ := strconv.Atoi(mm[0])
				msg := fmt.Sprintf("+%d %s", val, strings.Join(members, " "))
				_, err = conn.Write([]byte(msg + app.CRLF))
				if err != nil {
					glog.Errorf("Write failed: %v", err)
					return
				}
			}()
		}),
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

	ll := internal.LeaderLive{Data: appdata}
	go ll.Run(cctx(ctx)) // periodic leader liveness broadcaster

	<-ctx.Done() // wait signal
	<-doneHedge  // wait for hedge
	<-doneSock   // wait for socket
	done <- nil
}
