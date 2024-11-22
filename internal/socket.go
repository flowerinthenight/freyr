package internal

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/flowerinthenight/hedged/app"
	"github.com/flowerinthenight/hedged/params"
	"github.com/golang/glog"
)

func SocketListen(ctx context.Context, appdata *app.Data, done ...chan error) {
	rmsock := func() {
		if _, err := os.Stat(params.SocketFile); err == nil {
			os.RemoveAll(params.SocketFile)
		}
	}

	defer func() {
		rmsock()
		if len(done) > 0 {
			done[0] <- nil
		}
	}()

	rmsock()
	sock, err := net.Listen("unix", params.SocketFile)
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

	glog.Infof("listen on %v", params.SocketFile)
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

		go do(conn, appdata)
	}
}

func do(conn net.Conn, appdata *app.Data) {
	defer conn.Close()
	limit := 65_536 // max 65KB
	b := make([]byte, limit)
	n, err := conn.Read(b)
	if err != nil {
		glog.Error(err)
		conn.Write([]byte(fmt.Sprintf("-ERR %v%v", err.Error(), app.CRLF)))
		return
	}

	// We use Redis protocol bulk strings for the command.
	if b[0] != '$' {
		conn.Write([]byte("-ERR invalid command" + app.CRLF))
		return
	}

	// Should be properly terminated.
	if !(b[n-2] == '\r' && b[n-1] == '\n') {
		conn.Write([]byte("-ERR invalid command" + app.CRLF))
		return
	}

	cmds := bytes.Split(b[1:n-2], []byte(app.CRLF))
	if len(cmds) < 2 {
		conn.Write([]byte("-ERR no command" + app.CRLF))
		return
	}

	// Validate length entries.
	for i := 0; i < len(cmds); i += 2 {
		if fmt.Sprintf("%d", len(cmds[i+1])) != string(cmds[i]) {
			conn.Write([]byte("-ERR invalid command" + app.CRLF))
			return
		}
	}

	switch string(cmds[1]) {
	case "SUBLDR":
		if len(cmds) < 4 {
			conn.Write([]byte("-ERR invalid command format" + app.CRLF))
			return
		}

		appdata.SubLdrMutex.Lock()
		appdata.SubLdrSocket = string(cmds[3])
		appdata.SubLdrMutex.Unlock()

		if len(cmds) >= 6 { // timeout provided
			tm, err := strconv.Atoi(string(cmds[5]))
			if err != nil {
				serr := fmt.Sprintf("-ERR %v", err.Error())
				conn.Write([]byte(serr + app.CRLF))
				return
			}

			if tm > 0 {
				appdata.SubLdrInterval.Store(int64(tm))
			}
		}
	case "UNSUBLDR":
		appdata.SubLdrMutex.Lock()
		appdata.SubLdrSocket = ""
		appdata.SubLdrMutex.Unlock()
		appdata.SubLdrInterval.Store(0)
	default:
		conn.Write([]byte("-ERR unknown command" + app.CRLF))
		return
	}

	conn.Write([]byte("-OK" + app.CRLF))
}
