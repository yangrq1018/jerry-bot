//go:build docker || linux
// +build docker linux

package main

import (
	"github.com/yangrq1018/jerry-bot/telegram"
	"github.com/yangrq1018/jerry-bot/telegram/app"
	"github.com/yangrq1018/jerry-bot/telegram/coin"
	"github.com/yangrq1018/jerry-bot/telegram/tweet"
	"github.com/yangrq1018/jerry-bot/telegram/weather"
)

func commands() []telegram.Command {
	return append(
		weather.Commands(),
		tweet.Command(),
		coin.NewCoin(),
		app.ServerStatsCommand(),
		app.SplatoonCommand(),
		app.ZhihuCommand(),
		app.TexasPlayCommand(),
		app.TexasHistogramCommand(),
	)
}
