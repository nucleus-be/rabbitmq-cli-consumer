package command

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"net/http"
	"os/exec"
)

type Command interface {
	Execute() ([]byte, error)
}

type Factory interface {
	Create(body []byte) Command
}

type CommandFactory struct {
	Cmd  string
	Args []string
}

type HttpFactory struct {
	Url      string
	BodyType string
}

type CliCommand struct {
	*exec.Cmd
}

type HttpCommand struct {
	Url  string
	Type string
	Body []byte
}

func (ccmd *CliCommand) Execute() ([]byte, error) {
	return ccmd.CombinedOutput()
}

func (hcmd *HttpCommand) Execute() ([]byte, error) {
	resp, err := http.Post(hcmd.Url, hcmd.Type, bytes.NewBuffer(hcmd.Body))
	if err != nil {
		return []byte{}, err
	}
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		return []byte{}, errors.New(string(body))
	}
	return []byte{}, nil
}

func (me *HttpFactory) Create(body []byte) Command {
	return &HttpCommand{
		Url:  me.Url,
		Type: me.BodyType,
		Body: body,
	}
}

func (me *CommandFactory) Create(body []byte) Command {
	return &CliCommand{exec.Command(me.Cmd, append(me.Args, base64.StdEncoding.EncodeToString(body))...)}
}
