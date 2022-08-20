package nintendo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/yangrq1018/jerry-bot/util"
)

const (
	NSOAppVersionDefault = "2.0.0"

	// stat.ink api key, used by upload battle
	apiKey      = "jv0pmNerd8o9xyqPJ56rBeDYPD6p7dx7KyLMualN-9E"
	appUniqueID = "15635090535659533636"
	userLang    = "zh-cn" // have no effects, can be empty actually
	agent       = "splatnet2statink"
	// agentVersion is splatnet2statink version. Agent version in payload to battle upload
	agentVersion = "1.7.3"
)

// Nintendo Switch Online app version
// perhaps check every day?
func getNSOAppVersion() string {
	res, err := http.Get("https://itunes.apple.com/lookup?id=1234806557&country=JP")
	if err != nil {
		return NSOAppVersionDefault
	}
	defer res.Body.Close()
	var result struct {
		Results []struct {
			Version string `json:"version"`
		} `json:"results"`
	}
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return NSOAppVersionDefault
	}
	if len(result.Results) < 1 {
		return NSOAppVersionDefault
	}
	return result.Results[0].Version
}

func GetStatinkBattles() ([]int, error) {
	var result []int
	err := getHTTPUnmarshal("https://stat.ink/api/v2/user-battle?only=splatnet_number&count=100", authBearToken(apiKey), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// https://github.com/fetus-hina/stat.ink/blob/master/doc/api-2/post-battle.md
type battlePayload struct {
	UUID                       string            `json:"uuid"`
	SplatnetNumber             int               `json:"splatnet_number"`
	Lobby                      string            `json:"lobby"`
	Mode                       string            `json:"mode"`
	Rule                       string            `json:"rule"`
	Stage                      string            `json:"stage"`
	Weapon                     string            `json:"weapon"`
	Result                     string            `json:"result"`
	KnockOut                   string            `json:"knock_out"`
	RankInTeam                 int               `json:"rank_in_team"`
	Kill                       int               `json:"kill"`
	Death                      int               `json:"death"`
	MaxKillCombo               int               `json:"max_kill_combo"`
	MaxKillStreak              int               `json:"max_kill_streak"`
	KillOrAssist               int               `json:"kill_or_assist"`
	Special                    int               `json:"special"`
	Freshness                  float64           `json:"freshness"`
	Level                      int               `json:"level"`
	LevelAfter                 int               `json:"level_after"`
	StarRank                   int               `json:"star_rank"`
	Rank                       string            `json:"rank"`
	RankExp                    int               `json:"rank_exp"`
	RankAfter                  string            `json:"rank_after"`
	RankExpAfter               int               `json:"rank_exp_after"`
	XPower                     float64           `json:"x_power"`
	XPowerAfter                float64           `json:"x_power_after"`
	EstimateXPower             int               `json:"estimate_x_power"`
	MyPoint                    int               `json:"my_point"`
	EstimateGachiPower         int               `json:"estimate_gachi_power"`
	LeaguePoint                float64           `json:"league_point"`
	MyTeamEstimateLeaguePoint  int               `json:"my_team_estimate_league_point"`
	HisTeamEstimateLeaguePoint int               `json:"his_team_estimate_league_point"`
	MyTeamPoint                int               `json:"my_team_point"`
	HisTeamPoint               int               `json:"his_team_point"`
	MyTeamPercent              float64           `json:"my_team_percent"`
	HisTeamPercent             float64           `json:"his_team_percent"`
	MyTeamCount                int               `json:"my_team_count"`
	HisTeamCount               int               `json:"his_team_count"`
	MyTeamID                   string            `json:"my_team_id"`
	HisTeamID                  string            `json:"his_team_id"`
	Species                    string            `json:"species"`
	Gender                     string            `json:"gender"`
	FestTitle                  string            `json:"fest_title"`
	FestExp                    int               `json:"fest_exp"`
	FestTitleAfter             string            `json:"fest_title_after"`
	FestExpAfter               int               `json:"fest_exp_after"`
	FestPower                  float64           `json:"fest_power"`
	MyTeamEstimateFestPower    int               `json:"my_team_estimate_fest_power"`
	HisTeamEstimateFestPower   int               `json:"his_team_estimate_fest_power"`
	MyTeamFestTheme            string            `json:"my_team_fest_theme"`
	HisTeamFestTheme           string            `json:"his_team_fest_theme"`
	MyTeamNickname             string            `json:"my_team_nickname"`
	HisTeamNickname            string            `json:"his_team_nickname"`
	Clout                      int               `json:"clout"`
	TotalClout                 int               `json:"total_clout"`
	TotalCloutAfter            int               `json:"total_clout_after"`
	SynergyBonus               float64           `json:"synergy_bonus,omitempty"`
	MyTeamWinStreak            int               `json:"my_team_win_streak"`
	HisTeamWinStreak           int               `json:"his_team_win_streak"`
	SpecialBattle              string            `json:"special_battle"`
	Gears                      Gears             `json:"gears"`
	Players                    []Stat            `json:"players"` // scoreboard, set by setScoreBoard
	DeathReasons               map[string]string `json:"-"`
	Events                     interface{}       `json:"-"`
	SplatnetJson               interface{}       `json:"splatnet_json"`
	Automated                  string            `json:"automated"`
	LinkURL                    string            `json:"link_url"`
	Note                       string            `json:"note"`
	PrivateNote                string            `json:"private_note"`
	Agent                      string            `json:"agent"`
	AgentVersion               string            `json:"agent_version"`
	AgentCustom                string            `json:"agent_custom"`
	AgentVariables             map[string]string `json:"agent_variables"`
	ImageJudge                 []byte            `json:"image_judge"`  // 画像バイナリ(PNG/JPEG)
	ImageResult                []byte            `json:"image_result"` // 画像バイナリ(PNG/JPEG)
	ImageGear                  []byte            `json:"image_gear"`   // 画像バイナリ(PNG/JPEG)
	StartAt                    int64             `json:"start_at"`
	EndAt                      int64             `json:"end_at"`
}

// statinkGear See also: https://github.com/fetus-hina/stat.ink/blob/master/doc/api-2/post-battle.md#gears-structure
type statinkGear struct {
	Gear               string   `json:"gear"`
	PrimaryAbility     string   `json:"primary_ability"`
	SecondaryAbilities []string `json:"secondary_abilities"` // "" for ?, not yet obtained
}

type Gears struct {
	Headgear statinkGear `json:"headgear"`
	Clothing statinkGear `json:"clothing"`
	Shoes    statinkGear `json:"shoes"`
}

func battleUUID(battle *Battle) string {
	// microsoft encoding
	namespace, err := uuid.Parse("{73cf052a-fd0b-11e7-a5ee-001b21a098c2}")
	if err != nil {
		panic(err)
	}
	// identify this battle in stat.ink
	name := battle.BattleNumber + "@" + battle.PlayerResult.Player.PrincipalID
	return uuid.NewSHA1(namespace, []byte(name)).String()
}

type StatInk struct {
	Mode      string
	Splatoon  *SplatoonService
	Anonymous bool
	Debug     bool
}

// PostBattleToStatink Uploads battle #i that is battle
// "i" is the offset of this battle in timeline, 0 means the most recent
func (s *StatInk) PostBattleToStatink(battle Battle, sendGears bool, dryRun bool) error {
	/* PAYLOAD */
	payload := battlePayload{
		Agent:        agent,
		AgentVersion: agentVersion,
		Automated:    "yes",
		AgentVariables: map[string]string{
			"upload_mode": s.Mode,
			"anonymous":   strconv.FormatBool(s.Anonymous),
			"send_gears":  strconv.FormatBool(sendGears),
		},
	}
	payload.UUID = battleUUID(&battle)

	/* LOBBY & MODE */
	var lobby, mode string
	switch battle.GameMode.Key {
	case "regular": // turf war
		lobby = "standard"
		mode = "regular"
	case "gachi": // ranked solo
		lobby = "standard"
		mode = "gachi"
	case "league_pair":
		lobby = "squad_2"
		mode = "gachi"
	case "league_team":
		lobby = "squad_4"
		mode = "gachi"
	case "private":
		lobby = "private"
		mode = "private"
	case "fes_solo":
		lobby = "standard"
		mode = "fest"
	case "fes_team":
		lobby = "squad_4"
		mode = "fest"
	}
	payload.Lobby = lobby
	payload.Mode = mode

	/* RULE */
	var rule string
	switch battle.Rule.Key {
	case RuleTurfWar:
		rule = "nawabari"
	case RuleSplatZones:
		rule = "area"
	case RuleTowerControl:
		rule = "yagura"
	case RuleRainmaker:
		rule = "hoko"
	case RuleClamBlitz:
		rule = "asari"
	}
	payload.Rule = rule

	/* STAGE */
	payload.Stage = "#" + battle.Stage.ID

	/* WEAPON */
	payload.Weapon = "#" + battle.PlayerResult.Player.Weapon.ID

	/* RESULT */
	result := battle.MyTeamResult.Key
	switch result {
	case "victory":
		payload.Result = "win"
	case "defeat":
		payload.Result = "lose"
	}

	myPercent := battle.MyTeamPercentage
	theirPercent := battle.OtherTeamPercentage

	/* TEAM PERCENTS/COUNTS */
	myCount := battle.MyTeamCount
	theirCount := battle.OtherTeamCount

	switch battle.Type {
	case "regular", "fes":
		payload.MyTeamPercent = myPercent
		payload.HisTeamPercent = theirPercent
	case "gachi", "league":
		payload.MyTeamCount = myCount
		payload.HisTeamCount = theirCount

		if myCount == 100 || theirCount == 100 {
			payload.KnockOut = "yes"
		} else {
			payload.KnockOut = "no"
		}
	}

	/* TURF INKED */
	turfInked := battle.PlayerResult.GamePaintPoint // without bonus
	var myPoint int
	if rule == "turf_war" {
		if result == "victory" {
			myPoint = turfInked + 1000 // win bonus
		} else {
			myPoint = turfInked
		}
	} else {
		myPoint = turfInked
	}
	payload.MyPoint = myPoint

	/* KILLS, ETC */
	payload.Kill = battle.PlayerResult.KillCount
	payload.KillOrAssist = battle.PlayerResult.KillCount + battle.PlayerResult.AssistCount
	payload.Special = battle.PlayerResult.SpecialCount
	payload.Death = battle.PlayerResult.DeathCount

	/* LEVEL */
	payload.Level = battle.PlayerResult.Player.PlayerRank // this is before
	payload.LevelAfter = battle.PlayerRank
	payload.StarRank = battle.StarRank

	/* RANK */
	if rule != "turf_war" {
		// only upload if ranked
		payload.RankAfter = strings.ToLower(battle.Udemae.Name)
		payload.Rank = strings.ToLower(battle.PlayerResult.Player.Udemae.Name)
		payload.RankExpAfter = battle.Udemae.SPlusNumber
		payload.RankExp = battle.PlayerResult.Player.Udemae.SPlusNumber
	}

	if battle.Udemae.IsX {
		payload.XPowerAfter = battle.XPower
		if mode == "gachi" {
			payload.EstimateXPower = battle.EstimateXPower
		}
		// worldwide rank ignored, not seen in doc
	}

	var elapsedTime = battle.ElapsedTime
	if elapsedTime == 0 {
		elapsedTime = 180 // turf war - 3 miutes in seconds
	}
	payload.StartAt = battle.StartTime
	payload.EndAt = battle.StartTime + elapsedTime

	/* SPLATNET DATA */
	payload.PrivateNote = "Battle #" + battle.BattleNumber
	payload.SplatnetNumber, _ = strconv.Atoi(battle.BattleNumber)
	switch mode {
	case "league":
		payload.MyTeamID = battle.TagID
		payload.LeaguePoint = battle.LeaguePoint
		payload.MyTeamEstimateLeaguePoint = battle.MyEstimateLeaguePoint
		payload.HisTeamEstimateLeaguePoint = battle.OtherEstimateLeaguePoint
	case "gachi":
		payload.EstimateGachiPower = battle.EstimateGachiPower
	case "regular":
		payload.Freshness = battle.WinMeter
	}

	payload.Gender = battle.PlayerResult.Player.PlayerType.Style
	payload.Species = battle.PlayerResult.Player.PlayerType.Species

	/* SPLATFEST TITLES/POWER */
	/* SPLATFEST VER.4 */
	if battle.Type == "fes" {
		// sorry, know nothing about splatoon fest rank
		// there is a lot of logic going on here, since splatoon 2 fest is no longer held
		// I omit this section for safety
		// See also https://github.com/frozenpandaman/splatnet2statink/blob/master/splatnet2statink.py
	}

	/* SCOREBOARD */
	myStat := battle.PlayerResult.AsStat(true, true, &battle)
	err := s.setScoreBoard(myStat, &battle, &payload)
	if err != nil {
		return err
	}

	/* IMAGE RESULT */
	if !s.Debug {
		// normal scoreboard
		im, err := s.Splatoon.shareImageResult(battle.BattleNumber)
		if err != nil {
			return fmt.Errorf("error getting share image result: %w", err)
		}
		if s.Anonymous {
			im = blackout(im, getMasks(&battle, payload.Players), true)
		}
		payload.ImageResult = imageToBytes(im)
		if sendGears {
			stage, _ := strconv.Atoi(battle.Stage.ID)
			im, err = s.Splatoon.shareImageProfile(stage, profileColorsEnum[rand.Intn(6)])
			payload.ImageGear = imageToBytes(im)
		}
	}

	/* GEAR */
	payload.Gears.Headgear.Gear = "#" + battle.PlayerResult.Player.Head.ID
	payload.Gears.Clothing.Gear = "#" + battle.PlayerResult.Player.Clothes.ID
	payload.Gears.Shoes.Gear = "#" + battle.PlayerResult.Player.Shoes.ID

	/* Abilities */
	payload.Gears.Headgear.PrimaryAbility = abilitiesEnum[util.MustAtoi(battle.PlayerResult.Player.HeadSkills.Main.ID)]
	payload.Gears.Clothing.PrimaryAbility = abilitiesEnum[util.MustAtoi(battle.PlayerResult.Player.ClothesSkills.Main.ID)]

	payload.Gears.Headgear = statinkGearFromPlayer(battle.PlayerResult.Player.Head, battle.PlayerResult.Player.HeadSkills)
	payload.Gears.Clothing = statinkGearFromPlayer(battle.PlayerResult.Player.Clothes, battle.PlayerResult.Player.ClothesSkills)
	payload.Gears.Shoes = statinkGearFromPlayer(battle.PlayerResult.Player.Shoes, battle.PlayerResult.Player.ShoesSkills)

	if !dryRun {
		/* OUTPUT */
		// POST to stat.ink
		// https://github.com/fetus-hina/stat.ink/blob/master/doc/api-2/request-body.md
		// msgpack encoding is favored as we have binary image data in request
		// disable redirect we can check for status 302 uploaded properly
		res, err := postHTTPMsgPack(endpointStatinkUploadBattle, payload, authBearToken(apiKey), clientNoRedirect)
		if err != nil {
			return err
		}
		switch res.StatusCode {
		case http.StatusCreated: //ok
		case http.StatusFound:
			// battle already uploaded
			return fmt.Errorf("battle %s already uploaded", battle.BattleNumber)
		default:
			body, _ := ioutil.ReadAll(res.Body)
			return fmt.Errorf("battle upload failed: stat.ink rejects with status %s, body:\n%s", res.Status, body)
		}
	}
	return nil
}

func (s *StatInk) setScoreBoard(me Stat, b *Battle, payload *battlePayload) error {
	// request battle data with members of both sides
	battleData, err := s.Splatoon.GetBattle(b.BattleNumber)
	if err != nil {
		return err
	}

	var allyBoard []Stat
	for _, member := range battleData.MyTeamMembers {
		stat := member.AsStat(true, false, b)
		allyBoard = append(allyBoard, stat)
	}

	// add me
	allyBoard = append(allyBoard, me)

	// scoreboard sort order: sort_score (or turf inked), k+a, specials, deaths (more = better), kills, nickname
	// discussion: https://github.com/frozenpandaman/splatnet2statink/issues/6
	var sorter lessStats
	if b.Rule.Key == "turf_war" {
		// 8 1 3 4 2 11
		sorter = chainItemGetters(
			getPoint, // only this one is different
			getKoA,
			getSpecial,
			getDeath,
			getKill,
			getName,
		)
	} else {
		// 0 1 3 4 2 11
		sorter = chainItemGetters(
			getSortScore,
			getKoA,
			getSpecial,
			getDeath,
			getKill,
			getName,
		)
	}
	// sort reverse
	sort.Slice(allyBoard, func(i, j int) bool {
		return !sorter(allyBoard[i], allyBoard[j])
	})

	for n := range allyBoard {
		if allyBoard[n].IsMe == "yes" {
			payload.RankInTeam = n + 1
		}
	}

	var enemyBoard []Stat
	// now enemies
	for _, member := range battleData.OtherTeamMembers {
		stat := member.AsStat(false, false, b)
		enemyBoard = append(enemyBoard, stat)
	}

	sort.Slice(enemyBoard, func(i, j int) bool {
		return !sorter(enemyBoard[i], enemyBoard[j])
	})

	var fullScoreBoard = append(allyBoard, enemyBoard...)

	for n := range fullScoreBoard {
		if fullScoreBoard[n].IsMe != "yes" && s.Anonymous {
			fullScoreBoard[n].Name = digestUsername(fullScoreBoard[n].Name)
		}

		// set index in team
		if n < 4 {
			// ally
			fullScoreBoard[n].RankInTeam = n + 1
		} else {
			// enemy
			fullScoreBoard[n].RankInTeam = n - 3
		}
	}

	// we should already have our original json if we're using debug mode
	payload.SplatnetJson = *b
	payload.Players = fullScoreBoard
	return nil
}

func getMasks(battle *Battle, players []Stat) [8]uint8 {
	// blackout everyone except me
	masks := [...]uint8{0, 0, 0, 0, 0, 0, 0, 0}
	// victory team在上面，如果我们队lose了，要调换前四个和后四个
	scoreboard := make([]Stat, 8)
	if battle.Win() {
		copy(scoreboard, players)
	} else {
		copy(scoreboard[:4], players[4:])
		copy(scoreboard[4:], players[:4])
	}
	for i := range scoreboard {
		if scoreboard[i].IsMe == "yes" {
			masks[i] = 1
		}
	}
	return masks
}

// this masks player names in result share image
// if polygon is true, use polygon path to align mask with score lines
// else, draw stdlib rectangles that are orthogonal
func blackout(origin image.Image, player [8]uint8, polygon bool) image.Image {
	// copy content from im
	canvas := image.NewRGBA(origin.Bounds())
	// 绘制原始图层
	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	draw.Draw(canvas, origin.Bounds(), origin, image.ZP, draw.Src)

	for i := range player {
		if player[i] == 0 {
			if polygon {
				maskPlayerPolygon(canvas, i, black)
			} else {
				maskPlayer(canvas, i, black)
			}
		}
	}
	return canvas
}

func authBearToken(token string) http.Header {
	auth := makeHeader(map[string]string{
		"Authorization": "Bearer " + token, // stat.ink auth
	})
	return auth
}

// note: standard library does NOT support draw polygon
// so the rectangular masks work but are a bit ugly
// if polygon masks are required, use a third-party library
func maskPlayer(canvas draw.Image, i int, c color.Color) {
	var region image.Rectangle
	switch i {
	case 0:
		region = image.Rect(719, 101, 849, 119)
	case 1:
		region = image.Rect(721, 151, 851, 169)
	case 2:
		region = image.Rect(723, 201, 853, 219)
	case 3:
		region = image.Rect(725, 251, 855, 269)
	case 4:
		region = image.Rect(725, 379, 855, 406)
	case 5:
		region = image.Rect(723, 429, 853, 456)
	case 6:
		region = image.Rect(721, 479, 851, 506)
	case 7:
		region = image.Rect(719, 529, 849, 556)
	}
	draw.Draw(canvas, region, &image.Uniform{C: c}, image.ZP, draw.Over)
}

type rectPointEnds [4]image.Point

func maskPlayerPolygon(canvas draw.Image, i int, c color.Color) {
	gc := draw2dimg.NewGraphicContext(canvas)
	var points rectPointEnds
	switch i {
	case 0:
		points = rectPointEnds{{719, 101}, {719, 123}, {849, 119}, {849, 97}}
	case 1:
		points = rectPointEnds{{721, 151}, {721, 173}, {851, 169}, {851, 147}}
	case 2:
		points = rectPointEnds{{723, 201}, {723, 223}, {853, 219}, {853, 197}}
	case 3:
		points = rectPointEnds{{725, 251}, {725, 273}, {855, 269}, {855, 247}}
	case 4:
		points = rectPointEnds{{725, 379}, {725, 401}, {855, 406}, {855, 383}}
	case 5:
		points = rectPointEnds{{723, 429}, {723, 451}, {853, 456}, {853, 433}}
	case 6:
		points = rectPointEnds{{721, 479}, {721, 501}, {851, 506}, {851, 483}}
	case 7:
		points = rectPointEnds{{719, 529}, {719, 551}, {849, 556}, {849, 533}}
	}
	gc.SetFillColor(c)
	path := new(draw2d.Path)
	for n := range points {
		path.LineTo(float64(points[n].X), float64(points[n].Y))
	}
	path.Close() // go back to first point, necessary to draw a pretty polygon
	gc.Fill(path)
}

// provide <= 3 sub abilities, ignore if missing (not unlocked or not gained)
func statinkGearFromPlayer(g Gear, s WearableSkills) statinkGear {
	sg := statinkGear{
		Gear:           "#" + g.ID,
		PrimaryAbility: abilitiesEnum[util.MustAtoi(s.Main.ID)],
	}
	for i := 0; i < 3; i++ {
		if id := s.Subs[i].ID; id != "" {
			sa := abilitiesEnum[util.MustAtoi(id)]
			sg.SecondaryAbilities = append(sg.SecondaryAbilities, sa)
		}
	}
	return sg
}

func imageToBytes(im image.Image) []byte {
	content := bytes.NewBuffer(nil)
	_ = png.Encode(content, im)
	return content.Bytes()
}
