package coin

import (
	"fmt"
	"log"
	"strconv"

	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

// numeric key pad
type valueSetter struct {
	name  string
	value string
	ok    bool // signal input is complete
}

// action: append/set
func (v *valueSetter) callback(r, action string) string {
	return fmt.Sprintf("trade %s %s %s", v.name, action, r)
}

func (v *valueSetter) digitRow(s []string) []tgbotapi.InlineKeyboardButton {
	var row []tgbotapi.InlineKeyboardButton
	for _, r := range s {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(
			r,
			v.callback(r, "append"),
		))
	}
	return row
}

func (v *valueSetter) float() (float64, error) {
	return strconv.ParseFloat(v.value, 64)
}

func (v *valueSetter) String() string {
	return fmt.Sprintf("set %s", v.name)
}

func (v *valueSetter) keyboard(*order.Submit) [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		v.digitRow([]string{"1", "2", "3"}),
		v.digitRow([]string{"4", "5", "6"}),
		v.digitRow([]string{"7", "8", "9"}),
		v.digitRow([]string{".", "0", backspaceBtn}),
		v.digitRow([]string{okBtn}),
	}
}

const (
	backspaceBtn = "<-"
	okBtn        = "OK"
)

func (v *valueSetter) Next() (stage, bool) {
	return nil, false
}

func (v *valueSetter) Callback() callback {
	return func(q *tgbotapi.CallbackQuery, o *order.Submit) (tgbotapi.Chattable, bool, error) {
		var c, action string
		_, err := fmt.Sscanf(q.Data, fmt.Sprintf("trade %s %%s %%s", v.name), &action, &c)
		if err != nil {
			return nil, false, err
		}
		switch action {
		case "append":
			switch c {
			case okBtn:
				v.ok = true
			case backspaceBtn:
				// 退格
				if len(v.value) >= 1 {
					v.value = v.value[:len(v.value)-1]
				}
			default:
				v.value += c
			}
		case "set":
			v.value = c
			v.ok = true
		}
		return nil, false, nil
	}
}

// price
type priceSetter struct {
	b *binance.Binance
	c *Coin
	valueSetter
	//bestPrice *binance.BestPrice
}

func stringifyPrice(p float64, precision int) string {
	return strconv.FormatFloat(p, 'f', precision, 64)
}

func (p *priceSetter) priceOnSide(o *order.Submit, f func(price binance.BestPrice) float64) (float64, error) {
	best, err := p.b.GetBestPrice(p.c.ctx, o.Pair)
	if err != nil {
		return 0, err
	}
	return f(best), nil
}

// bestPrice 查询本方的最优成交价格，也就是order book上的consensus
// 如果是买单，返回Bid1（最强竞争对手的出价）；如果是卖单，返回Ask1。
func (p *priceSetter) bestPrice(o *order.Submit) (float64, error) {
	switch o.Side {
	case order.Buy:
		return p.priceOnSide(o, func(price binance.BestPrice) float64 {
			return price.BidPrice
		})
	case order.Sell:
		return p.priceOnSide(o, func(price binance.BestPrice) float64 {
			return price.AskPrice
		})
	}
	return 0, fmt.Errorf("unknown side %v", o.Side)
}

// bestPriceOpponentSide 查询对手方的最优订单成交价格
// 如果是买单，返回Ask1；如果是卖单价，返回Bid1。
// 按bestPriceOpponentSide的价格下单，可以马上成交
func (p *priceSetter) bestPriceOpponentSide(o *order.Submit) (float64, error) {
	switch o.Side {
	case order.Buy:
		return p.priceOnSide(o, func(price binance.BestPrice) float64 {
			return price.AskPrice
		})
	case order.Sell:
		return p.priceOnSide(o, func(price binance.BestPrice) float64 {
			return price.BidPrice
		})
	}
	return 0, fmt.Errorf("unknown side %v", o.Side)
}

func (p *priceSetter) prompt(o *order.Submit) string {
	if p.ok {
		return ""
	}
	best, err := p.b.GetBestPrice(p.c.ctx, o.Pair)
	if err != nil {
		return err.Error()
	}
	s := fmt.Sprintf(`%s trades at
- Bid %s
- Ask %s`, o.Pair, p.c.FormatPrice(best.BidPrice, o.Pair), p.c.FormatPrice(best.AskPrice, o.Pair))
	if p.value == "" {
		var side string
		switch o.Side {
		case order.Buy:
			side = "bid"
		case order.Sell:
			side = "ask"
		}
		s += "\n" + fmt.Sprintf("your %s?", side)
	} else {
		s += "\n" + fmt.Sprintf("%s: %s", p.name, p.value)
	}
	return s
}

func (p *priceSetter) keyboard(o *order.Submit) [][]tgbotapi.InlineKeyboardButton {
	if p.ok {
		return nil
	}
	kbd := p.valueSetter.keyboard(o)
	// add one more row
	bestThisSide, err := p.bestPrice(o)
	if err != nil {
		return nil
	}
	bestOpponentSide, err := p.bestPriceOpponentSide(o)
	if err != nil {
		return nil
	}
	bestThisSideString, bestOpponentSideString := p.c.FormatPrice(bestThisSide, o.Pair), p.c.FormatPrice(bestOpponentSide, o.Pair)
	kbd = append(kbd,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("本方价格%s", bestThisSideString),
				p.callback(bestThisSideString, "set"),
			)),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("对手方价格%s", bestOpponentSideString),
				p.callback(bestOpponentSideString, "set"),
			)),
		resetRow(),
	)
	return kbd
}

func (p *priceSetter) Callback() callback {
	if p.ok {
		return nil
	}
	return func(v *tgbotapi.CallbackQuery, o *order.Submit) (tgbotapi.Chattable, bool, error) {
		msg, done, err := p.valueSetter.Callback()(v, o)
		if err == nil && p.ok {
			o.Price, err = p.float()
			log.Printf("set %s to %v", p.name, o.Price)
		}
		return msg, done, err
	}
}

func (p *priceSetter) Next() (stage, bool) {
	if p.ok {
		q := &quantitySetter{
			b: p.b,
			c: p.c,
		}
		q.valueSetter.name = "quantity"
		return q, false
	}
	return p, true
}

// quantity
type quantitySetter struct {
	valueSetter
	b             *binance.Binance
	c             *Coin
	positionBase  *account.Balance
	positionQuote *account.Balance
}

func (q *quantitySetter) prompt(o *order.Submit) string {
	// holding
	var s string
	hold, _ := q.b.UpdateAccountInfo(q.c.ctx, asset.Spot)
	for _, acc := range hold.Accounts {
		for i, c := range acc.Currencies {
			// don't reference c, loop variable is not redeclared
			if c.CurrencyName == o.Pair.Base {
				q.positionBase = &acc.Currencies[i]
			}
			if c.CurrencyName == o.Pair.Quote {
				q.positionQuote = &acc.Currencies[i]
			}
		}
	}
	if q.positionBase == nil || q.positionQuote == nil {
		log.Println("failed to find base/quote position")
		return ""
	}
	switch o.Side {
	case order.Buy:
		// spend quote coin
		s = fmt.Sprintf("you have %.6f %s\n", q.positionQuote.Total, q.positionQuote.CurrencyName)
	case order.Sell:
		// spend base coin
		s = fmt.Sprintf("you have %.6f %s\n", q.positionBase.Total, q.positionBase.CurrencyName)
	}
	if q.value == "" {
		var side string
		switch o.Side {
		case order.Buy:
			side = "buy"
		case order.Sell:
			side = "sell"
		}
		s += fmt.Sprintf("%s how much?", side)
	} else {
		s += fmt.Sprintf("%s: %s", q.name, q.value)
	}
	return s
}

func (q *quantitySetter) Callback() callback {
	if q.ok {
		return nil
	}
	return func(v *tgbotapi.CallbackQuery, o *order.Submit) (tgbotapi.Chattable, bool, error) {
		msg, done, err := q.valueSetter.Callback()(v, o)
		if err == nil && q.ok {
			o.Amount, err = q.float()
			// handle lot size
			log.Printf("set %s to %v", q.name, o.Amount)
		}
		return msg, done, err
	}
}

func (q *quantitySetter) keyboard(o *order.Submit) [][]tgbotapi.InlineKeyboardButton {
	kbd := q.valueSetter.keyboard(o)
	// 可买/可卖数量
	var available float64

	switch o.Side {
	case order.Buy:
		// buy base with quote
		// 可用资金/价格
		available = q.positionQuote.Total / o.Price
	case order.Sell:
		available = q.positionBase.Total
	}
	// round to lotSize
	limit, ok := q.c.GetLimit(o.Pair)
	if !ok {
		log.Printf("cannot get limit of %s", o.Pair)
		return nil
	}
	roundToLotSize := func(q float64) float64 {
		step := limit.AmountStepIncrementSize
		return step * float64(int(q/step))
	}

	for _, split := range []struct {
		prop float64
		text string
	}{
		{0.25, "1/4"},
		{0.5, "1/2"},
		{0.75, "3/4"},
	} {
		quantity := roundToLotSize(split.prop * available)
		kbd = append(kbd,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					split.text,
					q.callback(fmt.Sprintf("%f", quantity), "set"),
				),
			))
	}
	quantity := roundToLotSize(available)
	kbd = append(kbd,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("100%% (%f)", quantity),
				q.callback(fmt.Sprintf("%f", quantity), "set"),
			)),
		resetRow(),
	)
	return kbd
}

func (q *quantitySetter) Next() (stage, bool) {
	if q.ok {
		return &finalizer{b: q.b, c: q.c}, false
	}
	return q, true
}
