package coin

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/enescakir/emoji"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/yangrq1018/jerry-bot/telegram"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

type Coin struct {
	binance      *binance.Binance
	websocket    *stream.Websocket
	order        *order.Submit // pending order
	stage        stage
	callback     callback
	exchangeInfo binance.ExchangeInfo
	limits       []order.MinMaxLevel // store exchange info on pair, such as step size, min notional
	balance      map[asset.Item][]account.Balance
	ctx          context.Context
}

//go:embed config/config.json
var gctConfigStream []byte
var gctLoggerSetupOnce sync.Once

func gctConfig() (cfg config.Config) {
	err := json.NewDecoder(bytes.NewReader(gctConfigStream)).Decode(&cfg)
	if err != nil {
		log.Fatalf("invalid config from go:embed config/config.json: %v", err)
	}
	return
}

func NewBinance(cfg config.Config) *binance.Binance {
	gctLoggerSetupOnce.Do(func() {
		// set up loggers
		gctlog.GlobalLogConfig = &cfg.Logging
		_ = gctlog.SetupGlobalLogger()
		_ = gctlog.SetupSubLoggers(cfg.Logging.SubLoggers)
		gctlog.Infoln(gctlog.Global, "Logger initialised.")
	})

	var b binance.Binance
	binanceConfig, err := cfg.GetExchangeConfig("Binance")
	if err != nil {
		log.Fatal("Binance GetExchangeConfig error: ", err)
	}
	b.SetDefaults()
	err = b.Setup(binanceConfig)
	if err != nil {
		log.Fatal("Binance setup error", err)
	}
	return &b
}

func NewCoin() *Coin {
	var (
		c              Coin
		err            error
		availablePairs currency.Pairs
	)
	c.ctx = context.Background()
	c.binance = NewBinance(gctConfig())
	c.exchangeInfo, err = c.binance.GetExchangeInfo(c.ctx)
	if err != nil {
		panic(err)
	}

	// enable all pairs
	enabledPairs, err := c.binance.CurrencyPairs.GetPairs(asset.Spot, true)
	if err != nil {
		log.Printf("cann get enabled pairs: %v", err)
	}

	for _, symbol := range c.exchangeInfo.Symbols {
		pair := currency.NewPair(currency.NewCode(symbol.BaseAsset), currency.NewCode(symbol.QuoteAsset))
		availablePairs = append(availablePairs, pair)
	}
	c.binance.CurrencyPairs.StorePairs(asset.Spot, availablePairs, false)
	for _, pair := range availablePairs {
		if !enabledPairs.Contains(pair, true) {
			// safe to ignore error
			_ = c.binance.CurrencyPairs.EnablePair(asset.Spot, pair)
		}
	}

	limits, err := c.binance.FetchSpotExchangeLimits(c.ctx)
	if err != nil {
		panic(err)
	}

	c.limits = limits
	return &c
}

func (c *Coin) updateBalance() error {
	c.balance = make(map[asset.Item][]account.Balance)
	for _, item := range []asset.Item{asset.Spot, asset.Margin} {
		holdings, err := c.binance.UpdateAccountInfo(c.ctx, item)
		if err != nil {
			return err
		}
		for _, subAccount := range holdings.Accounts {
			for _, bal := range subAccount.Currencies {
				if bal.Total > 0 {
					c.balance[item] = append(c.balance[item], bal)
				}
			}
		}
	}
	return nil
}

// SumBalance
// quoteCurrency 报价币种
func (c *Coin) SumBalance(sb *strings.Builder, quoteCurrency currency.Code) error {
	err := c.updateBalance()
	if err != nil {
		return err
	}

	var totalNetAsset float64
	for _, item := range []asset.Item{asset.Spot, asset.Margin} {
		sb.WriteString(fmt.Sprintf(`账户 %s
`, item))
		for _, bal := range c.balance[item] {
			switch bal.CurrencyName {
			case quoteCurrency:
				totalNetAsset += bal.Total
				sb.WriteString(fmt.Sprintf(
					`[%s]余额
	total %f
`,
					bal.CurrencyName, bal.Total,
				))
			default:
				pair := currency.NewPair(bal.CurrencyName, quoteCurrency)
				bp, err := c.binance.GetBestPrice(c.ctx, pair)
				if err != nil {
					return fmt.Errorf("failed to quote best price for pair %s: %v", pair, err)
				}

				avgPrice := (bp.AskPrice + bp.BidPrice) / 2
				netAsset := avgPrice * bal.Total
				totalNetAsset += netAsset

				// 显示一美元以上的持仓
				if netAsset < 1.0 {
					continue
				}

				// 获取最近24hr的价格变化
				rolling24hrPriceChange, err := c.binance.GetPriceChangeStats(c.ctx, pair)
				if err != nil {
					return err
				}
				var direction string
				var pctChange float64
				pctChange = rolling24hrPriceChange.PriceChangePercent
				switch {
				case pctChange > 0:
					direction = emoji.ChartIncreasing.String()
				default:
					direction = emoji.ChartDecreasing.String()
				}

				var avgPriceString string
				// 默认保留四位， 除非价格小于1e-4
				if avgPrice < 1e-4 {
					avgPriceString = fmt.Sprintf("%e", avgPrice)
				} else {
					avgPriceString = fmt.Sprintf("%.4f", avgPrice)
				}
				sb.WriteString(fmt.Sprintf(
					`[%s]
	total %f
	value %.2f%s
	trades @ %s %s/%s
	last 24hr %s%.2f%%
`,
					bal.CurrencyName, bal.Total,
					netAsset, quoteCurrency.String(),
					avgPriceString, quoteCurrency.String(), bal.CurrencyName.String(), direction, math.Abs(pctChange)))
			}
		}
	}
	// 计算总持仓净值，以报价币种估值
	sb.WriteString(fmt.Sprintf(`Total Net Asset (%s): %.8f
`, quoteCurrency, totalNetAsset))
	return nil
}

func (c *Coin) FormatPrice(p float64, pair currency.Pair) string {
	precision, ok := c.GetQuotePrecision(pair)
	if !ok {
		precision = 8 // 默认保留8位
	}
	return stringifyPrice(p, precision)
}

func (c *Coin) genSubscriptions() []stream.ChannelSubscription {
	return []stream.ChannelSubscription{
		{
			Channel: "executionReport",
		},
	}
}

func (c *Coin) wsUnsubscribe() error {
	return c.websocket.UnsubscribeChannels(c.genSubscriptions())
}

func (c *Coin) wsSubscribe() error {
	return c.websocket.SubscribeToChannels(c.genSubscriptions())
}

// InitBot satisfies the InitHandler of telegram.Bot
// starts the websocket handling routinue
func (c *Coin) InitBot(b *telegram.Bot) error {
	err := c.binance.WsConnect()
	if err != nil {
		panic(err)
	}
	c.websocket, _ = c.binance.GetWebsocket()
	err = c.wsSubscribe()
	if err != nil {
		panic(err)
	}
	go c.HandleData(b)
	go c.autoReconnect()
	return nil
}

const (
	reconnectMaxAttempt = 10
	reconnectDelay      = 10 * time.Second
)

func (c *Coin) autoReconnect() {
	for err := range c.websocket.ReadMessageErrors {
		log.Printf("receive a read messaeg error: %v, reconnect and resubscribe", err)
		var attempt int
		for {
			attempt++
			err = c.binance.WsConnect()
			if err != nil {
				log.Printf("%v, retry in %s", err, reconnectDelay)
				if attempt > reconnectMaxAttempt {
					log.Printf("reconnect failed more than %d times", reconnectMaxAttempt)
					break
				}
			} else {
				log.Printf("reconnect ok")
				break
			}
			time.Sleep(reconnectDelay)
		}
		err = c.wsResubscribe()
		if err != nil {
			log.Printf("cannot resubscribe: %v", err)
		} else {
			log.Printf("resubscribe ok")
		}
	}
}

func (c *Coin) wsResubscribe() error {
	err := c.wsUnsubscribe()
	if err != nil {
		return err
	}
	err = c.wsSubscribe()
	return err
}

func (c *Coin) HandleData(b *telegram.Bot) {
	for {
		data := <-c.websocket.DataHandler
		switch d := data.(type) {
		case error:
			log.Printf("encounter an error in websocket stream: %v", d)
		case *order.Detail:
			var text string
			switch d.Status {
			case order.Filled, order.PartiallyFilled:
				log.Printf("order %s %s", d.OrderID, d.Status)
				var feePercentage float64
				switch d.Side {
				case order.Sell:
					// 卖单以报价币种计手续费
					// Cost一直是报价币种
					feePercentage = d.Fee / d.Cost
				case order.Buy:
					// 买单以基准币种计手续费
					feePercentage = d.Fee / d.Amount
				}
				feePercentage *= 10000
				text = fmt.Sprintf(`【订单%s:%s】
币对:%s
类型:%s
方向:%s
状态:%s
原始价格:%s
原始数量:%f
订单金额:%.4f
已成:%f
未成:%f
IOC:%t
FOK:%t
杠杆:%f
创建时间:%s
账户:%s
累计已成交金额:%f %s
手续费:%f %s
手续费率:%.2f/W
`,
					d.OrderID, d.Status,
					d.Pair,
					d.Type,
					d.Side,
					d.Status,
					c.FormatPrice(d.Price, d.Pair),
					d.Amount,
					d.Amount*d.Price,
					d.ExecutedAmount,
					d.RemainingAmount,
					d.ImmediateOrCancel,
					d.FillOrKill,
					d.Leverage,
					d.Date.Format("2006-01-02 15:04:05"),
					d.AssetType,
					d.Cost, d.CostAsset,
					d.Fee, d.FeeAsset,
					feePercentage,
				)
			case order.New, order.Cancelled:
				text = fmt.Sprintf(`【订单%s:%s】`, d.OrderID, d.Status)
			}
			if text != "" {
				msg := tgbotapi.NewMessage(telegram.DeveloperChatID, text)
				_, err := b.Bot().Send(msg)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func (c *Coin) GetQuotePrecision(pair currency.Pair) (int, bool) {
	for _, symbol := range c.exchangeInfo.Symbols {
		if symbol.BaseAsset == pair.Base.String() && symbol.QuoteAsset == pair.Quote.String() {
			return symbol.QuotePrecision, true
		}
	}
	return 0, false
}

func (c *Coin) GetLimit(pair currency.Pair) (order.MinMaxLevel, bool) {
	for x := range c.limits {
		if c.limits[x].Pair.Equal(pair) {
			return c.limits[x], true
		}
	}
	return order.MinMaxLevel{}, false
}
