package main

import (
	goflag "flag"
	"log"

	"github.com/flowerinthenight/freyr/subcmds"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

func main() {
	root := &cobra.Command{
		Use:   "freyr",
		Short: "Companion service to https://github.com/flowerinthenight/freya/",
		Long: `Companion service to https://github.com/flowerinthenight/freya/.

The following example uses default arg values (see freyr run -h).

Example:
  # Run the first instance:
  $ freyr run --logtostderr --db projects/myproject/instances/myinstance/databases/mydb --host-port :8080 --socket-file /tmp/freyr-8080.sock

  # Run the second instance (different terminal):
  $ freyr run --logtostderr --db projects/myproject/instances/myinstance/databases/mydb --host-port :8082 --socket-file /tmp/freyr-8082.sock

  # Run the third instance (different terminal):
  $ freyr run --logtostderr --db projects/myproject/instances/myinstance/databases/mydb --host-port :8084 --socket-file /tmp/freyr-8084.sock
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
		subcmds.RunCmd(),
		subcmds.APICmd(),
		subcmds.SinkCmd(),
		subcmds.TestCmd(),
	)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine) // combine cobra and glog flags
	cobra.EnableCommandSorting = false
	root.Execute()
}
