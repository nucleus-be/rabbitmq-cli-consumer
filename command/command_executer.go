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

	if len(Cconf.Logs.Rpc) > 1 {
		netLogger := new(NetLogger)
		netLogger.Address = Cconf.Logs.Rpc

		return &CommandExecuter{
			errLogger: errLogger,
			infLogger: infLogger,
			netLogger: netLogger,
		}
	}

	return &CommandExecuter{
		errLogger: errLogger,
		infLogger: infLogger,
	}

}

func (me CommandExecuter) Execute(cmd *exec.Cmd, body []byte) bool {
	me.infLogger.Println("Processing message...")
	me.infLogger.Printf("Cmd: %s\n", cmd.Path)
	out, err := cmd.CombinedOutput()

	//log output php script to info
	me.infLogger.Printf("Output php: %s\n", string(out[:]))

	if err != nil {
		me.infLogger.Println("Failed. Check error log for details.")
		me.errLogger.Printf("Failed: %s\n", string(out[:]))
		me.errLogger.Printf("Error: %s\n", err)

		if len(Cconf.Logs.Rpc) > 1 {
			me.infLogger.Println("rpc parameters: %s", Cconf.Logs.Rpc)
			me.netLogger.Send([]byte(err.Error()), body[:], true)
			me.netLogger.Send(out[:], body[:], true)
		}

		return false
	}

	me.infLogger.Println("Processed!")

	if len(Cconf.Logs.Rpc) > 1 {
		me.netLogger.Send(out[:], body[:], false)
	}

	return true
}
