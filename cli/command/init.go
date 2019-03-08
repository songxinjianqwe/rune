package command

import (
	"github.com/sirupsen/logrus"
	"github.com/songxinjianqwe/capsule/libcapsule"
	_ "github.com/songxinjianqwe/capsule/libcapsule/nsenter"
	"github.com/urfave/cli"
)

/*
仅限内部调用，有可能是init process，也有可能是exec process。
*/
var InitCommand = cli.Command{
	Name:  "init",
	Usage: "init a container(execute init/exec process)",
	Action: func(ctx *cli.Context) error {
		factory, err := libcapsule.NewFactory(false)
		if err != nil {
			return err
		}
		if err := factory.StartInitialization(); err != nil {
			logrus.WithField("init", true).Errorf("init failed, err: %s", err.Error())
		}
		return nil
	},
}
