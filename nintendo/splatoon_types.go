package nintendo

import (
	"encoding/json"
	"fmt"
	"github.com/thoas/go-funk"
	"strconv"
	"strings"
	"time"
)

// endpoint "/api/schedules/
type scheduleEndpoint struct {
	Gachi   Schedule `json:"gachi"`
	League  Schedule `json:"league"`
	Regular Schedule `json:"regular"`
}

type Schedule []Session

// endpoint "/api/results"

type RecentResults struct {
	Summary struct {
		DeathCountAverage   float64 `json:"death_count_average"`
		SpecialCountAverage float64 `json:"special_count_average"`
		VictoryCount        int     `json:"victory_count"`
		Count               int     `json:"count"`
		AssistCountAverage  float64 `json:"assist_count_average"`
		VictoryRate         float64 `json:"victory_rate"`
		DefeatCount         int     `json:"defeat_count"`
		KillCountAverage    float64 `json:"kill_count_average"`
	} `json:"summary"`
	UniqueId string   `json:"unique_id"`
	Results  []Battle `json:"results"`
}

type TeamResult struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type Battle struct {
	MyTeamResult        TeamResult   `json:"my_team_result"`
	OtherTeamResult     TeamResult   `json:"other_team_result"`
	MyTeamPercentage    float64      `json:"my_team_percentage"`    // 涂地模式下有
	OtherTeamPercentage float64      `json:"other_team_percentage"` // 涂地模式下有
	Type                string       `json:"type"`
	EstimateXPower      int          `json:"estimate_x_power"`
	StarRank            int          `json:"star_rank"`
	WeaponPaintPoint    int          `json:"weapon_paint_point"`
	XPower              float64      `json:"x_power"`
	Udemae              Udemae       `json:"udemae"`
	EstimateGachiPower  int          `json:"estimate_gachi_power"`
	BattleNumber        string       `json:"battle_number"`
	PlayerRank          int          `json:"player_rank"`
	Rank                interface{}  `json:"rank"`
	ElapsedTime         int64        `json:"elapsed_time,omitempty"`
	Rule                Rule         `json:"rule"`
	StartTime           int64        `json:"start_time"`
	Stage               Stage        `json:"stage"`
	CrownPlayers        []string     `json:"crown_players"` // contain crown player ids
	PlayerResult        PlayerResult `json:"player_result"`
	GameMode            GameMode     `json:"game_mode"`
	OtherTeamCount      int          `json:"other_team_count"` // 真格模式下有
	MyTeamCount         int          `json:"my_team_count"`    // 真格模式下有

	TagID                    string  `json:"tag_id"`
	LeaguePoint              float64 `json:"league_point"`
	MyEstimateLeaguePoint    int     `json:"my_estimate_league_point"`
	OtherEstimateLeaguePoint int     `json:"other_estimate_league_point"`
	WinMeter                 float64 `json:"win_meter"` // freshness

	// these data are only populated in battle endpoint
	// https://app.splatoon2.nintendo.net/api/results/{battleNumber}
	MyTeamMembers    []PlayerResult `json:"my_team_members"`
	OtherTeamMembers []PlayerResult `json:"other_team_members"`
}

type GameMode struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

func (g GameMode) String() string {
	return g.Name
}

type PlayerResult struct {
	Player Player `json:"player"`
	Result
}

type Result struct {
	SpecialCount   int `json:"special_count"`
	AssistCount    int `json:"assist_count"`
	DeathCount     int `json:"death_count"`
	GamePaintPoint int `json:"game_paint_point"` //
	SortScore      int `json:"sort_score"`       // 在score board里的位置 # 0
	KillCount      int `json:"kill_count"`       // 1
}

var festRanks = map[int]string{
	0: "fanboy",
	1: "fiend",
	2: "defender",
	3: "champion",
	4: "king",
}

func (pr PlayerResult) AsStat(isMyTeam bool, isMe bool, b *Battle) Stat {
	var team string
	if isMyTeam {
		team = "my"
	} else {
		team = "his"
	}

	var me string
	if isMe {
		me = "yes"
	} else {
		me = "no"
	}

	var udemae string
	var point = pr.GamePaintPoint
	switch b.Type {
	case "gachi", "league":
		udemae = strings.ToLower(pr.Player.Udemae.Name)
	case "regular", "fes":
		if b.MyTeamResult.Win() {
			point = pr.GamePaintPoint + 1000 // bonus
		}
	}

	var isTop500 = "no"
	if b.CrownPlayers != nil && funk.ContainsString(b.CrownPlayers, pr.Player.PrincipalID) {
		isTop500 = "yes"
	}

	return Stat{
		SortScore:    pr.SortScore,
		Team:         team,
		IsMe:         me,
		Weapon:       "#" + pr.Player.Weapon.ID,
		Level:        pr.Player.PlayerRank,
		KillOrAssist: pr.KillCount + pr.AssistCount, // 1
		Kill:         pr.KillCount,                  // 2
		Death:        pr.DeathCount,                 // 4
		Special:      pr.SpecialCount,               // 3
		Point:        point,                         // 8 game_paint_point
		Name:         pr.Player.Nickname,            // 11
		SplanetID:    pr.Player.PrincipalID,         // 13
		StarRank:     pr.Player.StarRank,            // 14
		Gender:       pr.Player.PlayerType.Style,    // 15
		Species:      pr.Player.PlayerType.Species,  // 16
		Rank:         udemae,                        // 7 if gachi or league
		FestTitle:    "",                            // 12 if fes
		Top500:       isTop500,                      // 17
	}
}

// Stat select some key attributes from PlayerResult for score board display
// score board index is
type Stat struct {
	SortScore int `json:"sort_score"` // API提供的分数排名, only for sorting, no required by statink

	Team         string `json:"team"`
	IsMe         string `json:"is_me"`
	Weapon       string `json:"weapon"`
	Level        int    `json:"level"`
	RankInTeam   int    `json:"rank_in_team"`
	KillOrAssist int    `json:"kill_or_assist"` // kill + assist
	Kill         int    `json:"kill"`
	Death        int    `json:"death"`
	Special      int    `json:"special"`
	Point        int    `json:"point"` // paint point
	Name         string `json:"name"`
	SplanetID    string `json:"splanet_id"`
	StarRank     int    `json:"star_rank"`
	Gender       string `json:"gender"`
	Species      string `json:"species"`
	Rank         string `json:"rank"`       // gachi or league
	FestTitle    string `json:"fest_title"` // fest
	Top500       string `json:"top_500"`
}

func (b Battle) String() string {
	var info string
	switch b.Type {
	case "gachi", "league":
		info = b.Udemae.String()
		if b.MyTeamCount == 100 {
			info += " " + "KO BONUS!"
		} else {
			info += " " + strconv.Itoa(b.MyTeamCount) + " count"
		}
	case "regular":
		info = strconv.Itoa(b.PlayerResult.GamePaintPoint) + "p"
	}
	return fmt.Sprintf("%-7s|%-12s|%-15s|%s",
		b.MyTeamResult.Name, info, b.PlayerResult.Player.Weapon.Name, b.Stage.Name,
	)
}

// endpoint "/api/timeline"
type timelineEndpoint struct {
	Coop       interface{} `json:"coop"` // 打工模式
	OnlineShop OnlineShop  `json:"onlineshop"`
	Schedule   struct {
		Importance float64   `json:"importance"`
		Schedules  Schedules `json:"schedules"`
	}
	Stats struct {
		Importance float64  `json:"importance"`
		Recents    []Battle `json:"recents"`
	} `json:"stats"`
}

type Schedules struct {
	// 只显示一个Session
	League  []Session `json:"league"`  // 团队
	Gachi   []Session `json:"gachi"`   // 真格模式
	Regular []Session `json:"regular"` // 涂地模式
}

func (s Schedules) AnyMissing() bool {
	return len(s.League) == 0 || len(s.Gachi) == 0 || len(s.Regular) == 0
}

type RuleEnum string

const (
	RuleTurfWar      RuleEnum = "turf_war"
	RuleSplatZones            = "splat_zones"
	RuleTowerControl          = "tower_control"
	RuleRainmaker             = "rainmaker"
	RuleClamBlitz             = "clam_blitz"
)

type Rule struct {
	Key           RuleEnum `json:"key"`
	MultilineName string   `json:"multiline_name"`
	Name          string   `json:"name"`
}

func (r Rule) String() string {
	return r.Name
}

type Session struct {
	Rule     Rule `json:"rule"`
	GameMode struct {
		Name string `json:"name"` // 模式名字
		Key  string `json:"key"`  // 模式id
	} `json:"game_mode"`
	ID        int   `json:"id"`
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
	StageA    Stage `json:"stage_a"` // 两张地图
	StageB    Stage `json:"stage_b"`
}

func (s Session) String() string {
	return fmt.Sprintf(`模式: %s
时间: %s - %s
规则: %s
地图A: %s
地图B: %s
`,
		s.GameMode.Name,
		StringUnixTime(s.StartTime, "15:04"),
		StringUnixTime(s.EndTime, "15:04"),
		s.Rule.Name,
		s.StageA.Name,
		s.StageB.Name)
}

func StringUnixTime(ts int64, format string) string {
	return time.Unix(ts, 0).Format(format)
}

func StringUnixDate(ts int64) string {
	return StringUnixTime(ts, "2006-01-02")
}

type Stage Entity

type OnlineShop struct {
	Merchandise struct {
		EndTime int64       `json:"end_time"`
		ID      json.Number `json:"id"`
		Price   int         `json:"price"`
		Kind    string      `json:"kind"`
		Gear    Gear        `json:"gear"`
		Skill   Skill       `json:"skill"` // splatNet商店售卖的衣服只会有一个初始技能
	} `json:"merchandise"`
}

// endpoint "/api/records"
type recordEndpoint struct {
	Challenges interface{} `json:"challenges"` // 涂地挑战
	Festival   interface{} `json:"festival"`   // 祭典
	Records    Records     `json:"records"`
}

// Records 玩家数据
type Records struct {
	Player              Player             `json:"player"`
	RecentLoseCount     int                `json:"recent_lose_count"`
	RecentWinCount      int                `json:"recent_win_count"`
	WinCount            int                `json:"win_count"`
	LoseCount           int                `json:"lose_count"`
	StageStats          interface{}        `json:"stage_stats"` // 地图统计，key是该地图的ID
	StartTime           int64              `json:"start_time"`
	TotalPaintPointOcta int                `json:"total_paint_point_octa"`
	UniqueID            string             `json:"unique_id"`
	UpdateTime          int64              `json:"update_time"`
	WeaponStats         map[int]WeaponStat `json:"weapon_stats"` // 武器统计，key是该武器的ID
}

// Udemae 真格模式
type Udemae struct {
	IsNumberReached bool   `json:"is_number_reached"`
	IsX             bool   `json:"is_x"`
	Name            string `json:"name"`
	Number          int    `json:"number"`
	SPlusNumber     int    `json:"s_plus_number"`
}

type Entity struct {
	ID    string   `json:"id"`
	Image ImageURL `json:"image"`
	Name  string   `json:"name"`
}

type IntIDEntity struct {
	ID int `json:"id"`
	Entity
}

type Brand struct {
	Entity
	FrequentSkill Entity `json:"frequent_skill"`
}

type Gear struct {
	Entity
	Brand     Brand    `json:"brand"`
	Kind      string   `json:"kind"`
	Rarity    int      `json:"rarity"`
	Thumbnail ImageURL `json:"thumbnail"`
}

type Skill Entity

type WearableSkills struct {
	Main Skill   `json:"main"` // 1.0 up
	Subs []Skill `json:"subs"` // 0.1 up
}

// Character 角色
type Character struct {
	Species string `json:"species"` // octolings章鱼
	Style   string `json:"style"`   // 性别
}

// Player 玩家数据
type Player struct {
	// this field is not necessarily populated, for example in battle stats
	*RankingModeStat

	Udemae             Udemae  // battle-specific Udemae
	StarRank           int     `json:"star_rank"`
	MaxLeaguePointPair float64 `json:"max_league_point_pair"`
	MaxLeaguePointTeam float64 `json:"max_league_point_team"`

	Nickname    string    `json:"nickname"` // 玩家昵称
	PlayerRank  int       `json:"player_rank"`
	PlayerType  Character `json:"player_type"`
	PrincipalID string    `json:"principal_id"`

	Head       Gear           `json:"head"`
	HeadSkills WearableSkills `json:"head_skills"`

	Clothes       Gear           `json:"clothes"`
	ClothesSkills WearableSkills `json:"clothes_skills"`

	Shoes       Gear           `json:"shoes"`
	ShoesSkills WearableSkills `json:"shoes_skills"`

	Weapon Weapon `json:"weapon"` // 玩家当前使用的武器
}

type RankingModeStat struct {
	UdemaeClam      Udemae `json:"udemae_clam"`
	UdemaeRainmaker Udemae `json:"udemae_rainmaker"`
	UdemaeTower     Udemae `json:"udemae_tower"`
	UdemaeZones     Udemae `json:"udemae_zones"`
}

type Special struct {
	ID     string   `json:"id"`
	ImageA ImageURL `json:"image_a"`
	ImageB ImageURL `json:"image_b"`
	Name   string   `json:"name"`
}

type Sub struct {
	ID     string   `json:"id"`
	ImageA ImageURL `json:"image_a"`
	ImageB ImageURL `json:"image_b"`
	Name   string   `json:"name"`
}

type Weapon struct {
	Entity
	Special   Special  `json:"special"`   // 大招
	Sub       Sub      `json:"sub"`       // 副武器
	Thumbnail ImageURL `json:"thumbnail"` // 副图
}

type WeaponStat struct {
	Weapon Weapon `json:"weapon"`

	LastUseTime     int64   `json:"last_use_time"`
	LoseCount       int     `json:"lose_count"`
	MaxWinMeter     float64 `json:"max_win_meter"`
	TotalPaintPoint int     `json:"total_paint_point"`
	WinCount        int     `json:"win_count"`
	WinMeter        float64 `json:"win_meter"`
}

// ImageURL 图片资源
type ImageURL string
