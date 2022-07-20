package weather

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetGeocodeOversea(t *testing.T) {
	geo, err := GetGeocodeOversea("Melbourne VIC")
	assert.Nil(t, err)
	t.Log(geo)
}
