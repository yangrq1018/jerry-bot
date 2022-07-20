package nintendo

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// define splatoon service

const (
	endpointRecords          = "/api/records"
	endpointTimeline         = "/api/timeline"
	endpointStages           = "/api/data/stages"
	endpointResults          = "/api/results"
	endpointSchedules        = "/api/schedules"
	endpointIndividualResult = "/api/results/"
)

// for share result or share profile
// do not share header object! make sure you copy it
var appHeadShare = makeHeader(map[string]string{
	"Host":              "app.splatoon2.nintendo.net",
	"x-unique-id":       appUniqueID,
	"x-requested-with":  "XMLHttpRequest",
	"x-timezone-offset": "-480", // intercepted from Charles, no effect seems
	"User-Agent":        "Mozilla/5.0 (iPhone; CPU iPhone OS 14_8 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
	"Accept":            "*/*",
	"Referer":           "https://app.splatoon2.nintendo.net/home",
	"Accept-Encoding":   "gzip,deflate,br",
	"Accept-Language":   userLang,
})

var defaultSplatoon = defaultNintendo.Splatoon()

type SplatoonService struct {
	webService
}

func (s *SplatoonService) Scheme() string {
	return "https"
}

func (s *SplatoonService) Domain() string {
	return "app.splatoon2.nintendo.net"
}

func (s *SplatoonService) Player() (*Player, error) {
	r, err := s.records()
	if err != nil {
		return nil, err
	}
	return &r.Records.Player, err
}

func (s *SplatoonService) Records() (*Records, error) {
	r, err := s.records()
	if err != nil {
		return nil, err
	}
	return &r.Records, nil
}

func (s *SplatoonService) OnlineShop() (*OnlineShop, error) {
	t, err := s.timeline()
	if err != nil {
		return nil, err
	}
	return &t.OnlineShop, nil
}

// Recent4Battles 返回最近4次战斗数据
func (s *SplatoonService) Recent4Battles() (string, error) {
	t, err := s.timeline()
	if err != nil {
		return "", err
	}
	battles := t.Stats.Recents
	var sb strings.Builder
	for _, battle := range battles {
		sb.WriteString(battle.String() + "\n")
	}
	return sb.String(), nil
}

// CurrentSchedule 返回当前三个模式正在进行的规则、地图、开始和结束时间
func (s *SplatoonService) CurrentSchedule() (*Schedules, error) {
	t, err := s.timeline()
	if err != nil {
		return nil, err
	}
	return &t.Schedule.Schedules, nil
}

func (s *SplatoonService) Results() (*RecentResults, error) {
	var r RecentResults
	err := s.Api("GET", endpointResults, &r)
	return &r, err
}

// Recent50Battles 返回最近50次战斗数据
func (s *SplatoonService) Recent50Battles() ([]Battle, error) {
	r, err := s.Results()
	if err != nil {
		return nil, err
	}
	return r.Results, nil
}

// GachiSchedule 返回之后24小时的count个真格模式session， 最多12
func (s *SplatoonService) GachiSchedule(count int) ([]Session, error) {
	sch, err := s.schedules()
	if err != nil {
		return nil, err
	}
	gachi := sch.Gachi
	if count >= 0 {
		if count > len(gachi) {
			count = len(gachi)
		}
		gachi = gachi[:count]
	}
	return gachi, nil
}

func (s *SplatoonService) Stages() (map[string]Stage, error) {
	var res struct {
		Stages []Stage
	}
	err := s.Api("GET", endpointStages, &res)
	if err != nil {
		return nil, err
	}

	var data = make(map[string]Stage)
	for _, stage := range res.Stages {
		data[stage.ID] = stage
	}
	return data, nil
}

// GetBattle 带有队友信息，我方没有自己
func (s *SplatoonService) GetBattle(battleNumber string) (*Battle, error) {
	var result Battle
	err := s.Api("GET", endpointIndividualResult+battleNumber, &result)
	return &result, err
}

func (s *SplatoonService) shareImageResult(battleNumber string) (image.Image, error) {
	shareResult, err := postHTTPJson(
		fmt.Sprintf(endpointSplatNetShareResult, battleNumber),
		nil,
		appHeadShare,
		[]*http.Cookie{s.client.IksmCookie()})
	if err != nil {
		return nil, fmt.Errorf("cannot get share image url: %w", err)
	}
	// the amazon aws url works with faster JP nodes
	return imageFromURL(shareResult.String("url"))
}

func imageFromURL(url string) (image.Image, error) {
	res, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	// png image stream
	m, err := png.Decode(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, err
	}
	return m, err
}

type multipartForm map[string]string

// this endpoint takes multipart/form post
func (s *SplatoonService) shareImageProfile(stage int, color string) (image.Image, error) {
	var favStage int
	if stage >= 100 { // fav_stage can't be Shifty Stations
		// randomly pick a non-shifty stage
		favStage = rand.Intn(23)
		// ensure is valid stage
		_, ok := stageEnum[favStage]
		if !ok {
			return nil, fmt.Errorf("sorry, %d is not valid stage", favStage)
		}
	} else {
		favStage = stage
	}
	settings := multipartForm{
		"stage": strconv.Itoa(favStage),
		"color": color,
	}

	// "XMLHttpRequest"?
	shareResult, err := postHTTPMultipartForm(endpointSplatNetShareProfile, settings, appHeadShare, []*http.Cookie{s.client.IksmCookie()})
	if err != nil {
		return nil, err
	}
	return imageFromURL(shareResult.String("url"))
}

// endpoints, internal

func (s *SplatoonService) records() (*recordEndpoint, error) {
	var r recordEndpoint
	err := s.Api("GET", endpointRecords, &r)
	return &r, err
}

func (s *SplatoonService) timeline() (*timelineEndpoint, error) {
	var t timelineEndpoint
	err := s.Api("GET", endpointTimeline, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *SplatoonService) schedules() (*scheduleEndpoint, error) {
	var sch scheduleEndpoint
	err := s.Api("GET", endpointSchedules, &sch)
	return &sch, err
}

// 0 for equal, -1 for less, 1 for greater
type comparator func(a, b Stat) int
type lessStats func(a, b Stat) bool

func chainItemGetters(igs ...comparator) lessStats {
	return func(a, b Stat) bool {
		for i := range igs {
			diff := igs[i](a, b)
			if diff < 0 {
				return true
			} else if diff > 0 {
				return false
			}
		}
		return true
	}
}

// disguise opponent usernames
func digestUsername(name string) string {
	hash := sha1.Sum([]byte(name)) // 160 bit, 20 byte
	// [27]char string
	fullDigest := base64.RawURLEncoding.EncodeToString(hash[:])
	return fullDigest[:8] // should be 27 ascii char
}

var (
	// 0
	getSortScore comparator = func(a, b Stat) int {
		return a.SortScore - b.SortScore
	}
	// 8
	getPoint comparator = func(a, b Stat) int {
		return a.Point - b.Point
	}
	getKoA comparator = func(a, b Stat) int {
		return a.KillOrAssist - b.KillOrAssist
	}
	getSpecial comparator = func(a, b Stat) int {
		return a.Special - b.Special
	}
	getDeath comparator = func(a, b Stat) int {
		return a.Death - b.Death
	}
	getKill comparator = func(a, b Stat) int {
		return a.Kill - b.Kill
	}
	getName comparator = func(a, b Stat) int {
		return strings.Compare(a.Name, b.Name)
	}
)

func (r Records) SortWeapons(less func(x, y WeaponStat) bool) []WeaponStat {
	var weapons []WeaponStat
	for _, v := range r.WeaponStats {
		weapons = append(weapons, v)
	}
	sort.Slice(weapons, func(i, j int) bool {
		// reverse sort
		return !less(weapons[i], weapons[j])
	})
	return weapons
}

func (u Udemae) autoRank() string {
	switch {
	case u.IsX:
		// X power?
		return "X"
	case u.Name == "S+":
		return "S+" + strconv.Itoa(u.SPlusNumber)
	default:
		return u.Name
	}
}

func (u Udemae) String() string {
	return u.autoRank()
}

func CompareWeaponMostUsed(x, y WeaponStat) bool {
	return x.TotalPaintPoint < y.TotalPaintPoint
}

// RankUdemaeString 真格模式段位
func (p Player) RankUdemaeString() string {
	return fmt.Sprintf(
		`模式:蛤蜊
段位:%s

模式:抢鱼
段位:%s

模式:推塔
段位:%s

模式:区域
段位:%s
`,
		p.UdemaeClam.autoRank(),
		p.UdemaeRainmaker.autoRank(),
		p.UdemaeTower.autoRank(),
		p.UdemaeZones.autoRank(),
	)
}

func (s WeaponStat) Usage() string {
	return fmt.Sprintf(`武器: %s
(副%s 大%s)
涂地面积: %d
上次使用: %s
胜率: %.1f%%
当前Power: %.0f
最大Power: %.0f
`,
		s.Weapon.Name,
		s.Weapon.Sub.Name, s.Weapon.Special.Name,
		s.TotalPaintPoint,
		time.Unix(s.LastUseTime, 0).Format("2006-01-02"),
		float64(s.WinCount)/float64(s.WinCount+s.LoseCount)*100,
		s.WinMeter,
		s.MaxWinMeter,
	)
}

func (i ImageURL) URL() *url.URL {
	u := newURL(defaultSplatoon)
	u.Path = string(i)
	return u
}

func (p Player) UdemaeOf(mode string, rule string) *Udemae {
	if (mode == "gachi" || mode == "league") && p.RankingModeStat != nil {
		return p.RankingModeStat.UdemaeOf(rule)
	}
	return nil
}

func (tr TeamResult) Win() bool {
	return tr.Key == "victory"
}

func (b Battle) Win() bool {
	return b.MyTeamResult.Win()
}

func (s RankingModeStat) UdemaeOf(rule string) *Udemae {
	var u Udemae
	switch rule {
	case "clam":
		u = s.UdemaeClam
	case "rainmaker":
		u = s.UdemaeRainmaker
	case "key=tower_control":
		u = s.UdemaeTower
	case "splat_zones":
		u = s.UdemaeZones
	default:
		log.Printf("unknown rule key: %s", rule)
	}
	return &u
}
