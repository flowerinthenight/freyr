package main

import (
	"context"
	goflag "flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/flowerinthenight/hedged/params"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var (
	version = "?"

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

	cctx = func(p context.Context) context.Context {
		return context.WithValue(p, struct{}{}, nil)
	}
)

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
