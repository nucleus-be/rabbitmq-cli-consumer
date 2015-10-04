package command

import (
	"strings"
)

func NewCliFactory(baseCmd string) Factory {
	var pcs []string
	if split := strings.Split(baseCmd, " "); len(split) > 1 {
		baseCmd, pcs = split[0], split[1:]
	}
	return &CommandFactory{
		Cmd:  baseCmd,
		Args: pcs,
	}
}

func NewHttpFactory(url, tpe string) Factory {
	return &HttpFactory{
		Url:      url,
		BodyType: tpe,
	}
}
