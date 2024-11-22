package internal

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/flowerinthenight/hedged/app"
	"github.com/golang/glog"
)

type LeaderNotify struct{ *app.Data }

func (l *LeaderNotify) Do(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var isLdr int
		hl, ldr, _ := l.Hedge.HasLock2()
		if hl {
			isLdr = 1
		}

		l.SubLdrMutex.Lock()
		socket := l.SubLdrSocket
		l.SubLdrMutex.Unlock()

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
			members := l.Hedge.Members()
			m := []string{ldr}
			for _, v := range members {
				if v != ldr {
					m = append(m, v)
				}
			}

			msg := fmt.Sprintf("+%d %s", isLdr, strings.Join(m, " "))
			_, err = conn.Write([]byte(msg + app.CRLF))
			if err != nil {
				glog.Errorf("Write failed: %v", err)
				return
			}
		}()

		sec := l.SubLdrInterval.Load()
		if sec == 0 {
			sec = 1 // default 1s
		}

		time.Sleep(time.Second * time.Duration(sec))
	}
}
