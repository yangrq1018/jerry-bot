package gwy

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/thoas/go-funk"
	"github.com/yangrq1018/jerry-bot/util"
)

const (
	dateFormat = "2006-01-02"
	timeFormat = "2006-01-02 15:04:05"
)

// 固定节假日名称
var legalHolidays = []string{
	"元旦",
	"春节",
	"清明节",
	"劳动节",
	"端午节",
	"中秋节",
	"国庆节",
}

// 时间开始结束范围
func getTimeRanges(start, end string) (*time.Time, *time.Time) {
	startTime, endTime, after := timeAfter(start, end)
	if startTime != nil && endTime != nil && !after {
		t := startTime
		startTime = endTime
		endTime = t
		return startTime, endTime
	}
	return startTime, endTime
}

// 两日期字符串比较
func timeAfter(start, end string) (*time.Time, *time.Time, bool) {
	timeFormatTpl := timeFormat
	if len(timeFormatTpl) != len(start) {
		timeFormatTpl = timeFormatTpl[0:len(start)]
	}
	startTime, err := time.Parse(timeFormatTpl, start)
	if err != nil {
		return nil, nil, false
	}
	endTime, err := time.Parse(timeFormatTpl, end)
	if err != nil {
		return nil, nil, false
	}
	if endTime.After(startTime) {
		return &startTime, &endTime, true
	}
	return &startTime, &endTime, false
}

// 获取两日期之间的全部日期
func getDates(start, end string) []string {
	res := make([]string, 0)
	startTime, endTime := getTimeRanges(start, end)
	if startTime == nil || endTime == nil {
		return res
	}
	// 输出日期格式固定
	timeFormatTpl := dateFormat
	endStr := endTime.Format(timeFormatTpl)
	res = append(res, startTime.Format(timeFormatTpl))
	for {
		current := startTime.AddDate(0, 0, 1)
		dateStr := startTime.Format(timeFormatTpl)
		startTime = &current
		res = append(res, dateStr)
		if dateStr == endStr {
			break
		}
	}
	return res
}

// Holidays 从国务院网站爬取节假日安排
func Holidays(year int) ([]string, []*time.Time, []string, error) {
	c := colly.NewCollector(
		// 允许重复访问
		colly.AllowURLRevisit(),
	)
	extensions.RandomUserAgent(c)
	extensions.Referer(c)

	q := fmt.Sprintf("国务院办公厅关于%d年部分节假日安排的通知", year)
	// url加密, 避免中文无法识别
	u := url.Values{}
	u.Set("t", "paper")
	u.Set("advance", "false")
	u.Set("q", q)
	searchUrl := "http://sousuo.gov.cn/s.htm?" + u.Encode()

	// 节假日
	names := make([]string, 0)
	// 休假时间
	holidays := make([]string, 0)
	// 调休时间
	workdays := make([]string, 0)
	// 上一年调休时间(GOV一般是11月份发布第二年, 可能涉及跨年调休, 该调休应该属于上一年)
	lastYearWorkdays := make([]string, 0)

	// 查看所有a标签
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		if e.Request.URL.String() == searchUrl {
			link := e.Attr("href")
			text := e.Text
			// 链接标题与搜索关键字一致
			if text == q {
				c.Visit(e.Request.AbsoluteURL(link))
			}
		}
	})
	// 查看文章内容
	c.OnHTML("td#UCAP-CONTENT", func(e *colly.HTMLElement) {
		s := strings.Split(e.Text, "\n")
		// 去掉前面无效的信息
		arr := make([]string, 0)
		for i, v := range s {
			if strings.Contains(v, "一、") {
				arr = s[i:]
				break
			}
		}
		for _, v1 := range arr {
			for _, v2 := range legalHolidays {
				if strings.Contains(v1, v2) {
					lineArr := strings.Split(v1, "。")
					// 截取休假时间
					// 1月1日至3日
					// 2月11日至17日
					item0 := lineArr[0]
					r1 := regexp.MustCompile(`([\d]{1,2})月([\d]{1,2})日至([\d]{1,2})月([\d]{1,2})日`)
					r2 := regexp.MustCompile(`([\d]{1,2})月([\d]{1,2})日至([\d]{1,2})日`)
					r3 := regexp.MustCompile(`[一二三四五六七八九十]+、(.*)：`)
					var holidayDates []string
					var name string
					if r1.MatchString(item0) {
						params := r1.FindStringSubmatch(item0)
						if len(params) == 5 {
							startMonth := util.MustAtoi(params[1])
							endMonth := util.MustAtoi(params[3])
							holidayDates = getDates(
								fmt.Sprintf("%d-%02d-%02d", year, startMonth, util.MustAtoi(params[2])),
								fmt.Sprintf("%d-%02d-%02d", year, endMonth, util.MustAtoi(params[4])),
							)
						}
					} else if r2.MatchString(item0) {
						params := r2.FindStringSubmatch(item0)
						if len(params) == 4 {
							month := util.MustAtoi(params[1])
							holidayDates = getDates(
								fmt.Sprintf("%d-%02d-%02d", year, month, util.MustAtoi(params[2])),
								fmt.Sprintf("%d-%02d-%02d", year, month, util.MustAtoi(params[3])),
							)
						}
					}
					if r3.MatchString(item0) {
						name = r3.FindStringSubmatch(item0)[1]
					}
					// 去重
					for _, date := range holidayDates {
						if !funk.ContainsString(holidays, date) {
							holidays = append(holidays, date)
							names = append(names, name)
						}
					}

					// 截取调休时间
					if len(lineArr) > 2 {
						item1 := lineArr[1]
						item1Arr := strings.Split(item1, "、")
						for _, v3 := range item1Arr {
							// r3解释: GOV一般是11月份发布第二年, 可能涉及跨年调休, 该调休应该属于上一年
							r3 := regexp.MustCompile(`([\d]{4})年([\d]{1,2})月([\d]{1,2})日`)
							r4 := regexp.MustCompile(`([\d]{1,2})月([\d]{1,2})日`)
							if r3.MatchString(v3) {
								params := r3.FindStringSubmatch(v3)
								if len(params) == 4 {
									if util.MustAtoi(params[1]) == year-1 {
										date := fmt.Sprintf("%d-%02d-%02d", year-1, util.MustAtoi(params[2]), util.MustAtoi(params[3]))
										if !funk.ContainsString(lastYearWorkdays, date) {
											lastYearWorkdays = append(lastYearWorkdays, date)
										}
									}
								}
							} else if r4.MatchString(v3) {
								params := r4.FindStringSubmatch(v3)
								if len(params) == 3 {
									date := fmt.Sprintf("%d-%02d-%02d", year, util.MustAtoi(params[1]), util.MustAtoi(params[2]))
									if !funk.ContainsString(workdays, date) {
										workdays = append(workdays, date)
									}
								}
							}
						}
					}
				}
			}
		}
	})

	var wg sync.WaitGroup
	var err error
	wg.Add(1)
	// 访问完成
	c.OnScraped(func(r *colly.Response) {
		if r.Request.URL.String() == searchUrl {
			wg.Done()
		}
	})
	c.OnError(func(r *colly.Response, e error) {
		if r.Request.URL.String() == searchUrl {
			err = e
			wg.Done()
		}
	})

	// 访问页面
	c.Visit(searchUrl)

	// 等待结束
	c.Wait()

	var holidaysT []*time.Time
	for i := range holidays {
		h, err := time.Parse("2006-01-02", holidays[i])
		if err == nil {
			holidaysT = append(holidaysT, &h)
		}
	}
	return names, holidaysT, workdays, err
}

func NextHoliday(t time.Time) (string, *time.Time, error) {
	allHoliday, allHolidayDate, _, err := Holidays(t.Year())
	if err != nil {
		return "", nil, err
	}

	for i := range allHoliday {
		if allHolidayDate[i].After(t) {
			return allHoliday[i], allHolidayDate[i], nil
		}
	}
	return "", nil, nil
}
