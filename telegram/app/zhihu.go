package app

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/enescakir/emoji"
	"github.com/yangrq1018/jerry-bot/telegram"
	"github.com/yangrq1018/jerry-bot/telegram/zhihu"
	"github.com/yangrq1018/jerry-bot/util"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

const smallDotUnicode = "\u00b7"

var deleteMsgBtn = tgbotapi.InlineKeyboardButton{
	Text:         emoji.CrossMark.String(),
	CallbackData: util.StringPtr("msg delete"),
}

func answerPageKeyboard(q zhihu.Question, answerIndex int) tgbotapi.InlineKeyboardMarkup {
	var kbd []tgbotapi.InlineKeyboardButton
	left, right := tgbotapi.InlineKeyboardButton{
		Text: "<",
	}, tgbotapi.InlineKeyboardButton{
		Text: ">",
	}
	if answerIndex-1 >= 0 {
		left.CallbackData = util.StringPtr(fmt.Sprintf("q %s %d", q.QID(), answerIndex-1))
		kbd = append(kbd, left)
	}
	if answerIndex+1 < len(q.Answer) {
		right.CallbackData = util.StringPtr(fmt.Sprintf("q %s %d", q.QID(), answerIndex+1))
		kbd = append(kbd, right)
	}
	// delete button
	kbd = append(kbd, deleteMsgBtn)
	return tgbotapi.NewInlineKeyboardMarkup(kbd)
}

var zRegexp = regexp.MustCompile(`z_(\d+)`)

// HotStore is thread-safe question storage
type HotStore struct {
	items []zhihu.Question
}

func (h HotStore) Page(p int) string {
	// show page
	var msg strings.Builder
	for i := p * 10; i < util.Min[int]((p+1)*10, len(h.items)); i++ {
		item := h.items[i]
		// 对没有回答的问题，不显示command
		if len(item.Answer) > 0 {
			msg.WriteString(fmt.Sprintf(
				"[%d] /z_%s %s\n", i+1, item.QID(), item.Title))
		} else {
			var qt string
			switch item.Topic() {
			case zhihu.SpecialTopic:
				qt = "专题"
			case zhihu.AssessmentTopic:
				qt = "测试"
			case zhihu.RoundTableTopic:
				qt = "圆桌"
			}
			msg.WriteString(fmt.Sprintf(
				"[%d] %s %s\n", i+1, qt, item.Title))
		}
	}
	return msg.String()
}

func (h HotStore) SearchQ(qID string) (*zhihu.Question, bool) {
	for _, q := range h.items {
		if q.QID() == qID {
			return &q, true
		}
	}
	return nil, false
}

func (h *HotStore) Load() error {
	data, err := zhihu.Hot()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return fmt.Errorf("zero questions collected")
	}
	h.items = data
	return nil
}

func (h HotStore) Keyboard(nowAt int) tgbotapi.InlineKeyboardMarkup {
	kbd := make([]tgbotapi.InlineKeyboardButton, 5)
	for i := 0; i < 5; i++ {
		text := fmt.Sprintf("%d-%d", i*10+1, (i+1)*10)

		var data string
		if i == nowAt {
			text = smallDotUnicode + text + smallDotUnicode
			data = fmt.Sprintf("p %d", -1) // ignore
		} else {
			data = fmt.Sprintf("p %d", i)
		}
		kbd[i] = tgbotapi.InlineKeyboardButton{
			Text:         text,
			CallbackData: &data,
		}
	}
	return tgbotapi.NewInlineKeyboardMarkup(kbd, tgbotapi.NewInlineKeyboardRow(deleteMsgBtn))
}

type zhihuCommand struct {
	hot HotStore
}

func (z zhihuCommand) ID() tgbotapi.BotCommand {
	return tgbotapi.BotCommand{
		Command:     "zhihu",
		Description: "知乎热榜",
	}
}

func (z *zhihuCommand) Serve(bot *telegram.Bot) error {
	bot.Match(z).Subscribe(z.handle)
	bot.CallBackQueryEvent.Subscribe(z.handleQueryCallback)
	bot.UpdateEvent.Subscribe(func(b *telegram.Bot, u tgbotapi.Update) error {
		if zRegexp.MatchString(u.Message.Command()) {
			return z.handle(b, u)
		}
		return nil
	})
	return nil
}

func (z zhihuCommand) Init() {}

func (z zhihuCommand) Authorize() telegram.Authorizer {
	return telegram.PolicyAllow
}

func (z *zhihuCommand) handle(b *telegram.Bot, u tgbotapi.Update) error {
	switch cmd := u.Message.Command(); {
	case cmd == "zhihu":
		err := z.hot.Load()
		if err != nil {
			return err
		}
		m := tgbotapi.NewMessage(u.Message.Chat.ID, z.hot.Page(0))
		m.ReplyMarkup = z.hot.Keyboard(0)
		_, err = b.Bot().Send(m)
		return err
	case strings.HasPrefix(cmd, "z_"):
		// question page
		// show the first answer
		qID := zRegexp.FindStringSubmatch(cmd)[1] // 0 is the matched string itself
		if q, ok := z.hot.SearchQ(qID); ok {
			msg := tgbotapi.NewMessage(
				u.Message.Chat.ID,
				q.PrintAnswer(0),
			)
			msg.ReplyMarkup = answerPageKeyboard(*q, 0)
			msg.ParseMode = "HTML"
			_, err := b.Bot().Send(msg)
			return err
		}
		return fmt.Errorf("qid not found")
	}
	return nil
}

func (z *zhihuCommand) handleQueryCallback(b *telegram.Bot, query tgbotapi.CallbackQuery) error {
	answerQuery := func(text string) error {
		_, err := b.Bot().AnswerCallbackQuery(tgbotapi.CallbackConfig{
			CallbackQueryID: query.ID,
			Text:            text,
		})
		return err
	}
	switch data := query.Data; {
	case data == "msg delete":
		// delete this message
		_, err := b.Bot().Send(tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID))
		return err
	case strings.HasPrefix(data, "q"):
		var qID string
		var aID int
		_, err := fmt.Sscanf(data, "q %s %d", &qID, &aID)
		if err != nil {
			return fmt.Errorf("failed to scan qid: %v", err)
		}
		_ = answerQuery(fmt.Sprintf("requesting %s#%d", qID, aID))
		if q, ok := z.hot.SearchQ(qID); ok {
			msg := tgbotapi.NewEditMessageTextAndMarkup(
				query.Message.Chat.ID,
				query.Message.MessageID,
				q.PrintAnswer(aID),
				answerPageKeyboard(*q, aID),
			)
			msg.ParseMode = "HTML"
			_, err := b.Bot().Send(msg)
			return err
		} else {
			return fmt.Errorf("qid not found")
		}
	case strings.HasPrefix(data, "p"):
		var page int
		_, err := fmt.Sscanf(data, "p %d", &page)
		if err != nil {
			return fmt.Errorf("failed to scan page number: %v", err)
		}
		if page < 0 {
			// simply acknowledge
			return answerQuery("you are on this page")
		}
		_ = answerQuery(fmt.Sprintf("go to page %d", page+1))
		_, err = b.Bot().Send(tgbotapi.NewEditMessageTextAndMarkup(
			query.Message.Chat.ID,
			query.Message.MessageID,
			z.hot.Page(page),
			z.hot.Keyboard(page),
		))
		return err
	}
	return nil
}

func ZhihuCommand() telegram.Command {
	return &zhihuCommand{}
}
