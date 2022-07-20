package weather

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yangrq1018/jerry-bot/util"
)

func TestGetGeocode(t *testing.T) {
	geo, err := BaiduGeocodeDomestic(beijingAddr)
	assert.Nil(t, err)

	geo, err = BaiduGeocodeDomestic(shanghaiAddr)
	assert.Nil(t, err)
	util.PrintIndentedJSON(geo)
}

func TestBaiduReverseGeocodeDomestic(t *testing.T) {
	r, err := BaiduReverseGeocodeDomestic(&Coordinate{
		Lat: 39.9087,
		Lng: 116.3974,
	})
	assert.Nil(t, err)
	t.Log(r.FormattedAddress)
}
