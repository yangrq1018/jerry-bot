package coin

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

type callback func(q *tgbotapi.CallbackQuery, o *order.Submit) (tgbotapi.Chattable, bool, error)

// ring-like finite automata
type stage interface {
	String() string
	prompt(o *order.Submit) string
	keyboard(o *order.Submit) [][]tgbotapi.InlineKeyboardButton
	// Next
	// false to proceed, true to keep in loop
	Next() (stage, bool)
	Callback() callback
}
