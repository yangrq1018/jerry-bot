package weather

import (
	"github.com/yangrq1018/jerry-bot/telegram"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

type forecast struct{}

func (f forecast) Authorize() telegram.Authorizer {
	return telegram.PolicyAllow
}

func (f forecast) ID() tgbotapi.BotCommand {
	return tgbotapi.BotCommand{
		Command:     "forecast",
		Description: "七日预报",
	}
}

func (f *forecast) Serve(bot *telegram.Bot) error {
	bot.Match(f).Subscribe(f.handle)
	return nil
}

func (f forecast) Init() {
}

func (f forecast) location(b *telegram.Bot, msg tgbotapi.Message) error {
	loc := &Coordinate{
		Lng: msg.Location.Longitude,
		Lat: msg.Location.Latitude,
	}
	w, err := GetOneCallWeather(loc)
	if err != nil {
		return err
	}
	// reverse Geocode to find country code
	address, err := mapboxCoder.ReverseGeocode(loc.Lat, loc.Lng)
	if err != nil {
		return err
	}
	b.ReplyTo(msg, w.ReportEveryDay(address.Country+" "+address.City, address.CountryCode))
	return nil
}

func (f forecast) handle(b *telegram.Bot, u tgbotapi.Update) error {
	addr := u.Message.CommandArguments()
	if addr == "" {
		// send button
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, "/forecast <查询地址>")
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButtonLocation("send my location")),
		)
		_, err := b.Bot().Send(msg)
		// register location handler
		b.LocationEvent.Subscribe(f.location)
		return err
	}
	// 查找Geocode
	loc, err := GetGeocodeAuto(addr)
	if err != nil {
		return err
	}
	w, err := GetOneCallWeather(loc)
	if err != nil {
		return err
	}

	if addrRes, err := BaiduReverseGeocodeDomestic(loc); err == nil && addrRes.FormattedAddress != "" {
		addr = addrRes.FormattedAddress
	}

	// reverse Geocode to find country code
	var countryCode string
	if isChinese(addr) {
		countryCode = "CN"
	} else {
		// reverse geocode
		address, err := mapboxCoder.ReverseGeocode(loc.Latitude(), loc.Longitude())
		if err != nil {
			return err
		}
		countryCode = address.CountryCode
	}
	b.ReplyTo(*u.Message, w.ReportEveryDay(addr, countryCode))
	return nil
}
