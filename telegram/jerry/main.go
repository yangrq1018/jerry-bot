package main

import (
	"fmt"
	"net/url"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/yangrq1018/jerry-bot/telegram"
)

var (
	GitCommit string
	GitBranch string
	GitState  string
	BuildDate string
	Version   string
)

// JerryBot
// Production bot
// 在这里配置Jerry Bot的功能
func JerryBot() (*telegram.Bot, error) {
	bw, err := telegram.NewMessageBot(
		telegram.JerryToken(),
		telegram.SetHelp("This is a bot of Jerry, my cat"),
		telegram.SetHandleFromNow(true),
	)
	if err != nil {
		return nil, err
	}
	err = bw.RegisterCommand(
		commands()...,
	)
	if err != nil {
		return nil, err
	}
	return bw, nil
}

func main() {
	var (
		noProxy  bool
		urlProxy string
	)

	cliApp := cli.NewApp()
	cliApp.Name = "Jerry bot"
	cliApp.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "no-proxy",
			Destination: &noProxy,
		},
		&cli.StringFlag{
			Name:        "url-proxy",
			Destination: &urlProxy,
		},
	}
	cliApp.Action = func(_ *cli.Context) error {
		b, err := JerryBot()
		if err != nil {
			return err
		}

		if noProxy {
			telegram.SetNoProxy()(b)
		}
		if urlProxy != "" {
			u, err := url.Parse(urlProxy)
			if err != nil {
				return err
			}
			telegram.SetProxyFromURL(u)
		}

		err = b.Init()
		if err != nil {
			return err
		}
		b.Listen(60)
		fmt.Print("bot start listenning")
		return nil
	}
	err := cliApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
