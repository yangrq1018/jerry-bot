package coin

import (
	"fmt"

	"github.com/enescakir/emoji"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

type finalizer struct {
	b  *binance.Binance
	c  *Coin
	sr *order.SubmitResponse
}

func (f finalizer) String() string {
	return "finalize order"
}

func (f finalizer) prompt(o *order.Submit) string {
	return fmt.Sprintf(
		`
%s Check please
IOC:%t
币对:%s
价格:%s
数量:%f
预计金额:%.4f,
类型:%s
方向:%s
账户:%s
`,
		emoji.Coin,
		o.ImmediateOrCancel,
		o.Pair,
		f.c.FormatPrice(o.Price, o.Pair),
		o.Amount,
		o.Price*o.Amount,
		o.Type,
		o.Side,
		o.AssetType,
	)
}

func (f finalizer) keyboard(*order.Submit) [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("EXECUTE", "trade execute")),
		resetRow(),
	}
}

func (f *finalizer) Next() (stage, bool) {
	if f.sr != nil {
		// 如果服务器已经确认了订单, proceed
		f.sr = nil
		return f, false
	}
	return f, true
}

func (f finalizer) Callback() callback {
	return func(q *tgbotapi.CallbackQuery, o *order.Submit) (tgbotapi.Chattable, bool, error) {
		switch data := q.Data; {
		case data == "trade execute":
			// send order
			res, err := f.b.SubmitOrder(f.c.ctx, o)
			if err != nil {
				return nil, false, err
			}
			f.sr = res
			var text string
			if res.Status == order.Rejected {
				text = fmt.Sprintf("ID %s: 下单失败", res.OrderID)
			} else {
				text = fmt.Sprintf("ID %s: 已下单", res.OrderID)
			}
			msg := tgbotapi.NewEditMessageText(q.Message.Chat.ID, q.Message.MessageID, text)
			return &msg, true, nil
		}
		return nil, false, nil
	}
}
