package weather

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yangrq1018/jerry-bot/telegram"
	"github.com/yangrq1018/jerry-bot/util"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

const (
	beijingAddr  = "北京市海淀区上地十街10号"
	shanghaiAddr = "上海市南洋泾路555号"
)

var beijingLocation = &Coordinate{
	Lng: 116.3084202915042,
	Lat: 40.05703033345938,
}

func Test_everydayCron(t *testing.T) {
	expected := "30 08 * * *"
	assert.Equal(t, getCronString(8, 30), expected)
}

func TestMakeMsg(t *testing.T) {
	t.Run("fictional user", func(t *testing.T) {
		text := Subscription{
			User: tgbotapi.User{
				FirstName: "Bob",
			},
			Location: beijingLocation,
		}.makeMsg()
		t.Log(text)
	})

	t.Run("oversea user", func(t *testing.T) {
		text := Subscription{
			User: tgbotapi.User{
				ID:        1612463904,
				UserName:  "jauzicing",
				FirstName: "Anonymous",
				LastName:  "Anonymous",
			},
			Location: &Coordinate{Lng: 4.708177, Lat: 50.880173},
		}.makeMsg()
		t.Log(text)
	})

	t.Run("domestic user", func(t *testing.T) {
		text := Subscription{
			User: tgbotapi.User{
				ID:        telegram.DeveloperChatIDint, // UserID和ChatID完全相同
				UserName:  "martinys",
				FirstName: "Anonymous",
				LastName:  "Anonymous",
			},
			Location:      &Coordinate{Lng: 121.683073, Lat: 31.273206},
			PreferredName: "Anonymous",
			zone:          timezoneFromCoordinates(&Coordinate{Lng: 121.683073, Lat: 31.273206}),
		}.makeMsg()
		t.Log(text)
	})
}

func TestTimezoneFromCoordinates(t *testing.T) {
	// somewhere in Belgium
	zone := timezoneFromCoordinates(&Coordinate{
		Lng: 4.708177,
		Lat: 50.880173,
	})
	assert.Equal(t, zone.String(), "Europe/Brussels")
}

func TestGetWeatherOneCall(t *testing.T) {
	w, err := GetOneCallWeather(beijingLocation)
	assert.Nil(t, err)
	util.PrintIndentedJSON(w)
}

func TestGetWeatherForecast(t *testing.T) {
	w, err := GetOneCallWeather(beijingLocation)
	assert.Nil(t, err)
	for i := range w.Daily {
		fmt.Printf("weather on: %s\n", UnixUTCToLocalTime(w.Daily[i].Dt, w.TimezoneOffset))
	}
}

func TestGetCurrentWeather(t *testing.T) {
	w, err := GetCurrentWeather(beijingLocation)
	assert.Nil(t, err)
	util.PrintIndentedJSON(w)
}

func TestReportWeather(t *testing.T) {
	var w *Current
	data := []byte(`
{
  "Coordinates": {
    "lon": 0,
    "lat": 0
  },
  "Weather": [
    {
      "id": 501,
      "main": "Rain",
      "description": "中雨",
      "icon": "10d"
    }
  ],
  "base": "stations",
  "Main": {
    "temp": 17.95,
    "feels_like": 17.97,
    "temp_min": 17.95,
    "temp_max": 17.95,
    "pressure": 1004,
    "humidity": 83
  },
  "visibility": 10000,
  "Wind": {
    "speed": 3.19,
    "deg": 324,
    "gust": 8.96
  },
  "Clouds": {
    "all": 100
  },
  "Sys": {
    "country": "CN",
    "sunrise": 1632088810,
    "sunset": 1632132992
  },
  "timezone": 28800,
  "name": "Haidian"
}
`)
	assert.Nil(t, json.Unmarshal(data, &w))
	fmt.Println(w.Report("北京", nil))
}

func TestTodayInHistory(t *testing.T) {
	tih, err := GetTodayInHistory()
	assert.NoError(t, err)
	for _, event := range tih.Data.Events {
		fmt.Println(event.Text)
	}
}

func TestSubscription_SendMsg(t *testing.T) {
	sub := Subscription{
		User: tgbotapi.User{
			ID:        telegram.DeveloperChatIDint,
			UserName:  "jauzicing",
			FirstName: "Anonymous",
			LastName:  "Anonymous",
		},
		Location: &Coordinate{Lng: 4.708177, Lat: 50.880173},
	}
	bot, err := telegram.NewMessageBot(telegram.JerryToken())
	require.NoError(t, err)
	require.NoError(t, sub.SendMsg(bot))
}
