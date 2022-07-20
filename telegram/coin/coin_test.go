package coin

import (
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

var B = NewBinance(gctConfig())
var C = NewCoin()

func TestGetBitcoinQuote(t *testing.T) {
	time, err := B.GetBestPrice(C.ctx, currency.NewPair(currency.BTC, currency.USDT))
	assert.Nil(t, err)
	t.Log(time)
}

func TestNewCoin(t *testing.T) {
	NewCoin()
}

func TestSumBalance(t *testing.T) {
	var sb strings.Builder
	assert.Nil(t, NewCoin().SumBalance(&sb, currency.USDT))
	log.Println(sb.String())
}

func TestGetPrecision(t *testing.T) {
	for _, code := range []string{
		"SHIB",
		"DOGE",
	} {
		prec, ok := C.GetQuotePrecision(currency.NewPair(currency.NewCode(code), currency.USDT))
		assert.True(t, ok)
		t.Logf("Precesion of %s: %d", code, prec)
	}
}

func TestCoin_GetLimit(t *testing.T) {
	for _, code := range []string{
		"SHIB",
		"DOGE",
	} {
		limit, ok := C.GetLimit(currency.NewPair(currency.NewCode(code), currency.USDT))
		assert.True(t, ok)
		t.Logf("Limit of %s: %f", code, limit.MinNotional)
	}
}

func Test_lotSize(t *testing.T) {
	// 最小数量提升
	for _, code := range []string{
		"DOGE",
		"BTC",
		"SHIB",
		"ETH",
	} {
		limit, ok := C.GetLimit(currency.NewPair(currency.NewCode(code), currency.USDT))
		assert.True(t, ok)
		t.Logf("step %s: step %f market step", code, limit.AmountStepIncrementSize)
	}
}
