package command

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
)

type ProvisionEvent struct {
	JsonRpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Id      int32       `json:"id"`
	Params  *Parameters `json:"params"`
}

type Data struct {
	Error   bool   `json:"is_error"`
	Output  string `json:"out"`
	Message []byte `json:"message"`
}

type Parameters struct {
	Data *Data `json:"data"`
}

type NetLogger struct {
	Address string
}

func (n *NetLogger) SendOK(p []byte, bod []byte) error {
	event := ProvisionEvent{
		JsonRpc: "2.0",
		Method:  "Event::createProvisioningEvent",
		Id:      rand.Int31(),
		Params: &Parameters{
			Data: &Data{
				Error:   false,
				Output:  string(p),
				Message: bod,
			},
		},
	}
	return n.send(&event)
}

func (n *NetLogger) SendError(p []byte, bod []byte) error {
	event := ProvisionEvent{
		JsonRpc: "2.0",
		Method:  "Event::createProvisioningEvent",
		Id:      rand.Int31(),
		Params: &Parameters{
			Data: &Data{
				Error:   true,
				Output:  string(p),
				Message: bod,
			},
		},
	}
	return n.send(&event)
}

func (n *NetLogger) send(p *ProvisionEvent) error {
	post, _ := json.Marshal(p)
	_, err := http.Post(n.Address, "encoding/json", bytes.NewBuffer(post))
	if err != nil {
		return err
	}
	return nil

}
