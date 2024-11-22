package subcmds

import (
	"log"
	"strconv"

	"github.com/spf13/cobra"
)

func TestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Anything, throw-away code",
		Long:  "Anything, throw-away code.",
		Run: func(cmd *cobra.Command, args []string) {
			val, _ := strconv.Atoi("invalid")
			log.Println(val)
		},
		SilenceUsage: true,
	}

	cmd.Flags().SortFlags = false
	return cmd
}
