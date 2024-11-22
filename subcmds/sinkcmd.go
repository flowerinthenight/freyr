package subcmds

import (
	"context"
	"io"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

func SinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sink <path/to/socket>",
		Short: "Sink server for leader notifications",
		Long:  "Sink server for leader notifications.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				glog.Error("no socket file provided")
				return
			}

			ctx := context.Background()
			quit, cancel := context.WithCancel(ctx)
			done := make(chan error)
			go sink(quit, args[0], done)

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
	return cmd
}

func sink(ctx context.Context, socket string, done chan error) {
	rmsock := func() {
		if _, err := os.Stat(socket); err == nil {
			os.RemoveAll(socket)
		}
	}

	defer func() {
		rmsock()
		done <- nil
	}()

	rmsock()
	sock, err := net.Listen("unix", socket)
	if err != nil {
		glog.Error(err)
		return
	}

	var closed atomic.Int32

	go func() {
		<-ctx.Done()
		closed.Store(1)
		sock.Close() // terminate our loop below
	}()

	glog.Infof("listen on %v", socket)
	defer sock.Close()

	for {
		conn, err := sock.Accept()
		if closed.Load() == 1 {
			return
		}

		if err != nil {
			glog.Error(err)
			continue
		}

		go func(nc net.Conn) {
			b, err := io.ReadAll(nc)
			if err != nil {
				glog.Error(err)
				return
			}

			glog.Infof("notification: %v", string(b))
		}(conn)
	}
}
