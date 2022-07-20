package zhihu

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/enescakir/emoji"
	"github.com/gocolly/colly/v2"
	"github.com/yangrq1018/jerry-bot/telegram/common"
	"github.com/yangrq1018/jerry-bot/util"
	"golang.org/x/exp/utf8string"
	"golang.org/x/sync/errgroup"
)

const (
	hot           = "https://www.zhihu.com/hot"
	cookie        = `_zap=e0f6c066-64ed-4be9-bc6b-6343318203ef; d_c0="AECeG2ojGROPTiJ-OHdDVkMyzo-GKIuLbgE=|1620884519"; __snaker__id=wPv7YqWk1v33eRfr; YD00517437729195:WM_TID=AC8vlXyGRmVARBQFERNridL9NPi8eKRW; _9755xjdesxxd_=32; Hm_lvt_98beee57fd2ef70ccdd5ca52b9740c49=1627020349,1627023850,1627461229,1627463263; _xsrf=wpTLrlQK4NoxZ0ySwoRZF4915ixffBKv; capsion_ticket="2|1:0|10:1627613416|14:capsion_ticket|44:ZDY1MTI2N2Q2N2JkNDE5NTk4ZGRlZjQ0ZjhlN2RhYmM=|d14a369dd6d35825cc5204bceb78332214e2ff040f83a19ac17b6c6e3d7ff593"; SESSIONID=CIkB30B7idjktTgTziqbYjT7mryHE2SxIdPIdX468DN; JOID=Vl4cCkjbHvzreT7nStYLr-sfEPtUmliXo00Mqn21eJmAB22CEd9M-Id8M-hArdlFhcUg4BNGfxzPfB3JHh39mhI=; osd=UF4XBkPdHvfncjjnQdoAqesUHPBSmlObqEsMoXG-fpmLC2aEEdRA84F8OORLq9lOic4m4BhKdBrPdxHCGB32lhk=; YD00517437729195:WM_NI=FBm6A9aJ4nGgMpRBOElWt++MUBzsr3YnAjZNRMcid9eRXmQTcyZzAM8x6QWlRaaG7O3UTSJc3cbMP7J2CnVeSIJ6GJyc5HhDPFYG05ZL9m5KuqRxUzeiStOpNfi9u3g+anM=; YD00517437729195:WM_NIKE=9ca17ae2e6ffcda170e2e6ee99f252a59fffccf9398db08fa6c15e968f8faff467acb0a0ccf24df1b09889e62af0fea7c3b92a869ea3d7c433a396fc8fc66d8fb09ea3ce4fb089bcccaa47979eb694f1399bb28f8bd9469293ad8ae733a78e988ed460a1a7969bc447fca8fab2d66892bfafa8b645f18b9cb4c969f1bdaeaed33d8aee8d8fd56f919786d7c642bbbdfba9eb50949a97a8b33f9a94f78eb841b7af9e99d872a7b79d95ce7c8ebaa7daf839a5f5af8fd037e2a3; gdxidpyhxdE=6w2Yjl+0u5Sp1XkgmWIiKdwoRCyVT\1gEP2AkM3xzxf4YI6M1p\1xT73zdrk428xarva1VuLbtwgPyuHs4poCDoT7etpzVLSDNfuU9Bb5r06SGEV5KIlEvR93SDHG1h6H\BLsMY\\6gtjUSKHwExmP6dew\EDbt9Qujih5f1mhvoBOTM:1628665452442; captcha_ticket_v2="2|1:0|10:1628664557|17:captcha_ticket_v2|704:eyJ2YWxpZGF0ZSI6IkNOMzFfNlVaMDV5dzFBeTd5dWI4emY3X3lOZXFodEFVZGRtVGpCZXFZQjJVLkdNOUwuZmZBaEFzUDdlanFxeFhIOGVUWS5IdFJtQnNpOW9mSnpJTV9tbXVhdG1TN1VzcG41cEdGNDhtT24tUXdLY295bVJPLUVMeEQ5OGdtQlNReHJnLTUycGhqNllVZGljYWNNQ290Z1NoR1ctMFVRUFlsYjhHWXhaWVFqcFNXX0ticWU4RC1MVkUyRTJsLU1VSlFzeFA2N0puaXNaekxqWUlDQkpabWlwbW5XekRZY3lUVlFMUDYyQmhHc1R1dDJER1ouWlJBS2I1UW5jSW41MFZhWDVnbzhQNWVocmlKdXN2MXFhdjdvWlBnUW5SZkFFaGRoUXVtZU9qUjdkblJ5bVpvNDVRdkg1X2gxZzRjWllZWk1KSTBJZUNqV1A3c2pwMnFldC1pcGxIRHlFd3lZLXBWNVM2d25KVjdSX2xtcXFPODRMUUl2R21oOUZVckZLb3hBMkdDWWJ1Rk5Nb19BRGtHajdXdmN0S3NtdzhDR0F0NWVjemJhOW5xaE1vbXJZYnQ5LllaWTd5VS5HTmh0eV9WMkVKaWxBSW5XNXVKTHRMZjJ2Q3BzYmthejBLcDl1Mkxhby5FdEdkRmJHa0FaVEJWUXgyNzdYdy1IejJWWnFtMyJ9|70ac9e67527061bbeb02f7a677a4a2647bedd430a8db5ba6ce21495fa21d506d"; captcha_session_v2="2|1:0|10:1628664575|18:captcha_session_v2|88:VC9xa3F0U0k2ZlkzLzhWb0h0cEFEMmQrVW84TjRCZmhyN0ZGeDBGL3p5WlNoaHhHUVBkMmRrNTUzclppMXlQWA==|3cf812d7021f509a7fa956d8869546a260c9ecf7ab7e9c6c527016ec0d7ce0e1"; l_n_c=1; r_cap_id="YzcxMDBiNjk0MWM4NDEyYWFhZTM4OTBmMDJmYjM2Yzk=|1628664577|4aa40487c35ff9c3ab748af15ad7aaf1850187bd"; cap_id="MjVmYzE2OWI2Mzc5NDk0MTkyMWExMWJjM2YxZTdkMTU=|1628664577|704bedf588123bf1804e46759fe9b166fdd401c3"; l_cap_id="ZTU4MzIwMzU4YWZhNGY3ZTkyMGUxNjU4Zjc5ODc5MTc=|1628664577|b787eb5455f272dfabba83811cbdeb84de4870db"; n_c=1; z_c0=Mi4xQW1BZUFBQUFBQUFBUUo0YmFpTVpFeGNBQUFCaEFsVk5FY0VBWWdDNGdQdmNwYU5MVkRKbG56QzZiSXZUYUlnVGtB|1628664593|b47b3cc529e27a7a6877a1bbbe2618194af6ff25; KLBRSID=76ae5fb4fba0f519d97e594f1cef9fab|1628664594|1628662213`
	answerInclude = "data[*].is_normal,admin_closed_comment,reward_info,is_collapsed,annotation_action,annotation_detail,collapse_reason,is_sticky,collapsed_by,suggest_edit,comment_count,can_comment,content,editable_content,attachment,voteup_count,reshipment_settings,comment_permission,created_time,updated_time,review_info,relevant_info,question,excerpt,is_labeled,paid_info,paid_info_content,relationship.is_authorized,is_author,voting,is_thanked,is_nothelp,is_recognized;data[*].mark_infos[*].url;data[*].author.follower_count,vip_info,badge[*].topics;data[*].settings.table_of_content.enabled"
	userAgent     = `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36`
)

func authReq(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	for _, c := range cookCookies() {
		req.AddCookie(c)
	}
}

func cookCookies() []*http.Cookie {
	var cookies []*http.Cookie
	for _, c := range strings.Split(cookie, ";") {
		kv := strings.Split(c, "=")
		name, value := kv[0], kv[1]
		value = regexp.MustCompile(`[\\"]`).ReplaceAllString(value, "")
		cookies = append(cookies, &http.Cookie{
			Name:  name,
			Value: value,
		})
	}
	return cookies
}

func collector() *colly.Collector {
	col := colly.NewCollector()
	col.UserAgent = userAgent
	col.SetCookies(hot, cookCookies())
	return col
}

func getHotQuestions() ([]Question, error) {
	var items []Question
	// HTML parse
	col := collector()
	col.OnHTML(".HotList-list .HotItem", func(element *colly.HTMLElement) {
		var text, link string
		element.ForEachWithBreak("a", func(i int, element *colly.HTMLElement) bool {
			link = element.Attr("href")
			return false
		})
		element.ForEachWithBreak("h2", func(i int, element *colly.HTMLElement) bool {
			text = element.Text
			return false
		})
		item := Question{
			Title: text,
		}
		var err error
		item.URL, err = url.Parse(link)
		if err != nil {
			panic(err)
		}
		items = append(items, item)
	})
	err := col.Visit(hot)
	if err != nil {
		return nil, err
	}
	var loadCount int
	for i := range items {
		if !items[i].CanMatchQID() {
			log.Printf("cannot match id: %s", items[i].URL.Path)
			continue
		}
		err = items[i].load()
		// filter away unloaded question
		if err != nil {
			log.Printf("error loading question %d %s: %v", i, items[i].URL, err)
		} else {
			loadCount++
		}
	}
	log.Printf("successfully load %d questions", loadCount)
	return items, err
}

func Hot() ([]Question, error) {
	data, err := getHotQuestions()
	if err != nil {
		return nil, err
	}
	err = loadAnswer(data)
	return data, err
}

func loadAnswer(items []Question) error {
	// 并发XHR请求获得每个问题的前十个答案
	var wg errgroup.Group
	for i := range items {
		i := i
		if items[i].CanMatchQID() {
			wg.Go(func() error {
				return items[i].loadAnswer(10)
			})
		}
	}
	return wg.Wait()
}

// populate question fields
func (q *Question) load() error {
	api, _ := url.Parse(fmt.Sprintf("https://www.zhihu.com/api/v4/questions/%s", q.QID()))
	req, err := http.NewRequest("GET", api.String(), nil)
	if err != nil {
		return err
	}
	authReq(req)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	var data struct {
		Error *struct {
			Code    int    `json:"code"`
			Name    string `json:"name"`
			Message string `json:"message"`
		} `json:"error"`
		*Question
	}
	data.Question = q
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return err
	}
	if data.Error != nil {
		return fmt.Errorf(data.Error.Message)
	}
	return nil
}

func (q *Question) loadAnswer(limit int) error {
	// 前十条回答
	ansAPI, _ := url.Parse(fmt.Sprintf("https://www.zhihu.com/api/v4/questions/%s/answers", q.QID()))
	values := url.Values{}
	values.Add("limit", strconv.Itoa(limit))
	values.Add("offset", "0")
	values.Add("include", answerInclude)
	ansAPI.RawQuery = values.Encode()
	req, err := http.NewRequest("GET", ansAPI.String(), nil)
	if err != nil {
		return err
	}
	authReq(req)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	var data struct {
		Data   []Answer `json:"data"`
		Paging struct {
			IsEnd    bool   `json:"is_end"`
			IsStart  bool   `json:"is_start"`
			Next     string `json:"next"`
			Previous string `json:"previous"`
			Total    int    `json:"total"`
		}
		ReadCount int `json:"read_count"`
	}
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return err
	}
	q.Answer = data.Data
	return nil
}

type Author struct {
	Headline string `json:"headline"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	UserType string `json:"user_type"`
}

type Answer struct {
	AnswerType   string   `json:"answer_type"`
	Author       Author   `json:"author"`
	ID           int      `json:"id"`
	Content      string   `json:"content"`
	CommentCount int      `json:"comment_count"`
	Question     Question `json:"question"`
	CreatedTime  int      `json:"created_time"`
	Excerpt      string   `json:"excerpt"`
	API          string   `json:"url"`
	Type         string   `json:"type"`
	UpdatedTime  int64    `json:"updated_time"`
	VoteUpCount  int      `json:"voteup_count"` // 赞同
}

func (q *Question) PrintAnswer(i int) string {
	a := q.Answer[i]
	var sb strings.Builder
	// 问题标题
	sb.WriteString(fmt.Sprintf("\u300E%s\u300F\n\n", q.Title)) // 使用繁体中文引号
	// 问题详情
	sb.WriteString(fmt.Sprintf(
		`%s %s
%s%s
【%s%d %s%d %s%d】
%supdated at %s
%s`,
		emoji.WritingHand, a.Author.Name,
		emoji.IdButton, fmt.Sprintf(`<a href="https://www.zhihu.com/question/%s/answer/%d">%d</a>`, q.QID(), a.ID, a.ID), // link to go to answer
		emoji.KeycapHash, i+1, emoji.UpArrow, a.VoteUpCount, emoji.SpeechBalloon, a.CommentCount,
		emoji.AlarmClock, time.Unix(a.UpdatedTime, 0).Format("2006-01-02 15:04"),
		a.Prefix(3000), // 保留3000个UTF字符，TL允许的最大消息是4096个字符
	))
	return sb.String()
}

const TLMsgMaxSize = 4096 // UTF characters

func (a Answer) Prefix(limit int) string {
	content := a.Content
	content = regexp.MustCompile("<p>").ReplaceAllString(content, "")
	content = regexp.MustCompile("</p>").ReplaceAllString(content, "\n")

	utf := utf8string.NewString(common.PolicySanitizer("a").Sanitize(content))
	short := utf.Slice(0, util.Min(limit, utf.RuneCount()))
	if limit < utf.RuneCount() {
		short += "..."
	}
	return short
}

type Question struct {
	ID           int      `json:"id"`
	QuestionType string   `json:"question_type"`
	Title        string   `json:"title"`
	Created      int      `json:"created"`
	UpdatedTime  int      `json:"updated_time"`
	API          string   `json:"url"`
	URL          *url.URL `json:"-"`
	Answer       []Answer `json:"-"`
}

type topic string

const (
	QuestionTopic   topic = "question"
	SpecialTopic    topic = "special"
	AssessmentTopic topic = "xen/market/assessment"
	RoundTableTopic topic = "roundtable"
	UnknownTopic    topic = ""
)

// 匹配URL：普通问题，专题，测试和圆桌
var qidRegexp = regexp.MustCompile(`/(question|special|xen/market/assessment|roundtable)/(\d+)`)

func (q Question) QID() string {
	match := qidRegexp.FindStringSubmatch(q.URL.Path)
	if len(match) >= 2 {
		return match[2]
	}
	return ""
}

func (q Question) Topic() topic {
	if !q.CanMatchQID() {
		return UnknownTopic
	}
	match := qidRegexp.FindStringSubmatch(q.URL.Path)
	return topic(match[1])
}

// CanMatchQID determines whether we can handle this question
func (q Question) CanMatchQID() bool {
	return qidRegexp.MatchString(q.URL.Path)
}
