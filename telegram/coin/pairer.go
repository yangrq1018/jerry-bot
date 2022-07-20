package coin

import (
	"fmt"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
	"log"
)

type pairer struct {
	b *binance.Binance
	c *Coin
}

func (p pairer) String() string {
	return "set pair"
}

func (p pairer) prompt(*order.Submit) string {
	return "which pair?"
}

func (p pairer) keyboard(*order.Submit) [][]tgbotapi.InlineKeyboardButton {
	holdings, err := p.b.UpdateAccountInfo(p.c.ctx, asset.Spot)
	if err != nil {
		log.Println(err)
		return nil
	}
	var kbd [][]tgbotapi.InlineKeyboardButton
	for _, sub := range holdings.Accounts {
		for _, c := range sub.Currencies {
			if c.Total > 0 && (c.CurrencyName != currency.USDT && c.CurrencyName != currency.BUSD) {
				// 可交易币对是持仓币种和USDT
				pairUSDT := currency.NewPair(c.CurrencyName, currency.USDT)
				pairUSDT.Delimiter = "-"
				pairBUSD := currency.NewPair(c.CurrencyName, currency.BUSD)
				pairBUSD.Delimiter = "-"
				kbd = append(kbd,
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(
							pairUSDT.String(),
							fmt.Sprintf("trade pair %s", pairUSDT.String()),
						),
						tgbotapi.NewInlineKeyboardButtonData(
							pairBUSD.String(),
							fmt.Sprintf("trade pair %s", pairBUSD.String()),
						),
					),
				)
			}
		}
	}
	kbd = append(kbd, resetRow())
	return kbd
}

func (p pairer) Next() (stage, bool) {
	x := &priceSetter{b: p.b, c: p.c}
	// ugly way to get the interface working, without this "nil pointer dereference"
	x.valueSetter.name = "price"
	return x, true
}

func (p pairer) Callback() callback {
	return func(q *tgbotapi.CallbackQuery, o *order.Submit) (tgbotapi.Chattable, bool, error) {
		var pairString string
		fmt.Sscanf(q.Data, "trade pair %s", &pairString)
		pair, err := currency.NewPairFromString(pairString)
		if err != nil {
			return nil, true, err
		}
		o.Pair = pair
		s := fmt.Sprintf("order pair set ot %s", pair)
		return tgbotapi.NewEditMessageText(q.Message.Chat.ID, q.Message.MessageID, s), false, nil
	}
}
