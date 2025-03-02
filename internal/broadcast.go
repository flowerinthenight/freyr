package internal

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/flowerinthenight/hedged/app"
	"github.com/golang/glog"
)

var (
	CtrlBroadcastLeaderLiveness = "CTRL_BROADCAST_LEADER_LIVENESS"

	broadcast = map[string]func(app *app.Data, e *cloudevents.Event) ([]byte, error){
		CtrlBroadcastLeaderLiveness: doBroadcastLeaderLiveness,
	}
)

func BroadcastHandler(ad interface{}, msg []byte) ([]byte, error) {
	appdata := ad.(*app.Data)
	var e cloudevents.Event
	err := json.Unmarshal(msg, &e)
	if err != nil {
		glog.Errorf("Unmarshal failed: %v", err)
		return nil, err
	}

	if _, ok := broadcast[e.Type()]; !ok {
		return nil, fmt.Errorf("failed: unsupported type: %v", e.Type())
	}

	return broadcast[e.Type()](appdata, &e)
}

func doBroadcastLeaderLiveness(app *app.Data, e *cloudevents.Event) ([]byte, error) {
	app.LeaderOk.On()
	return nil, nil
}
