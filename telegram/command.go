package telegram

import (
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

type Command interface {
	ID() tgbotapi.BotCommand

	// Serve 向Bot注册服务函数
	Serve(bot *Bot) error

	// Init 初始化
	// 待所有 Module 初始化完成后
	// 进行服务注册 Serve
	Init()

	Authorize() Authorizer
}
