package app

import (
	"github.com/yangrq1018/jerry-bot/telegram"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

type SimpleCommand struct {
	name,
	description string
	handle telegram.HandleFunc[tgbotapi.Update]
}

func (s SimpleCommand) ID() tgbotapi.BotCommand {
	return tgbotapi.BotCommand{
		Command:     s.name,
		Description: s.description,
	}
}

func (s SimpleCommand) Serve(bot *telegram.Bot) error {
	bot.Match(s).Subscribe(s.handle)
	return nil
}

func (s SimpleCommand) Init() {}

func (s SimpleCommand) Authorize() telegram.Authorizer {
	return telegram.PolicyAllow
}
