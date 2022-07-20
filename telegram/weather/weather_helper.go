package weather

import (
	"net/url"
	"regexp"
	"strconv"
	"time"
)

// 匹配任何汉字
func isChinese(s string) bool {
	m, _ := regexp.MatchString(`\p{Han}+`, s)
	return m
}

// 配置:
// 经纬度查询
// 公制单位
// 中文
// 密钥
func configureValues(values *url.Values, loc Location) {
	values.Set("lat", strconv.FormatFloat(loc.Latitude(), 'f', 8, 64))
	values.Set("lon", strconv.FormatFloat(loc.Longitude(), 'f', 8, 64))
	values.Set("units", "metric") // 公制单位: 摄氏度, 米
	values.Set("lang", "zh_cn")   // Get weather description in Simplified Chinese
	values.Set("appid", openWeatherMapAPISecret)
}

// UnixUTCToLocalTime
// converts a unix UTC timestamp in seconds, to local time
// offset is the offset of local timezone from UTC in seconds
func UnixUTCToLocalTime(d, offset int64) time.Time {
	return time.Unix(d, 0).In(time.UTC).Add(time.Second * time.Duration(offset))
}
