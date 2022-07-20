package weather

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
)

const (
	baiduMapOpenAPISecret       = "iVZh5McuRibpAgTmDXyy1P3ixcmTnUeg"
	baiduMapGeocodingURL        = "http://api.map.baidu.com/geocoding/v3/"
	baiduMapReverseGeocodingURL = "https://api.map.baidu.com/reverse_geocoding/v3/"
)

type Location interface {
	Latitude() float64
	Longitude() float64
}

type Coordinate struct {
	Lng float64 `json:"lng"` // 经度
	Lat float64 `json:"lat"` // 纬度
}

func (c Coordinate) Latitude() float64 {
	return c.Lat
}

func (c *Coordinate) Longitude() float64 {
	return c.Lng
}

const (
	degreeSymbol   = "\u00B0"
	minuteInRadian = 1 / 60.0
	secondInRadian = minuteInRadian * minuteInRadian
)

func radianToDegree(r float64) string {
	return fmt.Sprintf("%d%s%d'%d''",
		int(r),
		degreeSymbol,
		int(math.Mod(r, 1)/minuteInRadian),
		int(math.Mod(math.Mod(r, 1), minuteInRadian)/secondInRadian),
	)
}

func (c *Coordinate) FormatRadian() string {
	return fmt.Sprintf(
		"Lng: %.2f, Lat: %.2f",
		c.Lng,
		c.Lat,
	)
}

func getDirection(loc Location) (string, string) {
	var lngDirection, latDirection string
	if loc.Longitude() > 0 {
		lngDirection = "E" // 东经
	} else {
		lngDirection = "W" // 西经
	}
	if loc.Latitude() > 0 {
		latDirection = "N" // 北纬
	} else {
		latDirection = "S" // 南维
	}
	return lngDirection, latDirection
}

func formatDegree(loc Location) string {
	lngDirection, latDirection := getDirection(loc)
	return fmt.Sprintf(
		"%s%s, %s%s",
		radianToDegree(math.Abs(loc.Longitude())),
		lngDirection,
		radianToDegree(math.Abs(loc.Latitude())),
		latDirection,
	)
}

type GeocodeResponse struct {
	Status int     `json:"status"` // 返回结果状态值， 成功返回0
	Result Geocode `json:"result"`
}

type ReverseGeocodeResponse struct {
	Status int            `json:"status"` // 返回结果状态值， 成功返回0
	Result ReverseGeocode `json:"result"`
}

type ReverseGeocode struct {
	FormattedAddress   string `json:"formatted_address"`   // 结构化地址信息
	Business           string `json:"business"`            // 坐标所在商圈信息，如 "人民大学,中关村,苏州街"。最多返回3个
	SematicDescription string `json:"sematic_description"` // 当前位置结合POI的语义化结果描述。需设置extensions_poi=1才能返回
	AddressComponent   struct {
		Country         string `json:"country"`
		CountryCode     int    `json:"country_code"`
		CountryCodeIso  string `json:"country_code_iso"`
		CountryCodeIso2 string `json:"country_code_iso2"`
		Province        string `json:"province"`
		City            string `json:"city"`
		CityLevel       int    `json:"city_level"`
		District        string `json:"district"`
		Town            string `json:"town"`
		TownCode        string `json:"town_code"`
		Adcode          string `json:"adcode"`
		Street          string `json:"street"`
		StreetNumber    string `json:"street_number"`
		Direction       string `json:"direction"`
		Distance        string `json:"distance"`
	} `json:"addressComponent"`
}

type Geocode struct {
	Location      Coordinate `json:"location"`
	Confidence    int        `json:"confidence"`    // 描述打点绝对精度（即坐标点的误差范围）
	Comprehension int        `json:"comprehension"` // 描述地址理解程度。分值范围0-100，分值越大
	Level         string     `json:"level"`         // 能精确理解的地址类型
}

func newValues() url.Values {
	v := url.Values{}
	v.Set("ak", baiduMapOpenAPISecret)
	v.Set("output", "json") // 默认值为xml
	return v
}

// BaiduGeocodeDomestic 百度地图提供的正地理编码服务
func BaiduGeocodeDomestic(addr string) (Location, error) {
	var geo GeocodeResponse
	values := newValues()
	values.Set("address", addr)
	fullURL := baiduMapGeocodingURL + "?" + values.Encode()
	res, err := client.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&geo)
	if err != nil {
		return nil, err
	}
	if geo.Status != 0 {
		return nil, fmt.Errorf("API returns error code %d", geo.Status)
	}
	return &geo.Result.Location, nil
}

// BaiduReverseGeocodeDomestic 百度地图提供的逆地理编码服务
func BaiduReverseGeocodeDomestic(loc Location) (*ReverseGeocode, error) {
	var reverse ReverseGeocodeResponse
	values := newValues()
	values.Set("location", fmt.Sprintf("%f,%f", loc.Latitude(), loc.Longitude()))
	values.Set("extensions_poi", "1") // 显示pois数据和pois语义化数据
	fullURL := baiduMapReverseGeocodingURL + "?" + values.Encode()
	res, err := client.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(&reverse)
	if err != nil {
		return nil, err
	}
	if reverse.Status != 0 {
		return nil, fmt.Errorf("API returns error code %d", reverse.Status)
	}
	return &reverse.Result, nil
}
