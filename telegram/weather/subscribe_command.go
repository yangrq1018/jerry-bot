package weather

import (
	"fmt"

	"github.com/yangrq1018/jerry-bot/gcloud"
	"github.com/yangrq1018/jerry-bot/telegram"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
	"github.com/zsefvlol/timezonemapper"

	"os"
	"regexp"
	"time"
)

var logger = telegram.GetModuleLogger("weather")

// TelegramLocationWrapper is a wrapper around tgbotapi.Location
// that implements Location
type TelegramLocationWrapper struct {
	*tgbotapi.Location
}

func (t TelegramLocationWrapper) Latitude() float64 {
	return t.Location.Latitude
}

func (t TelegramLocationWrapper) Longitude() float64 {
	return t.Location.Longitude
}

type subscribe struct {
	reminder *Reminder
}

func (s subscribe) Authorize() telegram.Authorizer {
	return telegram.PolicyAllow
}

func (s subscribe) ID() tgbotapi.BotCommand {
	return tgbotapi.BotCommand{
		Command:     "subscribe",
		Description: "订阅每日天气提醒，如果换了地址可以再次点击更新地址",
	}
}

func (s *subscribe) Serve(bot *telegram.Bot) error {
	logger.Infof("loading subscription from GCS")
	bot.Match(s).Subscribe(s.subscribe)
	bot.TeardownEvent.Subscribe(s.teardown)
	re := regexp.MustCompile("subscribe(.*)")
	bot.UpdateEvent.Subscribe(func(b *telegram.Bot, u tgbotapi.Update) error {
		if re.MatchString(u.Message.Command()) {
			return s.subscribe(b, u)
		}
		return nil
	})

	s.reminder.bot = bot
	return s.reminder.loadSubscribersFromCloud()
}

func (s *subscribe) Init() {}

func (s subscribe) subscribeLocation(b *telegram.Bot, msg tgbotapi.Message) error {
	if !s.reminder.IsRegistered(msg.Chat.ID) {
		err := s.reminder.subscribeFromLocationMessage(msg)
		if err != nil {
			return err
		}
		b.ReplyTo(msg, "(first subscribe)成功登记了你的定位, 下次通知在\n%s.",
			nextJobRunTime(s.reminder.GetScheduler(msg.Chat.ID)).Format(time.RFC3339))
	} else {
		// subscribed already, update timezone and location
		scheduler := s.reminder.GetScheduler(msg.Chat.ID)
		sub := s.reminder.Get(msg.Chat.ID)
		sub.Location = TelegramLocationWrapper{msg.Location}
		before := nextJobRunTime(scheduler)
		// clear them remake jobs
		scheduler.Clear()
		scheduler.ChangeLocation(timezoneFromCoordinates(sub.Location))
		err := addRunEverydayJob(scheduler, sub.GetJobFunc(s.reminder.bot), getCronString(8, 30))
		if err != nil {
			return err
		}
		logger.Infof("user %q changed his location, coord: %+v, tz: %s", msg.From.UserName, sub.Location, sub.zone)
		b.ReplyTo(msg, "成功登记了你的定位, 下次通知在\n%s。\n更改前是%s", nextJobRunTime(scheduler).Format(time.RFC3339), before.Format(time.RFC3339))
	}
	return nil
}

func (s subscribe) subscribe(b *telegram.Bot, u tgbotapi.Update) error {
	switch u.Message.Command() {
	case "subscribe_send":
		jf := s.reminder.GetJobFunc(u.Message.Chat.ID)
		if jf != nil {
			jf()
		}
		return nil
	case "subscribe_time":
		scheduler := s.reminder.GetScheduler(u.Message.Chat.ID)
		if scheduler == nil {
			return nil
		}
		b.ReplyTo(*u.Message, "the next run time is %s", nextJobRunTime(scheduler).Format("2006-01-02 15:04:05"))
		return nil
	}

	if args := u.Message.CommandArguments(); args != "" {
		from := *u.Message.From
		sub := s.reminder.Get(int64(from.ID))
		if sub != nil {
			// purge html tags
			sub.PreferredName = regexp.MustCompile("[<>]").ReplaceAllString(args, "")
			b.ReplyTo(*u.Message, "修改了%d的PreferredName为: %s", from.ID, sub.PreferredName)
		} else {
			b.ReplyTo(*u.Message, "没有找到%d的订阅，请先发送/subscribe以订阅", from.UserName)
		}
		return nil
	}
	// location can be sent in private chats only
	if u.Message.Chat.Type == "private" {
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "Alright, I need your location")
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButtonLocation("send my location")), // prompt for user location
		)
		_, err := b.Bot().Send(msg)
		b.LocationEvent.Subscribe(s.subscribeLocation)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s subscribe) teardown(*telegram.Bot, os.Signal) error {
	// save subscription objects to cloud storage
	subs := s.reminder.PersistentSubscribers()
	for i := range subs {
		logger.Infof("persist subscription %s", subs[i].User.UserName)
	}
	err := gcloud.SaveObject(gcloudObjectKey, subs)
	if err != nil {
		return fmt.Errorf("failed to save subscribers to GCS: %v", err)
	}
	return nil
}

func timezoneFromCoordinates(loc Location) *time.Location {
	zone := timezonemapper.LatLngToTimezoneString(loc.Latitude(), loc.Longitude())
	if zone == "" {
		logger.Info("warning: cannot determine timezone from geocode, will use UTC")
	}
	tz, err := time.LoadLocation(zone)
	if err != nil {
		panic(err)
	}
	return tz
}
