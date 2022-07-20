package weather

import (
	"fmt"
	"net/url"

	"github.com/yangrq1018/jerry-bot/telegram"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

type current struct {
	reminder *Reminder
}

func (c current) Authorize() telegram.Authorizer {
	return telegram.PolicyAllow
}

func (c current) ID() tgbotapi.BotCommand {
	return tgbotapi.BotCommand{Command: "weather", Description: "查询天气"}
}

func (c current) Serve(bot *telegram.Bot) error {
	bot.Match(c).Subscribe(c.handle)
	return nil
}

func (c current) Init() {}

func (c current) handle(b *telegram.Bot, u tgbotapi.Update) error {
	var loc Location
	var addr string
	var err error
	args := u.Message.CommandArguments()
	var target *Subscription
	if args == "" {
		// look up the sender's location
		target = c.reminder.Get(int64(u.Message.From.ID))
		if target != nil {
			loc = target.Location
		} else {
			return fmt.Errorf("provide an address please")
		}
	} else {
		// any mentioned user ?
		if user := c.getLocationOfMentionedUser(u); user != nil {
			loc = user.Location
		} else {
			// take the args as addr
			addr = args
			// 查找Geocode
			loc, err = GetGeocodeAuto(addr)
			if err != nil {
				return err
			}
		}
	}
	if a, err := BaiduReverseGeocodeDomestic(loc); err == nil && a.FormattedAddress != "" {
		if addr != "" {
			addr = addr + " - " + a.FormattedAddress
		} else {
			addr = a.FormattedAddress
		}
	}
	w, err := GetCurrentWeather(loc)
	if err != nil {
		return err
	}
	text := w.Report(addr, loc)
	if len(w.Weather) > 0 {
		iconURL, err := url.Parse(w.Weather[0].IconURL())
		if err != nil {
			return fmt.Errorf("cannot parse URL of icon: %v\n", err)
		}
		msg := tgbotapi.NewPhotoUpload(u.Message.Chat.ID, *iconURL)
		msg.Caption = text
		_, err = b.Bot().Send(msg)
		if err != nil {
			return fmt.Errorf("cannot send icon: %v\n", err)
		}
	} else {
		b.ReplyTo(*u.Message, text)
	}
	return nil
}

// 如果消息中含有"@"某个用户，返回第一个匹配到订阅的用户所在地址
func (c current) getLocationOfMentionedUser(u tgbotapi.Update) *Subscription {
	if u.Message.Entities == nil {
		return nil
	}
	mentioned := telegram.ParseMentionedUsername(u)
	if mentioned == nil {
		return nil
	}
	for _, username := range mentioned {
		target := c.reminder.GetTargetByUsername(username)
		if target != nil {
			return target
		}
	}
	return nil
}
