package command

import (
	"log"
	"os/exec"
)

type CommandExecuter struct {
	errLogger *log.Logger
	infLogger *log.Logger
	netLogger *NetLogger
}

func New(errLogger, infLogger *log.Logger) *CommandExecuter {
	netLogger := new(NetLogger)
	netLogger.Address = Cconf.Logs.Rpc

	return &CommandExecuter{
		errLogger: errLogger,
		infLogger: infLogger,
		netLogger: netLogger,
	}
}

func (me CommandExecuter) Execute(cmd *exec.Cmd, body []byte) bool {
	me.infLogger.Println("Processing message...")
	out, err := cmd.CombinedOutput()

	//log output php script to info
	me.infLogger.Printf("Output php: %s\n", string(out[:]))

	if err != nil {
		me.infLogger.Println("Failed. Check error log for details.")
		me.errLogger.Printf("Failed: %s\n", string(out[:]))
		me.errLogger.Printf("Error: %s\n", err)
		me.netLogger.SendError([]byte(err.Error()))
		me.netLogger.SendError(out[:])
		return false
	}

	me.infLogger.Println("Processed!")
	me.netLogger.SendOK(out)

	return true
}
