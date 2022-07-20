package telegram

import (
	"log"
	"os"
	"sync"
)

const (
	// DeveloperChatID Telegram user chat ID where messages are sent to
	DeveloperChatID          int64 = 1246830576
	DeveloperChatIDint             = int(DeveloperChatID)
	DeveloperChannelUsername       = "@jerrybillboard"
)

// jerryToken username: @cuteJerryBot name: Jerry
var jerryToken string
var jerryTokenOnce sync.Once

func JerryToken() string {
	jerryTokenOnce.Do(func() {
		var ok bool
		jerryToken, ok = os.LookupEnv("TL_JERRY_TOKEN")
		if !ok {
			log.Fatal("provide \"TL_JERRY_TOKEN\" in OS_ENV")
		}
	})
	return jerryToken
}
