package main

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/flowerinthenight/hedged/app"
	"github.com/golang/glog"
)

var (
	ctrlBroadcastLeaderLiveness = "CTRL_BROADCAST_LEADER_LIVENESS"

	broadcast = map[string]func(app *app.App, e *cloudevents.Event) ([]byte, error){
		ctrlBroadcastLeaderLiveness: doBroadcastLeaderLiveness,
	}
)

func broadcastHandler(data interface{}, msg []byte) ([]byte, error) {
	app := data.(*app.App)
	var e cloudevents.Event
	err := json.Unmarshal(msg, &e)
	if err != nil {
		glog.Errorf("Unmarshal failed: %v", err)
		return nil, err
	}

	if _, ok := broadcast[e.Type()]; !ok {
		return nil, fmt.Errorf("failed: unsupported type: %v", e.Type())
	}

	return broadcast[e.Type()](app, &e)
}

func doBroadcastLeaderLiveness(app *app.App, e *cloudevents.Event) ([]byte, error) {
	app.LeaderOk.On()
	return nil, nil
}
