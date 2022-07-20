package weather

import (
	"encoding/json"
	"fmt"
)

type errCap struct {
	Code    int    `json:"cod"`
	Message string `json:"message"`
}

func (ec errCap) Error() string {
	return fmt.Sprintf("error code: %d message: %s", ec.Code, ec.Message)
}

func SendPayload(url string, result interface{}) error {
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	interim := json.RawMessage{}
	ec := errCap{}
	err = json.NewDecoder(res.Body).Decode(&interim)
	if err != nil {
		return err
	}
	// try err cap
	err = json.Unmarshal(interim, &ec)
	if err == nil && ec.Code != 200 && ec.Code != 0 {
		return ec
	}

	err = json.Unmarshal(interim, result)
	if err != nil {
		return err
	}
	return nil
}
