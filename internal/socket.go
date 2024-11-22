package internal

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"strings"
	"sync/atomic"

	"github.com/flowerinthenight/hedged/app"
	"github.com/flowerinthenight/hedged/params"
	"github.com/golang/glog"
)

func SocketListen(ctx context.Context, app *app.App, done ...chan error) {
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

		go do(conn)
	}
}

func do(conn net.Conn) {
	defer conn.Close()
	glog.Infof("connected: %s", conn.RemoteAddr().Network())

	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, conn)
	if err != nil {
		glog.Error(err)
		return
	}

	s := strings.ToUpper(buf.String())

	buf.Reset()
	buf.WriteString(s)

	_, err = io.Copy(conn, buf)
	if err != nil {
		glog.Error(err)
		return
	}

	glog.Info("<<< ", s)
}
