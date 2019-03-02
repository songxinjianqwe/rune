package command

import (
	"github.com/songxinjianqwe/capsule/cli/util"
	"github.com/urfave/cli"
	"os"
)

var CreateCommand = cli.Command{
	Name:  "create",
	Usage: "create a container",
	Action: func(ctx *cli.Context) error {
		if err := util.CheckArgs(ctx, 1, util.ExactArgs); err != nil {
			return err
		}
		// 将spec转为container config对象
		// 加载factory
		// 调用factory.create
		spec, err := loadSpec()
		if err != nil {
			return err
		}
		status, err := util.LaunchContainer(ctx.Args().First(), spec, util.ContainerActCreate, true, false)
		if err != nil {
			return err
		}
		// 正常返回0，异常返回-1
		os.Exit(status)
		return nil
	},
}
