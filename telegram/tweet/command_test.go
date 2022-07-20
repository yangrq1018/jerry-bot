package tweet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yangrq1018/jerry-bot/telegram"
)

func TestCommand(t *testing.T) {
	cmd := Command()
	bw, err := telegram.NewMessageBot(
		telegram.JerryToken(),
	)
	assert.NoError(t, err)
	assert.NoError(t, bw.RegisterCommand(cmd))
	assert.NoError(t, bw.Init())
	bw.Listen(80)
}
