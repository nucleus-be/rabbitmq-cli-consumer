package command

import (
	"log"
)

type Runner interface {
	Run(cmd Command) bool
}

type CommandExecuter struct {
	errLogger *log.Logger
	infLogger *log.Logger
}

func NewRunner(errLogger, infLogger *log.Logger) Runner {
	return CommandExecuter{
		errLogger: errLogger,
		infLogger: infLogger,
	}
}

func (me CommandExecuter) Run(cmd Command) bool {
	me.infLogger.Println("Processing message...")
	out, err := cmd.Execute()

	//log output php script to info
	me.infLogger.Printf("Output php: %s\n", string(out[:]))

	if err != nil {
		me.infLogger.Println("Failed. Check error log for details.")
		me.errLogger.Printf("Failed: %s\n", string(out[:]))
		me.errLogger.Printf("Error: %s\n", err)
		return false
	}

	me.infLogger.Println("Processed!")

	return true
}
