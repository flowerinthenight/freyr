package main

import (
	"context"
	goflag "flag"
	"log"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var (
	version = "?"

	cctx = func(p context.Context) context.Context {
		return context.WithValue(p, struct{}{}, nil)
	}
)

func main() {
	root := &cobra.Command{
		Use:   "hedged",
		Short: "A generic daemon based on https://flowerinthenight/hedge/",
		Long: `A generic daemon based on https://flowerinthenight/hedge/.

The following example uses default arg values (see hedged run -h).

Example:
  # Run the first instance:
  $ hedged run --logtostderr --db projects/myproject/instances/myinstance/databases/mydb --host-port :8080

  # Run the second instance (different terminal):
  $ hedged run --logtostderr --db projects/myproject/instances/myinstance/databases/mydb --host-port :8082

  # Run the third instance (different terminal):
  $ hedged run --logtostderr --db projects/myproject/instances/myinstance/databases/mydb --host-port :8084
`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			goflag.Parse() // combine cobra and glog flags
		},
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("See -h for subcommands.")
		},
	}

	root.PersistentFlags().SortFlags = false
	root.AddCommand(
		runCmd(),
		apiCmd(),
		sinkCmd(),
		testCmd(),
	)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine) // combine cobra and glog flags
	cobra.EnableCommandSorting = false
	root.Execute()
}
