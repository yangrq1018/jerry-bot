package coin

import (
	"fmt"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

type directioner struct {
	b *binance.Binance
	c *Coin
}

func (d directioner) String() string {
	return "set direction"
}

func (d directioner) prompt(*order.Submit) string {
	return "Buy or Sell?"
}

func resetRow() []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("CANCEL", "trade RESET"))
}

func (d directioner) keyboard(*order.Submit) [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("BUY", "trade BUY"),
			tgbotapi.NewInlineKeyboardButtonData("SELL", "trade SELL"),
		},
		resetRow(),
	}
}

func (d directioner) Next() (stage, bool) {
	return &pairer{
		b: d.b,
		c: d.c,
	}, true
}

func (d directioner) Callback() callback {
	return func(query *tgbotapi.CallbackQuery, o *order.Submit) (tgbotapi.Chattable, bool, error) {
		var side string
		fmt.Sscanf(query.Data, "trade %s", &side)
		switch side {
		case "BUY":
			o.Side = order.Buy
		case "SELL":
			o.Side = order.Sell
		default:
			return nil, true, fmt.Errorf("unknown side %q", side)
		}
		return nil, false, nil
	}
}
