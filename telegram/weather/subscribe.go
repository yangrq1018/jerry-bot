package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Lofanmi/chinese-calendar-golang/calendar"
	"github.com/enescakir/emoji"
	"github.com/go-co-op/gocron"
	flag "github.com/jayco/go-emoji-flag"
	"github.com/yangrq1018/jerry-bot/gcloud"
	"github.com/yangrq1018/jerry-bot/gwy"
	"github.com/yangrq1018/jerry-bot/telegram"
	"github.com/yangrq1018/jerry-bot/telegram/common"
	"github.com/yangrq1018/jerry-bot/util"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
	"golang.org/x/exp/rand"
)

const (
	openWeatherMapAPISecret                            = "65ae0292453eacc27a0ae38e0b2f7c3b"
	openWeatherMapAPIURL                               = "https://api.openweathermap.org/data/2.5"
	endpointOneCall          OpenWeatherMapAPIEndpoint = "/onecall"
	endpointWeather          OpenWeatherMapAPIEndpoint = "/weather" // endpointWeather
	openWeatherMapAPIIconURL                           = "http://openweathermap.org/img/wn/%s@2x.png"

	gcloudObjectKey = "subscribers"
)

func newNoProxyHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
}

var client = newNoProxyHTTPClient()

type OpenWeatherMapAPIEndpoint string

func OWEndpoint(endpoint OpenWeatherMapAPIEndpoint) string {
	return openWeatherMapAPIURL + string(endpoint)
}

// Current See also https://openweathermap.org/current
type Current struct {
	Coordinates *struct {
		json.RawMessage
		Lon float64 `json:"lon"` // City geo location, longitude
		Lat float64 `json:"lat"` // City geo location, latitude
	} `json:"coordinates,omitempty"`
	Weather []Condition
	Base    string `json:"base"`
	Main    struct {
		Temp      Temperature `json:"temp"`
		FeelsLike Temperature `json:"feels_like"`
		TempMin   Temperature `json:"temp_min"`
		TempMax   Temperature `json:"temp_max"`
		Pressure  int         `json:"pressure"`
		Humidity  int         `json:"humidity"`
	}
	Visibility int `json:"visibility"`
	Wind       struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
		Gust  float64 `json:"gust"`
	}
	Clouds struct {
		All int `json:"all"` // Cloudiness, %
	}
	Sys struct {
		Country string `json:"country"` // Country code (GB, JP etc.)
		Sunrise int64  `json:"sunrise"` // Sunrise time, unix, UTC
		Sunset  int64  `json:"sunset"`  // Sunset time, unix, UTC
	}
	Timezone int64  `json:"timezone"` // Shift in seconds from UTC
	Name     string `json:"name"`     // City name
}

type Temperature float64

// String format the temperature as celsius
func (t Temperature) String() string {
	return fmt.Sprintf("%.0f\u2103", t)
}

// OneCall
// One call API
// See https://openweathermap.org/api/one-call-api
type OneCall struct {
	Timezone       string  `json:"timezone"`        // Timezone name for the requested location
	TimezoneOffset int64   `json:"timezone_offset"` // Shift in seconds from UTC
	Lat            float64 `json:"lat"`
	Lon            float64 `json:"lon"`
	Current        struct {
		Dt         int     `json:"dt"`
		Sunrise    int     `json:"sunrise"`
		Sunset     int     `json:"sunset"`
		Temp       float64 `json:"temp"`
		FeelsLike  float64 `json:"feels_like"`
		Pressure   float64 `json:"pressure"`
		Humidity   int     `json:"humidity"`
		UVI        float64 `json:"uvi"`
		Clouds     int     `json:"clouds"`
		Visibility int     `json:"visibility"`
		WindSpeed  float64 `json:"wind_speed"`
		WindDeg    int     `json:"wind_deg"`
	} `json:"current"`
	Daily  []DailyForecast `json:"daily"`
	Alerts []struct {
		SenderName  string `json:"sender_name"`
		Event       string `json:"event"`
		Start       int64  `json:"start"`
		End         int64  `json:"end"`
		Description string `json:"description"`
	} // National weather alerts data from major national weather warning systems
}

func (w OneCall) Latitude() float64 {
	return w.Lat
}

func (w OneCall) Longitude() float64 {
	return w.Lon
}

func (w *OneCall) report(i int) string {
	var mainWeatherDescription string
	daily := w.Daily[i]
	if len(daily.Weather) > 0 {
		mainWeatherDescription = daily.Weather[0].Description
	}
	return fmt.Sprintf(
		`%s %s
%s, %s到%s
温度: %s
体感: %s
`,
		emoji.Calendar.String(), UnixUTCToLocalTime(daily.Dt, w.TimezoneOffset).Format("2006/01/02"),
		mainWeatherDescription, daily.Temp.Min, daily.Temp.Max,
		daily.Temp.TempInDay, // String() must have a value receiver as daily.Temp is value
		daily.FeelsLike,
	)
}

func (w *OneCall) ReportDay(t time.Time) string {
	for i := range w.Daily {
		if util.DateEqual(w.Daily[i].Time(), t) { // 中午12点
			return w.report(i)
		}
	}
	return ""
}

func (w OneCall) ReportAlerts() string {
	if len(w.Alerts) == 0 {
		return "None"
	}
	var sb strings.Builder
	for _, alert := range w.Alerts {
		sb.WriteString(fmt.Sprintf(
			"%s%s提示, %s%s至%s, 有[%s]\n%s\n",
			emoji.SatelliteAntenna,
			alert.SenderName,
			emoji.AlarmClock,
			UnixUTCToLocalTime(alert.Start, w.TimezoneOffset).Format("02日15时"),
			UnixUTCToLocalTime(alert.End, w.TimezoneOffset).Format("02日15时"),
			alert.Event,
			alert.Description))
	}
	return sb.String()
}

func (w *OneCall) ReportEveryDay(addr string, countryCode string) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf(
		`%q
%s %s %s
`, addr, emoji.RoundPushpin, formatDegree(w.Location()), flag.GetFlag(countryCode))) // country not available here, query Baidu Map maybe?
	for i := range w.Daily {
		sb.WriteString(w.report(i))
		if i != len(w.Daily)-1 {
			sb.WriteString("-------------------------------\n")
		}
	}
	return sb.String()
}

func (t TempInDayWithMinMax) String() string {
	return fmt.Sprintf("早%s 中%s 晚%s 夜%s 高%s 低%s", t.Morn, t.Day, t.Eve, t.Night, t.Max, t.Min)
}

func (t TempInDay) String() string {
	return fmt.Sprintf("早%s 中%s 晚%s", t.Morn, t.Day, t.Eve)
}

type TempInDay struct {
	Morn  Temperature `json:"morn"`  // Morning temperature
	Day   Temperature `json:"day"`   // Day temperature
	Eve   Temperature `json:"eve"`   // Evening temperature
	Night Temperature `json:"night"` // Night temperature
}

type TempInDayWithMinMax struct {
	TempInDay
	Min Temperature `json:"min"` // Min daily temperature
	Max Temperature `json:"max"` // Max daily temperature
}

type DailyForecast struct {
	Dt       int64 `json:"dt"` // Time of the forecasted data, Unix, UTC
	Sunrise  int   `json:"sunrise"`
	Sunset   int   `json:"sunset"`
	Moonrise int   `json:"moonrise"`
	Moonset  int   `json:"moonset"`
	// Moon phase. 0 and 1 are 'new moon', 0.25 is 'first quarter moon', 0.5 is 'full moon' and 0.75 is
	// 'last quarter moon'. The periods in between are called 'waxing crescent', 'waxing gibous', 'waning gibous',
	// and 'waning crescent', respectively.
	MoonPhase float64 `json:"moon_phase"`
	Temp      TempInDayWithMinMax
	FeelsLike TempInDay   `json:"feels_like"`
	Pressure  float64     `json:"pressure"`  // Atmospheric pressure on the sea level, hPa
	Humidity  int         `json:"humidity"`  // Humidity, %
	DewPoint  float64     `json:"dew_point"` // Atmospheric temperature
	WindSpeed float64     `json:"wind_speed"`
	WindDeg   int         `json:"wind_deg"` // Wind direction, degrees (meteorological)
	Clouds    int         `json:"clouds"`   // Cloudiness, %
	UVI       float64     `json:"uvi"`      // The maximum value of UV index for the day
	Pop       float64     `json:"pop"`      // Probability of precipitation
	Weather   []Condition `json:"weather"`  // List of weather conditions
}

func (d DailyForecast) Time() time.Time {
	return time.Unix(d.Dt, 0)
}

type Condition struct {
	ID          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func (w *Current) Location() Location {
	if w.Coordinates == nil {
		return nil
	}
	// open weather map returns empty location (0, 0)
	// we won't go to location (0, 0) on earth, obviously...
	if w.Coordinates.Lon == 0 && w.Coordinates.Lat == 0 {
		return nil
	}
	return &Coordinate{
		Lat: w.Coordinates.Lat,
		Lng: w.Coordinates.Lon,
	}
}

func (w *OneCall) Location() *Coordinate {
	return &Coordinate{
		Lat: w.Lat,
		Lng: w.Lon,
	}
}

func (w *Current) Report(addr string, loc Location) string {
	var mainWeatherDescription string
	if len(w.Weather) > 0 {
		mainWeatherDescription = w.Weather[0].Description
	}

	// time.Unix returns local time object
	sunRise := UnixUTCToLocalTime(w.Sys.Sunrise, w.Timezone)
	sunSet := UnixUTCToLocalTime(w.Sys.Sunset, w.Timezone)
	if loc == nil {
		loc = w.Location()
	}
	var locString string
	if loc != nil {
		locString = formatDegree(loc)
	}
	return fmt.Sprintf(
		`%s查询地址：%s %s%s %s
%s描述: %s
%s温度: %s (体感 %s)
%s湿度：%d%%
%s能见度: %dm
%s日出: %s
%s日落: %s
`,
		emoji.RoundPushpin, addr, emoji.GlobeWithMeridians, locString, flag.GetFlag(w.Sys.Country),
		emoji.SpeechBalloon, mainWeatherDescription,
		emoji.Thermometer, w.Main.Temp, w.Main.FeelsLike,
		emoji.WaterWave, w.Main.Humidity,
		emoji.Foggy.String(), w.Visibility,
		emoji.Sunrise.String(), sunRise.Format("15:04"),
		emoji.Sunset.String(), sunSet.Format("15:04"),
	)
}

// IconURL returns the URL where you can download icon image
func (w Condition) IconURL() string {
	return fmt.Sprintf(openWeatherMapAPIIconURL, w.Icon)
}

type Subscription struct {
	Location Location
	tgbotapi.User
	PreferredName string
	zone          *time.Location
}

type PersistentSubscription struct {
	Location      Coordinate    `json:"location"`
	User          tgbotapi.User `json:"user"`
	PreferredName string        `json:"preferred_name"`
	Zone          string        `json:"zone"`
}

func encodeSubscription(s Subscription) PersistentSubscription {
	return PersistentSubscription{
		Location: Coordinate{
			Lat: s.Location.Latitude(),
			Lng: s.Location.Longitude(),
		},
		User:          s.User,
		PreferredName: s.PreferredName,
		Zone:          s.zone.String(),
	}
}

func decodeSubscription(s PersistentSubscription) Subscription {
	zone, err := time.LoadLocation(s.Zone)
	if err != nil {
		logger.Errorf("cannot load location of subscription %s: %v", s.User.UserName, err)
	}
	return Subscription{
		Location: &Coordinate{
			Lat: s.Location.Lat,
			Lng: s.Location.Lng,
		},
		User:          s.User,
		PreferredName: s.PreferredName,
		zone:          zone,
	}
}

func (s Subscription) titleUser() string {
	if s.PreferredName != "" {
		return s.PreferredName
	}
	return s.FirstName
}

// GetJobFunc keep pointer receiver! Change in s will be reflected in the callback then
// which is convenient to modify PreferredName or so.
func (s *Subscription) GetJobFunc(b *telegram.Bot) func() {
	return func() {
		sendTime := time.Now()
		err := s.SendMsg(b)
		if err != nil {
			logger.Error(err)
		} else {
			logger.Infof("send msg to %s, timezone %s, trigger time (local) %s, msg time (local) %s",
				s.titleUser(), s.zone, sendTime.In(s.zone).Format(time.RFC3339), time.Now().In(s.zone).Format(time.RFC3339))
		}
	}
}

func buildHoliday() string {
	var holiday string
	n, h, errT := gwy.NextHoliday(time.Now())
	if errT == nil && h != nil {
		holiday = fmt.Sprintf(
			"%s%s还有%.0f天",
			emoji.MoonCake, n, h.Sub(time.Now()).Hours()/24)
	}
	return holiday
}

type TodayInHistory struct {
	Date string `json:"date"`
	URL  string `json:"url"` // wikipedia
	Data struct {
		Events []Event `json:"events"`
	} `json:"data"`
}

type Event struct {
	Year       string `json:"year"`
	Text       string `json:"text"`
	HTML       string `json:"html"` // some html that describes the event, with links to wikipedia
	NoYearHTML string `json:"no_year_html"`
	Links      []struct {
		Title string `json:"title"`
		Link  string `json:"link"`
	} `json:"links"`
}

func GetTodayInHistory() (*TodayInHistory, error) {
	res, err := http.Get("http://history.muffinlabs.com/date")
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("history.muffinlabs.com returns %v", http.StatusServiceUnavailable)
	}
	defer func() {
		_ = res.Body.Close()
	}()
	var m TodayInHistory
	err = json.NewDecoder(res.Body).Decode(&m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

var random = rand.New(rand.NewSource(uint64(time.Now().UnixNano())))

func (s Subscription) makeMsg() string {
	logger.Infof("remind %q %d %q", s.UserName, s.ID, s.FirstName)
	var text string

	loc := s.Location
	weatherNow, err1 := GetCurrentWeather(loc)
	weatherToday, err2 := GetOneCallWeather(loc)
	tih, err3 := GetTodayInHistory()

	var eventDescription string
	if err3 == nil {
		event := tih.Data.Events[random.Intn(len(tih.Data.Events))]
		eventDescription = common.PolicySanitizer("a").Sanitize(event.HTML)
	} else {
		eventDescription = "error: " + err3.Error()
	}

	if err1 != nil {
		logger.Error(err1)
		return "error: " + err1.Error()
	}
	if err2 != nil {
		logger.Error(err2)
		return "error: " + err2.Error()
	}
	var addr string
	// try chinese geocode first
	if addrPtr, err := BaiduReverseGeocodeDomestic(s.Location); err == nil {
		addr = addrPtr.FormattedAddress
		if addr != "" {
			goto AddrOK
		}
	}

	// try global geocode then
	if addrPtr, err := mapboxCoder.ReverseGeocode(s.Location.Latitude(), s.Location.Longitude()); err == nil {
		addr = addrPtr.Country + " " + addrPtr.City
	}

AddrOK:
	now := time.Now()
	today := calendar.ByTimestamp(now.Unix())

	text = fmt.Sprintf(`%s, hey there.
%s今天是:%s，星期%s，%s%s年%s%s
%s节假日:%s

[%s预警]
%s

[%s当前天气]
%s

[%s今天预报]:
%s

[%s历史上的今天]:
%s
`,
		s.titleUser(),
		emoji.Calendar, now.Format("2006年01月02日"), today.Solar.WeekAlias(), today.Ganzhi.YearGanzhiAlias(), today.Ganzhi.Animal().Alias(), today.Lunar.MonthAlias(), today.Lunar.DayAlias(),
		emoji.MoonCake, buildHoliday(),
		emoji.Warning, weatherToday.ReportAlerts(),
		emoji.Cloud, weatherNow.Report(addr, s.Location),
		emoji.Umbrella, weatherToday.ReportDay(now),
		emoji.Scroll, eventDescription,
	)
	return text
}

func (s Subscription) SendMsg(b *telegram.Bot) error {
	msg := tgbotapi.NewMessage(int64(s.ID), s.makeMsg())
	// the message contains Wikipedia links
	msg.ParseMode = "HTML"
	_, err := b.Bot().Send(msg)
	return err
}

/*
build a cron header that runs every day at hour:minute
	# ┌───────────── minute (0 - 59)
	# │ ┌───────────── hour (0 - 23)
	# │ │ ┌───────────── day of the month (1 - 31)
	# │ │ │ ┌───────────── month (1 - 12)
	# │ │ │ │ ┌───────────── day of the week (0 - 6) (Sunday to Saturday;
	# │ │ │ │ │                                   7 is also Sunday on some systems)
	# │ │ │ │ │
	# │ │ │ │ │
	# * * * * * <command to execute>
*/
func getCronString(hour int, minute int) string {
	return fmt.Sprintf("%02d %02d * * *", minute, hour)
}

func addRunEverydayJob(s *gocron.Scheduler, jobFunc interface{}, cronString string) error {
	_, err := s.
		Cron(cronString).
		Do(jobFunc)
	return err
}

// time.Location like "Europe/Brussels", represents the conceptual geological timezone,
// not a GMT offset like "GMT+2".
// For example, "Europe/Brussels" is GMT+2  (CEST) in summer, GMT+1 (CET) in winter.
// The system library handles this change in `time.Now().In(zone)`

type Reminder struct {
	bot      *telegram.Bot
	mu       sync.Mutex
	cronJobs map[int64]*gocron.Scheduler
	subs     map[int64]*Subscription
	jobs     map[int64]func()
}

func NewReminder() *Reminder {
	return &Reminder{
		cronJobs: make(map[int64]*gocron.Scheduler),
		subs:     make(map[int64]*Subscription),
		jobs:     make(map[int64]func()),
	}
}

func (l *Reminder) IsRegistered(user int64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.subs[user]
	return ok
}

func (l *Reminder) GetScheduler(user int64) *gocron.Scheduler {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.cronJobs[user]
}

func (l *Reminder) SetScheduler(user int64, s *gocron.Scheduler) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cronJobs[user] = s
}

func (l *Reminder) SetSub(user int64, target *Subscription) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.subs[user] = target
}

func (l *Reminder) Get(user int64) *Subscription {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.subs[user]
}

func (l *Reminder) doSubscribe(sub *Subscription) error {
	serverNow := time.Now().In(time.Local)
	logger.Infof("current server time is %s", serverNow.Format(time.RFC3339))
	s := gocron.NewScheduler(sub.zone)
	jf := sub.GetJobFunc(l.bot)
	err := addRunEverydayJob(s, jf, getCronString(8, 30))
	if err != nil {
		return err
	}
	l.SetScheduler(int64(sub.ID), s)
	l.SetSub(int64(sub.ID), sub)
	l.SetJobFunc(int64(sub.ID), jf)
	s.StartAsync()
	next := nextJobRunTime(s)
	logger.Infof("subscribe %d %s, next run time is %s (%s from now)",
		sub.ID, sub.UserName,
		next.Format(time.RFC3339),
		next.Sub(serverNow),
	)
	return nil
}

func (l *Reminder) Subscribers() []Subscription {
	var subs []Subscription
	for k := range l.subs {
		subs = append(subs, *(l.subs[k]))
	}
	return subs
}

func (l *Reminder) PersistentSubscribers() []PersistentSubscription {
	var subs []PersistentSubscription
	for _, sub := range l.Subscribers() {
		subs = append(subs, encodeSubscription(sub))
	}
	return subs
}

func (l *Reminder) loadSubscribersFromCloud() error {
	var subs []PersistentSubscription
	err := gcloud.LoadObject(gcloudObjectKey, &subs)
	if err != nil {
		return fmt.Errorf("cannot load subscribers from cloud: %v", err)
	}
	for i := range subs {
		sub := decodeSubscription(subs[i])
		err = l.doSubscribe(&sub)
		if err != nil {
			return err
		}
		logger.Infof("subscribed %s", sub.UserName)
	}
	return nil
}

// 注册位置消息
func (l *Reminder) subscribeFromLocationMessage(msg tgbotapi.Message) error {
	// not yet subscribed
	loc := TelegramLocationWrapper{msg.Location}
	sub := &Subscription{
		Location: loc,
		User:     *msg.From,
		zone:     timezoneFromCoordinates(loc),
	}
	return l.doSubscribe(sub)
}

func (l *Reminder) GetTargetByUsername(username string) *Subscription {
	for _, v := range l.subs {
		if v.UserName == username {
			return v
		}
	}
	return nil
}

func (l *Reminder) SetJobFunc(i int64, jf func()) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.jobs[i] = jf
}

func (l *Reminder) GetJobFunc(i int64) func() {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.jobs[i]
}

func nextJobRunTime(s *gocron.Scheduler) time.Time {
	_, t := s.NextRun()
	return t
}

func GetOneCallWeather(loc Location) (*OneCall, error) {
	path := OWEndpoint(endpointOneCall)
	var w OneCall
	values := url.Values{}
	configureValues(&values, loc)
	fullPath := path + "?" + values.Encode()
	err := SendPayload(fullPath, &w)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func GetCurrentWeather(loc Location) (*Current, error) {
	path := OWEndpoint(endpointWeather)
	var w Current
	values := url.Values{}
	configureValues(&values, loc)
	fullPath := path + "?" + values.Encode()
	err := SendPayload(fullPath, &w)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func GetGeocodeAuto(addr string) (Location, error) {
	var (
		loc Location
		err error
	)
	if isChinese(addr) {
		loc, err = BaiduGeocodeDomestic(addr)
	} else {
		loc, err = GetGeocodeOversea(addr)
	}
	return loc, err
}
