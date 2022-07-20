package coin

import (
	"fmt"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/yangrq1018/jerry-bot/telegram"
	"github.com/yangrq1018/jerry-bot/util"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

func (c *Coin) reset() {
	c.stage = &directioner{
		b: c.binance,
		c: c,
	}
	c.callback = nil
	c.order = new(order.Submit)
	c.order.Type = order.Limit // 限价单
	c.order.AssetType = asset.Spot
}

func (c *Coin) ID() tgbotapi.BotCommand {
	return tgbotapi.BotCommand{
		Command:     "coin",
		Description: "虚拟货币交易",
	}
}

func (c *Coin) Serve(bot *telegram.Bot) error {
	err := c.InitBot(bot)
	if err != nil {
		return err
	}
	bot.Match(c).Subscribe(c.handle)
	bot.CallBackQueryEvent.Subscribe(c.callbackQuery)
	return nil
}

func (c *Coin) handle(b *telegram.Bot, u tgbotapi.Update) error {
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, "What would you like to do?")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			{Text: "balance", CallbackData: util.StringPtr("balance")},
			{Text: "trade", CallbackData: util.StringPtr("trade")},
		},
	)
	_, err := b.Bot().Send(msg)
	return err
}

func (c *Coin) Init() {}

func (c *Coin) Authorize() telegram.Authorizer {
	return telegram.SimpleAuth{
		DeveloperOnly: true,
	}
}

func (c *Coin) callbackQuery(b *telegram.Bot, query tgbotapi.CallbackQuery) error {
	switch data := query.Data; {
	case strings.HasPrefix(data, "trade"):
		// user pressed trade button again
		if data == "trade" || data == "trade RESET" {
			c.reset()
		}
		for {
			if c.stage == nil {
				c.reset()
			}
			if c.callback != nil {
				msg, done, err := c.callback(&query, c.order)
				if msg != nil {
					_, err = b.Bot().Send(msg)
					if err != nil {
						return err
					}
				}
				if done || err != nil {
					c.reset()
					return err
				}
			}
			msg := tgbotapi.NewEditMessageTextAndMarkup(
				query.Message.Chat.ID,
				query.Message.MessageID,
				c.stage.prompt(c.order),
				tgbotapi.NewInlineKeyboardMarkup(c.stage.keyboard(c.order)...),
			)
			c.callback = c.stage.Callback()
			var br bool
			c.stage, br = c.stage.Next()
			_, err := b.Bot().Send(msg)
			switch x := err.(type) {
			case tgbotapi.Error:
				// hacky way to determine the message has not changed
				if x.Code == 400 {
					err = nil
				}
			}
			if err != nil {
				return err
			}
			if br {
				break
			}
		}
		return nil
	case data == "balance":
		var sb strings.Builder
		err := c.SumBalance(&sb, currency.USDT) // quoted in USDT, free pick other currencies of course
		if err != nil {
			return err
		}
		// this will erase the markup!
		b.Bot().Send(tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, sb.String()))
	case strings.HasPrefix(data, "bestPrice"):
		var args string
		fmt.Sscanf(data, "bestPrice %s", &args)
		pair, err := currency.NewPairFromString(args)
		if err != nil {
			return err
		}
		bestPrice, err := c.binance.GetBestPrice(c.ctx, pair)
		if err != nil {
			return err
		}
		b.Bot().Send(tgbotapi.NewMessage(query.Message.Chat.ID,
			fmt.Sprintf(`%s now trades at:
Bid %f, Qty %f
Ask %f, Qty %f`, pair, bestPrice.BidPrice, bestPrice.BidQty, bestPrice.AskPrice, bestPrice.AskQty)))
	}
	return nil
}
