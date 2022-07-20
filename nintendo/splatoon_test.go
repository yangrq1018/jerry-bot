package nintendo

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func getSplatoon() *SplatoonService {
	cookie := "bc6456c0b5e23e226736f42c8ae529a9b75ed907" // auto generated
	return NewClient(cookie).Splatoon()
}

var splatoon = getSplatoon()

func TestSplatoonService_Player(t *testing.T) {
	p, err := splatoon.Player()
	assert.NoError(t, err)
	assert.NotNil(t, p)
	fmt.Printf("%s\n", p.RankUdemaeString())
}

func TestSplatoonService_RecentBattles(t *testing.T) {
	b, err := splatoon.Recent4Battles()
	assert.NoError(t, err)
	fmt.Printf("%s\n", b)
}

func TestSplatoonService_shareImageResult(t *testing.T) {
	image, err := splatoon.shareImageResult("4580")
	assert.NoError(t, err)
	assert.NotNil(t, image)
}

func TestSplatoonService_shareImageResultAfterShareProfile(t *testing.T) {
	// make sure cookie is not set twice
	_, err := splatoon.shareImageProfile(3, "yellow")
	assert.NoError(t, err)
	fmt.Println("cookie after request #1", appHeadShare.Get("cookie"))
	image, err := splatoon.shareImageResult("4732")
	fmt.Println("cookie after request #2", appHeadShare.Get("cookie"))
	assert.NoError(t, err)
	assert.NotNil(t, image)
}

func TestSplatoonService_shareImageProfile(t *testing.T) {
	image, err := splatoon.shareImageProfile(3, "yellow")
	assert.NoError(t, err)
	assert.NotNil(t, image)
}

func TestSplatoonService_GetBattle(t *testing.T) {
	// the battle should not be too old, will be clear by splat net, return 404 not found
	b, err := splatoon.GetBattle("4137")
	assert.Error(t, err)

	b, err = splatoon.GetBattle("4207")
	assert.NoError(t, err)
	assert.Greater(t, len(b.MyTeamMembers), 0)
	assert.Greater(t, len(b.OtherTeamMembers), 0)
}

func TestSplatoonService_OnlineShop(t *testing.T) {
	shop, err := splatoon.OnlineShop()
	assert.NoError(t, err)
	assert.NotNil(t, shop)
	fmt.Printf("%+v\n", shop)
}

func TestSplatoonService_RecentResults(t *testing.T) {
	rr, err := splatoon.Results()
	assert.NoError(t, err)
	assert.NotNil(t, rr)
	battles := rr.Results
	for i := range battles {
		if battles[i].Type == "regular" {
			fmt.Println(i)
			break
		}
	}
}

func TestSplatoonService_Stages(t *testing.T) {
	stages, err := splatoon.Stages()
	assert.NoError(t, err)
	assert.NotNil(t, stages)
}

func TestSplatoonService_Schedules(t *testing.T) {
	schedules, err := splatoon.CurrentSchedule()
	assert.NoError(t, err)
	assert.NotNil(t, schedules)

	fmt.Println(schedules.Gachi[0])
	fmt.Println(schedules.Regular[0])
	fmt.Println(schedules.League[0])
}

func TestSplatoonService_Records(t *testing.T) {
	records, err := splatoon.Records()
	assert.NoError(t, err)
	assert.NotNil(t, records)
	weapons := records.SortWeapons(CompareWeaponMostUsed)
	for _, w := range weapons {
		fmt.Println(w.Usage())
	}
}
