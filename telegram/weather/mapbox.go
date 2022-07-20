package weather

import "github.com/codingsince1985/geo-golang/mapbox"

const (
	mapboxAPISecret = "pk.eyJ1IjoibWFydGlueWFuZ3JxIiwiYSI6ImNrcXU5ZnozajAyYzcydm1uOGxjN28wY2oifQ.yEOR4u6Vtv9CDpycZCBz5Q"
)

var mapboxCoder = mapbox.Geocoder(mapboxAPISecret)

func GetGeocodeOversea(addr string) (Location, error) {
	geo, err := mapboxCoder.Geocode(addr)
	if err != nil {
		return nil, err
	}
	return &Coordinate{
		Lat: geo.Lat,
		Lng: geo.Lng,
	}, nil
}
