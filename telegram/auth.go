package telegram

import tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"

type Authorizer interface {
	Validate(u tgbotapi.Update) (ok bool, reason string)
}

type allow struct{}

func (a allow) Validate(u tgbotapi.Update) (ok bool, reason string) {
	return true, " "
}

var PolicyAllow = allow{}
