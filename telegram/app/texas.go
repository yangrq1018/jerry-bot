package app

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/enescakir/emoji"
	"github.com/yangrq1018/jerry-bot/telegram"
	"github.com/yangrq1018/jerry-bot/telegram/texas"
	"github.com/yangrq1018/jerry-bot/util"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

// query data
var (
	spade   = "spade"
	heart   = "heart"
	club    = "club"
	diamond = "diamond"
	clear   = "clear"
)

var (
	spadeBtn   = tgbotapi.InlineKeyboardButton{CallbackData: &spade, Text: "Spade"}
	heartBtn   = tgbotapi.InlineKeyboardButton{CallbackData: &heart, Text: "Heart"}
	clubBtn    = tgbotapi.InlineKeyboardButton{CallbackData: &club, Text: "Club"}
	diamondBtn = tgbotapi.InlineKeyboardButton{CallbackData: &diamond, Text: "Diamond"}
	clearBtn   = tgbotapi.InlineKeyboardButton{CallbackData: &clear, Text: "clear"}
)

// stage
const (
	pickSuite            = "pick suite of card"
	pickRank             = "pick rank of card"
	userAskedReset       = "cards reset"
	sevenCardResetStatus = "you typed 7 cards, cards are reset to empty"
)

var suitePickKeyboard = [][]tgbotapi.InlineKeyboardButton{
	{
		spadeBtn,
		heartBtn,
		clubBtn,
		diamondBtn,
	},
	{clearBtn},
}

var rankPickKeyboard = newRankPickKeyboard()

func newRankPickKeyboard() [][]tgbotapi.InlineKeyboardButton {
	keyboards := make([][]tgbotapi.InlineKeyboardButton, 4)
	for i := 2; i <= 14; i++ {
		row := (i - 2) / 4 // 4 by 4
		keyboards[row] = append(keyboards[row], tgbotapi.InlineKeyboardButton{
			CallbackData: util.StringPtr(fmt.Sprintf("rank %s", strconv.Itoa(i))),
			Text:         texas.Rank(i).String(),
		})
	}
	keyboards[3] = append(keyboards[3], clearBtn)
	return keyboards
}

type Cards []texas.Card

func emojiSuite(s texas.Suite) string {
	var representation string
	switch s {
	case texas.Spade:
		representation = emoji.SpadeSuit.String()
	case texas.Heart:
		representation = emoji.HeartSuit.String()
	case texas.Club:
		representation = emoji.ClubSuit.String()
	case texas.Diamond:
		representation = emoji.DiamondSuit.String()
	}
	return representation
}

func (cs Cards) String() string {
	var acc []string
	if len(cs) == 0 {
		return ""
	}
	for i := range cs {
		acc = append(acc, emojiSuite(cs[i].Suite)+" "+cs[i].Rank.String())
	}
	return strings.Join(acc, " ")
}

type SyncCards struct {
	lock      sync.Mutex
	cards     []texas.Card
	temp      texas.Card // holder for a card with suite typed only
	temSet    bool
	histogram string
}

const initialHistogram = "Probabilities: input more to calculate..."

func (s *SyncCards) PushCard(c texas.Card) {
	s.lock.Lock()
	s.cards = append(s.cards, c)
	s.lock.Unlock()
}

func (s *SyncCards) CardsString() string {
	sb := s.Cards().String()
	if s.temSet {
		sb += " " + emojiSuite(s.temp.Suite)
	}
	return sb
}

func (s *SyncCards) Cards() Cards {
	s.lock.Lock()
	cards := make(Cards, len(s.cards))
	copy(cards, s.cards)
	s.lock.Unlock()
	return cards
}

func (s *SyncCards) Reset() {
	s.lock.Lock()
	s.cards = s.cards[:0]
	s.histogram = initialHistogram
	s.lock.Unlock()
}

func (s *SyncCards) SetRank(r texas.Rank) {
	s.temSet = false
	s.lock.Lock()
	s.temp.Rank = r
	s.lock.Unlock()
}

func (s *SyncCards) SetSuite(suite texas.Suite) {
	s.temSet = true
	s.lock.Lock()
	s.temp.Suite = suite
	s.lock.Unlock()
}

func (s *SyncCards) Len() int {
	return len(s.cards)
}

type playTexas struct {
	userSCMap     map[int]*SyncCards
	userSCMapLock sync.Mutex
}

func (p *playTexas) ID() tgbotapi.BotCommand {
	return tgbotapi.BotCommand{
		Command:     "play",
		Description: "计算德州hand概率",
	}

}

func (p *playTexas) Serve(bot *telegram.Bot) error {
	bot.Match(p).Subscribe(p.handle)
	bot.CallBackQueryEvent.Subscribe(p.handleQueryCallback)
	return nil
}

func (p *playTexas) Init() {}

func (p *playTexas) Authorize() telegram.Authorizer {
	return telegram.PolicyAllow
}

func (p *playTexas) handle(b *telegram.Bot, u tgbotapi.Update) error {
	msg := tgbotapi.NewMessage(u.Message.Chat.ID, fmt.Sprintf("Start game! let's pick a suite for the first card first"))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(suitePickKeyboard...)
	// reset as the user types the command again
	if sc, ok := p.userSCMap[u.Message.From.ID]; ok {
		sc.Reset()
	}
	_, err := b.Bot().Send(msg)
	return err
}

func (p *playTexas) handleQueryCallback(b *telegram.Bot, query tgbotapi.CallbackQuery) error {
	var sc *SyncCards
	var ok bool
	if _, ok = p.userSCMap[query.From.ID]; !ok {
		p.userSCMapLock.Lock()
		p.userSCMap[query.From.ID] = new(SyncCards)
		p.userSCMap[query.From.ID].histogram = initialHistogram
		p.userSCMapLock.Unlock()
	}
	sc = p.userSCMap[query.From.ID]
	refreshKeyboard := func(b *telegram.Bot, query *tgbotapi.CallbackQuery, keyboard [][]tgbotapi.InlineKeyboardButton, msg string) {
		b.Bot().Send(tgbotapi.NewEditMessageTextAndMarkup(
			query.Message.Chat.ID,
			query.Message.MessageID,
			fmt.Sprintf(`
%s
%s
%s`, sc.CardsString(), strings.Trim(sc.histogram, "\n"), msg), // alter message text
			tgbotapi.NewInlineKeyboardMarkup(keyboard...), // alter message inline keyboard
		))
	}

	reset := func(status string) {
		sc.Reset()
		refreshKeyboard(b, &query, suitePickKeyboard, status)
	}

	switch data := query.Data; {
	case data == clear:
		reset(userAskedReset)
		return nil
	case len(data) > 4 && data[:4] == "rank":
		// pick rank
		var rank int
		_, err := fmt.Sscanf(query.Data, "rank %d", &rank)
		if err != nil {
			return err
		}
		sc.SetRank(texas.Rank(rank))
		sc.PushCard(sc.temp)
		// calculate histogram
		if sc.Len() > 1 { // calculate one card histogram is too slow
			// display a banner message (calculating, please wait)
			b.Bot().AnswerCallbackQuery(tgbotapi.CallbackConfig{
				CallbackQueryID: query.ID,
				Text:            "calculating",
			})
			sc.histogram = formatHistogram(texas.HistogramHandTypes(sc.Cards()))
		}

		if sc.Len() == 7 {
			// if the user gives more than 7 cards, reset cards to zero
			reset(sevenCardResetStatus)
			return nil
		}
		// switch tp suite keyboard
		refreshKeyboard(b, &query, suitePickKeyboard, pickSuite)
	default:
		switch data {
		case spade:
			sc.SetSuite(texas.Spade)
		case heart:
			sc.SetSuite(texas.Heart)
		case club:
			sc.SetSuite(texas.Club)
		case diamond:
			sc.SetSuite(texas.Diamond)
		default:
			return nil
		}
		// switch to rank keyboard
		refreshKeyboard(b, &query, rankPickKeyboard, pickRank)
	}
	return nil
}

func TexasPlayCommand() telegram.Command {
	return &playTexas{
		userSCMap: make(map[int]*SyncCards),
	}
}

func TexasHistogramCommand() telegram.Command {
	return SimpleCommand{
		name:        "hist",
		description: "texas hold'em utility: histogram. Example: /hist <Suite> <Rank>, <Suite> <Rank>",
		handle: func(b *telegram.Bot, u tgbotapi.Update) error {
			holeCards := strings.Split(u.Message.CommandArguments(), ",")
			for i := range holeCards {
				holeCards[i] = strings.Trim(holeCards[i], " ")
			}
			known, err := texas.CardsFromStrings(holeCards)
			if err != nil {
				return err
			}
			hist := texas.HistogramHandTypes(known)
			b.ReplyTo(*u.Message, formatHistogram(hist))
			return nil
		},
	}
}

func formatHistogram(hist []texas.HandCardsProbability) string {
	sb := strings.Builder{}
	for _, h := range hist {
		sb.WriteString(fmt.Sprintf("%s: %.2f%%, acc: %.2f%%\n", h.Hand.String(), h.Prob*100, h.AccProb*100))
	}
	return sb.String()
}
