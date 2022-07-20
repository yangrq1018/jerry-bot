package telegram

import (
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

const (
	proxyDefaultURL = "http://10.168.1.169:7890"
)

// 简化消息通知，不需要在caller创建一个bot实例，
// 而是共用一个私有的专用bot实例，这个实例不能listen，只能用来发送消息
// 环境变量TL_PROXY的值优先，否则使用默认值
func createBot() (*Bot, error) {
	var proxy string
	if tlProxy := os.Getenv("TL_PROXY"); tlProxy != "" {
		proxy = tlProxy
	} else {
		proxy = proxyDefaultURL
	}
	return NewMessageBotWithURLProxy(
		JerryToken(),
		proxy,
	)
}

var notifyBot *Bot

// guard that notifyBot is only initialized once
var notifyBotOnce sync.Once
var notifyBotErr error

// NotifyByMessage 使用默认bot发送一条文本信息
func NotifyByMessage(msg tgbotapi.MessageConfig) error {
	notifyBotOnce.Do(func() {
		notifyBot, notifyBotErr = createBot()
	})
	if notifyBotErr != nil {
		return notifyBotErr
	}
	_, err := notifyBot.Bot().Send(msg)
	return err
}

func TextMessage(msg string) {
	err := NotifyByMessage(tgbotapi.NewMessageToChannel(DeveloperChannelUsername, msg))
	if err != nil {
		logrus.Error(err)
	}
}

func extractMentionedUsername(u tgbotapi.Update, mentionEntity tgbotapi.MessageEntity) string {
	username := u.Message.Text[mentionEntity.Offset : mentionEntity.Offset+mentionEntity.Length]
	return strings.ReplaceAll(username, "@", "")
}

func ParseMentionedUsername(u tgbotapi.Update) []string {
	if u.Message.Entities == nil {
		return nil
	}
	var usernames []string
	for _, entity := range *u.Message.Entities {
		switch entity.Type {
		case "mention":
			// remove "@"
			username := extractMentionedUsername(u, entity)
			usernames = append(usernames, username)
		}
	}
	return usernames
}

// GetModuleLogger - 提供一个为 Module 使用的 logrus.Entry
// 包含 logrus.Fields
func GetModuleLogger(name string) logrus.FieldLogger {
	return logrus.WithField("module", name)
}
