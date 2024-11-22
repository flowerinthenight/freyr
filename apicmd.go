package main

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flowerinthenight/hedged/app"
	"github.com/spf13/cobra"
)

func apiCmd() *cobra.Command {
	var (
		socketFile string
	)

	cmd := &cobra.Command{
		Use:   "api <cmd>",
		Short: "Socket API client",
		Long:  `Socket API client.`,
		Run: func(cmd *cobra.Command, args []string) {
			defer func(begin time.Time) {
				log.Println("api took", time.Since(begin))
			}(time.Now())

			if len(args) == 0 {
				slog.Error("invalid command")
				return
			}

			conn, err := net.Dial("unix", socketFile)
			if err != nil {
				slog.Error("Dial failed:", "err", err)
				return
			}

			defer conn.Close()
			var api strings.Builder
			fmt.Fprintf(&api, "$")
			for _, v := range args {
				fmt.Fprintf(&api, "%d", len(v))
				fmt.Fprintf(&api, app.CRLF)
				fmt.Fprintf(&api, "%s", v)
				fmt.Fprintf(&api, app.CRLF)
			}

			_, err = conn.Write([]byte(api.String()))
			if err != nil {
				slog.Error("Write failed:", "err", err)
				return
			}

			b, err := io.ReadAll(conn)
			if err != nil {
				slog.Error("ReadAll failed:", "err", err)
				return
			}

			slog.Info("reply:",
				"str", string(b),
				"hex", fmt.Sprintf("0x%X", b),
				"dec10", fmt.Sprintf("%d", b),
			)
		},
		SilenceUsage: true,
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&socketFile,
		"socket-file",
		filepath.Join(os.TempDir(), "hedged.sock"),
		"Socket file for the API",
	)

	return cmd
}
