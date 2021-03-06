package util

import (
	"fmt"
	"github.com/urfave/cli"
	"os"
)

const (
	ExactArgs = iota
	MinArgs
	MaxArgs
)

func CheckArgs(context *cli.Context, expected, checkType int) error {
	var err error
	cmdName := context.Command.Name
	switch checkType {
	case ExactArgs:
		if context.NArg() != expected {
			err = fmt.Errorf("%s: %q requires exactly %d argument(s)", os.Args[0], cmdName, expected)
		}
	case MinArgs:
		if context.NArg() < expected {
			err = fmt.Errorf("%s: %q requires a minimum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	case MaxArgs:
		if context.NArg() > expected {
			err = fmt.Errorf("%s: %q requires a maximum of %d argument(s)", os.Args[0], cmdName, expected)
		}
	}
	if err != nil {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(context, cmdName)
		return err
	}
	return nil
}

func CheckNoFile(fileName string) error {
	_, err := os.Stat(fileName)
	if err == nil {
		return fmt.Errorf("file %s exists. Remove it first", fileName)
	}
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}
