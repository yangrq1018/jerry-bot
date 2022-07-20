package nintendo

import (
	"fmt"
	"image/png"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var ink = StatInk{
	Splatoon:  splatoon,
	Anonymous: true,
}

func TestPostBattle(t *testing.T) {
	results, _ := splatoon.Results()
	i := 0

	err := ink.PostBattleToStatink(results.Results[i], i == 0, false)
	assert.NoError(t, err)
}

func TestGetNSOAppVersion(t *testing.T) {
	version := getNSOAppVersion()
	assert.Equal(t, "1.14.0", version)
}

func TestPostBattleDryRun(t *testing.T) {
	results, _ := splatoon.Results()
	i := 0
	err := ink.PostBattleToStatink(results.Results[i], i == 0, false)
	assert.NoError(t, err)
}

func TestGetMasks(t *testing.T) {
	rr, err := splatoon.Results()
	assert.NoError(t, err)
	b := rr.Results[0] // test the most recent one, compare board with nintendo
	me := b.PlayerResult.AsStat(true, true, &b)
	payload := &battlePayload{}
	assert.NoError(t, ink.setScoreBoard(me, &b, payload))
	masks := getMasks(&b, payload.Players)
	fmt.Println(masks)
}

func TestSplatoonService_maskShareImageResult(t *testing.T) {
	bn := "4488"
	battle, err := splatoon.GetBattle(bn)
	assert.NoError(t, err)
	im, err := splatoon.shareImageResult(bn)
	assert.NoError(t, err)
	payload := &battlePayload{}
	assert.NoError(t, ink.setScoreBoard(battle.PlayerResult.AsStat(true, true, battle), battle, payload))
	im = blackout(im, getMasks(battle, payload.Players), true)
	assert.NotNil(t, im)
	file, err := os.OpenFile("../../test_data/blackout_except_me.png", os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	err = png.Encode(file, im)
	assert.NoError(t, err)
}

func TestSplatoonService_SetScoreBoard(t *testing.T) {
	rr, err := splatoon.Results()
	assert.NoError(t, err)
	b := rr.Results[0] // test the most recent one, compare board with nintendo
	me := b.PlayerResult.AsStat(true, true, &b)
	assert.NoError(t, ink.setScoreBoard(me, &b, &battlePayload{}))
}

func TestBlackout(t *testing.T) {
	im, err := splatoon.shareImageResult("4271")
	assert.NoError(t, err)
	im = blackout(im, [8]uint8{0, 1, 0, 0, 1, 1, 0, 1}, true) // mark everyone except me
	file, err := os.OpenFile("test_data\\blackout.png", os.O_CREATE|os.O_WRONLY, 0644)
	assert.NoError(t, err)
	err = png.Encode(file, im)
	assert.NoError(t, err)
}

func TestDigestUsername(t *testing.T) {
	digest := digestUsername("ryang")
	fmt.Println(digest, len(digest))
}
