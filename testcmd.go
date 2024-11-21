package main

import (
	"log/slog"

	"github.com/spf13/cobra"
)

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
