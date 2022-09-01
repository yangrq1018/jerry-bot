package app

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/enescakir/emoji"
	"github.com/go-co-op/gocron"
	"github.com/thoas/go-funk"
	"github.com/yangrq1018/jerry-bot/gcloud"
	"github.com/yangrq1018/jerry-bot/nintendo"
	"github.com/yangrq1018/jerry-bot/telegram"
	"github.com/yangrq1018/jerry-bot/util"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

var logger = telegram.GetModuleLogger("splatoon")

const (
	maxWeaponDisplayed = 3
)

// SafeStore thread-safe map to store user int id -> string
type SafeStore[K comparable, V any] struct {
	m sync.Map
}

func (s *SafeStore[K, V]) Set(k K, v V) {
	s.m.Store(k, v)
}

func (s *SafeStore[K, V]) Get(k K) (V, bool) {
	v, ok := s.m.Load(k)
	if !ok {
		var t V
		return t, ok
	}
	return v.(V), ok
}

func uploadHandler(splat *nintendo.SplatoonService, callback func(battle nintendo.Battle, err error)) (int, error) {
	ink := nintendo.StatInk{
		Mode:      "tg-bot",
		Splatoon:  splat,
		Anonymous: true,
	}
	battles, err := splat.Recent50Battles()
	if err != nil {
		return 0, fmt.Errorf("error get recent 50 battles: %w", err)
	}

	statInkUploaded, err := nintendo.GetStatinkBattles()
	if err != nil {
		return 0, fmt.Errorf("error get stat ink battles: %w", err)
	}
	var uploadedCount int
	for n := len(battles) - 1; n >= 0; n-- {
		// check stat.ink uploaded
		if funk.ContainsInt(statInkUploaded, util.MustAtoi(battles[n].BattleNumber)) {
			continue
		}
		err = ink.PostBattleToStatink(battles[n], n == 0, false)
		if err != nil {
			err = fmt.Errorf("error post battle #%s to stat ink: %w", battles[n].BattleNumber, err)
		} else {
			uploadedCount++
		}
		if callback != nil {
			callback(battles[n], err)
		}
		if err != nil {
			break
		}
	}
	return uploadedCount, nil
}

func genCookieAndStore(b *telegram.Bot, u tgbotapi.Update, store *SafeStore[int, string], sessionToken string, fromID int) (string, error) {
	b.ReplyTo(*u.Message, "Generate new cookie for telegram user %s...", u.Message.From.UserName)
	nickname, cookie, err := nintendo.GetCookie(sessionToken, "")
	if err != nil {
		return "", err
	}
	b.ReplyTo(*u.Message,
		`Done! 为用户%s生成cookie:
%s
已经存储你的Cookie%s，下次可以直接使用/splatoon命令, enjoy!`, nickname, cookie, emoji.Cookie)
	store.Set(fromID, cookie)
	return cookie, nil
}

func authNewUser(authCodeStore *SafeStore[int, string], sessionTokenStore *SafeStore[int, string], match []string, userID int) (string, error) {
	if len(match) == 2 && match[1] != "" {
		link := match[1]
		authCode, ok := authCodeStore.Get(userID)
		if !ok {
			return "", fmt.Errorf("auth code not found, please try again")
		}
		sessionToken, err := nintendo.ExtractSessionToken(link, authCode)
		if err != nil {
			return "", err
		}
		sessionTokenStore.Set(userID, sessionToken)
		return fmt.Sprintf("已存储session token: %0.10s，下次可以直接登录", sessionToken), nil
	} else {
		// ask for token
		postLoginURL, authCode, err := nintendo.GeneratePostLogin()
		if err != nil {
			return "", err
		}
		msg := fmt.Sprintf(
			`服务器上没有你的Cookie或者Session token记录，所以无法查询你的账户信息。
请在<b>电脑端</b>浏览器打开<a href="%s">选择联动账号</a>
（如果需要登录，输入账号密码登录），在账户列表中，右键"选择此人"按钮，复制链接地址。
然后发送下面格式的信息给我

/splatoon cookie [copied address]

[copied address]是你复制下来的内容

如
/splatoon cookie npf71b963c1b7b6d119://auth#session_state=c1508b5740ed9888...

注意：
- 你发送的链接应该以
npf71b963c1b7b6d119://auth
开头
`,
			postLoginURL,
		)
		authCodeStore.Set(userID, authCode)
		return msg, nil
	}
}

type splatoonCommand struct {
	sessionTokenStore SafeStore[int, string]
	cookieStore       SafeStore[int, string]
	authCodeStore     SafeStore[int, string]
	uploadMu          sync.Mutex
}

func (s *splatoonCommand) ID() tgbotapi.BotCommand {
	return tgbotapi.BotCommand{
		Command:     "splatoon",
		Description: "splatoon 2命令",
	}
}

func (s *splatoonCommand) Serve(bot *telegram.Bot) error {
	err := s.init(bot)
	if err != nil {
		return err
	}
	bot.TeardownEvent.Subscribe(s.teardown)
	bot.Match(s).Subscribe(s.handle)
	return nil
}

func (s *splatoonCommand) Init() {
	// set a session token, then we can generate cookie on the fly infinitely
	s.sessionTokenStore.Set(telegram.DeveloperChatIDint, nintendo.DefaultSessionToken())
}

func (s *splatoonCommand) Authorize() telegram.Authorizer {
	return telegram.PolicyAllow
}

func (s *splatoonCommand) teardown(*telegram.Bot, os.Signal) error {
	cookies := make(map[int]string)
	s.cookieStore.m.Range(func(k, v interface{}) bool {
		cookies[k.(int)] = v.(string)
		return true
	})
	logger.Infof("save %d cookies to gcloud", len(cookies))
	return gcloud.SaveObject(gcloudObjectKeySplatoon, cookies)
}

func (s *splatoonCommand) init(b *telegram.Bot) error {
	scheduler := gocron.NewScheduler(time.Local)
	// 每隔1小时定时上传一次结果
	interval := 60 * time.Minute
	// 每天允许允许的时间，防止夜间提醒
	allowHourMin, allowHourMax := 8, 23
	splatDefault, err := func() (*nintendo.SplatoonService, error) {
		m := make(map[int]string)
		err := gcloud.LoadObject(gcloudObjectKeySplatoon, &m)
		if err == nil {
			logger.Infof("splatoon: loading %d cookies from gcloud", len(m))
			c := m[telegram.DeveloperChatIDint]
			if c == "" {
				return nil, fmt.Errorf("because developer has no cookie")
			}
			// set a session token, then we can generate cookie on the fly infinitely
			s.cookieStore.Set(telegram.DeveloperChatIDint, c)
			return nintendo.NewClient(c).Splatoon(), nil
		}
		return nil, err
	}()
	if err != nil {
		logger.Infof("[WARN] splatoon cannot start scheduler: %v", err)
		return nil
	}
	_, err = scheduler.
		Every(interval).
		StartAt(time.Now().Add(interval)). // start 1 hour from now
		Do(func() {
			// 只在白天运行
			hour := time.Now().Hour()
			if hour < allowHourMin || hour > allowHourMax {
				return
			}
			uploaded, err := uploadHandler(splatDefault, nil)
			if err != nil {
				logger.Error(err)
			}
			if uploaded != 0 {
				b.Bot().Send(tgbotapi.NewMessage(telegram.DeveloperChatID,
					fmt.Sprintf("uploaded %d battles to stat ink", uploaded)))
			}
		})
	if err != nil {
		return err
	}
	scheduler.StartAsync()
	return nil
}

func (s *splatoonCommand) handle(b *telegram.Bot, u tgbotapi.Update) error {
	var splat *nintendo.SplatoonService
	arguments := u.Message.CommandArguments()
	fromId := u.Message.From.ID
	// auth
	// try to authenticate the user
	if cookie, ok := s.cookieStore.Get(fromId); ok {
		splat = nintendo.NewClient(cookie).Splatoon()
		goto authDone
	}
	if sessionToken, ok := s.sessionTokenStore.Get(fromId); ok {
		cookie, err := genCookieAndStore(b, u, &s.cookieStore, sessionToken, fromId)
		if err != nil {
			return err
		}
		splat = nintendo.NewClient(cookie).Splatoon()
		goto authDone
	}
	if text, err := authNewUser(&s.authCodeStore, &s.sessionTokenStore, regexp.MustCompile(`cookie\s?(\S*)`).FindStringSubmatch(arguments), fromId); err != nil {
		return err
	} else {
		msg := tgbotapi.NewMessage(int64(fromId), text)
		msg.ParseMode = "HTML"
		_, err = b.Bot().Send(msg)
		return err
	}

authDone:
	switch arguments {
	case "":
		// send reply keyboard with commands
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, `pick a command from the keyboard.
- schedule: 之后6个真格session安排
- player: 查询真格段位查询
- session: 查询当前3个模式的规则和地图
- battles: 查询最近50场战斗
- weapon: 查询前三个使用最多武器
- account: 查询玩家任天堂账户信息
- upload: 上次最近五十次battle到stat.ink（自动去重），需要提供你的 stat.ink API key
`)
		row := func(text string) []tgbotapi.KeyboardButton {
			return tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(text))
		}
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			row("/splatoon schedule"),
			row("/splatoon player"),
			row("/splatoon session"),
			row("/splatoon battles"),
			row("/splatoon weapon"),
			row("/splatoon account"),
			row("/splatoon upload"),
			row("/splatoon regen"),
		)
		_, err := b.Bot().Send(msg)
		if err != nil {
			return err
		}
	case "regen":
		if sessionToken, ok := s.sessionTokenStore.Get(fromId); ok {
			_, err := genCookieAndStore(b, u, &s.cookieStore, sessionToken, fromId)
			if err != nil {
				return err
			}
		} else {
			b.ReplyTo(*u.Message, "Sorry, we don't have your token. send \"/splatoon cookie\" to get one")
		}
		return nil
	case "upload":
		// mutex sync the user presses command too fast, causing multiple routines to upload
		b.ReplyTo(*u.Message, "开始上传战斗，请等待...")
		s.uploadMu.Lock()

		callback := func(battle nintendo.Battle, err error) {
			if err == nil {
				startTime := time.Unix(battle.StartTime, 0)
				date := startTime.Format("01-02 15:04")
				b.ReplyTo(*u.Message,
					"上传战斗：编号#%s 时间%s 模式%s 规则%s 结果%s",
					battle.BattleNumber, date,
					battle.GameMode, battle.Rule, battle.MyTeamResult.Name)
			} else {
				b.ReplyTo(*u.Message, "上传战斗：编号#%s 失败：%v", battle.BattleNumber, err)
			}
		}
		uploaded, err := uploadHandler(splat, callback)
		s.uploadMu.Unlock()
		if err != nil {
			return err
		}
		var msg string
		msg = fmt.Sprintf("成功上传%d场战斗", uploaded)
		_, err = b.Bot().Send(tgbotapi.NewMessage(telegram.DeveloperChatID, msg))
		return err
	case "account":
		sessionToken, ok := s.sessionTokenStore.Get(fromId)
		if !ok {
			return fmt.Errorf("I need a session token to retrieve your account info")
		}
		user, err := nintendo.GetUserInfo(sessionToken, "")
		if err != nil {
			return err
		}
		b.ReplyTo(*u.Message, nintendo.StringifyUserInfo(user))
	case "weapon":
		records, err := splat.Records()
		if err != nil {
			return err
		}
		var (
			msg tgbotapi.PhotoConfig
		)
		for i, ws := range records.SortWeapons(nintendo.CompareWeaponMostUsed) {
			if i >= maxWeaponDisplayed {
				break
			}
			// send Photo
			msg = tgbotapi.NewPhotoUpload(
				u.Message.Chat.ID,
				*ws.Weapon.Image.URL(),
			)
			msg.Caption = ws.Usage()
			msg.ReplyToMessageID = u.Message.MessageID
			_, err = b.Bot().Send(msg)
			if err != nil {
				return err
			}
		}
	case "session":
		schedules, err := splat.CurrentSchedule()
		if err != nil {
			return err
		}
		if schedules.AnyMissing() {
			return fmt.Errorf("schedule: a mode is missing session")
		}
		for i, session := range []nintendo.Session{
			schedules.Gachi[0],   // 真格
			schedules.Regular[0], // 涂地
			schedules.League[0],  // 组排
		} {
			var caption = session.String()
			attach1 := tgbotapi.InputMediaPhoto{
				Type:  "photo",
				Media: session.StageA.Image.URL().String(),
			}
			attach2 := tgbotapi.InputMediaPhoto{
				Type:  "photo",
				Media: session.StageB.Image.URL().String(),
			}

			if i == 0 {
				// 真格模式显示玩家的段位
				playerInfo, err := splat.Player()
				var ude nintendo.Udemae
				if err == nil {
					switch session.Rule.Key {
					case nintendo.RuleRainmaker:
						ude = playerInfo.UdemaeRainmaker
					case nintendo.RuleTowerControl:
						ude = playerInfo.UdemaeTower
					case nintendo.RuleClamBlitz:
						ude = playerInfo.UdemaeClam
					case nintendo.RuleSplatZones:
						ude = playerInfo.UdemaeZones
					}
					caption += "当前模式段位: " + ude.String() + "\n"
				}
			}
			attach2.Caption = caption
			msg := tgbotapi.NewMediaGroup(u.Message.Chat.ID, []interface{}{attach1, attach2})
			msg.ReplyToMessageID = u.Message.MessageID
			_, err = b.Bot().Send(msg)
			if err != nil {
				return err
			}
		}
		return nil
	case "player":
		playerInfo, err := splat.Player()
		if err != nil {
			return err
		}
		msg := tgbotapi.NewMessage(u.Message.Chat.ID, playerInfo.RankUdemaeString())
		msg.ReplyToMessageID = u.Message.MessageID
		_, err = b.Bot().Send(msg)
		return err
	case "battles":
		rr, err := splat.Results()
		if err != nil {
			return err
		}
		var msgText strings.Builder
		msgText.WriteString("Last 50 battles\n\n")

		/* summary */
		summary := rr.Summary
		msgText.WriteString(fmt.Sprintf(`VICTORY %d | DEFEAT %d
K%.1f A%.1f S%.1f D%.1f

`,
			summary.VictoryCount, summary.DefeatCount,
			summary.KillCountAverage, summary.AssistCountAverage, summary.SpecialCountAverage, summary.DeathCountAverage,
		))

		var mode string
		var rule nintendo.RuleEnum
		for i, battle := range rr.Results {
			// determine session (a group of battles)
			if battle.GameMode.Key != mode || battle.Rule.Key != rule {
				var section string
				if battle.GameMode.Key == "regular" {
					section = battle.GameMode.Name
				} else {
					section = battle.Rule.Name
				}

				msgText.WriteString("---" + emoji.VideoGame.String() + section + "---\n")

				mode = battle.GameMode.Key
				rule = battle.Rule.Key
			}
			if i+1 < len(rr.Results) {
				prevBattle := rr.Results[i+1]
				if prevBattle.GameMode.Key == battle.GameMode.Key &&
					prevBattle.Rule.Key == battle.Rule.Key &&
					battle.Udemae.Number > prevBattle.Udemae.Number {
					// rank up!
					msgText.WriteString(emoji.UpButton.String())
				}
			}

			var isWin string
			if battle.Win() {
				isWin = emoji.VictoryHand.String()
			} else {
				isWin = emoji.DisappointedFace.String()
			}
			msgText.WriteString(isWin + battle.String() + "\n")
		}
		_, err = b.Bot().Send(tgbotapi.NewMessage(u.Message.Chat.ID, msgText.String()))
		if err != nil {
			return err
		}
	case "schedule":
		schedules, err := splat.GachiSchedule(6)
		if err != nil {
			return err
		}
		text := "真格模式时间表\n"
		for _, session := range schedules {
			text += session.String() + "\n"
		}
		msg := tgbotapi.NewPhotoUpload(u.Message.Chat.ID, *schedules[0].StageA.Image.URL())
		msg.Caption = text
		msg.ReplyToMessageID = u.Message.MessageID
		_, err = b.Bot().Send(msg)
		return err
	default:
		return fmt.Errorf("unknown command %s", arguments)
	}
	return nil
}

const gcloudObjectKeySplatoon = "splatoon"

func SplatoonCommand() telegram.Command {
	return &splatoonCommand{}
}
