package nintendo

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

const (
	// but you can keep sessionToken returned by LogIn, it won't expire
	// and can be passed to GetCookie again
	// GetCookie returns a different cookie each time, even you call it with the same sessionToken
	// because of the dynamic timestamp
	// however you can still reuse the generated cookie for as many times as you like

	// this should be regenerated each a few minutes (pasted from browser)
	// if it expires, some tests won't pass
	userPickBtnURL = "npf71b963c1b7b6d119://auth#session_state=c1508b5740ed988883e55e230eb465406404e4727af658f1b3f70db9d1500a13&session_token_code=eyJhbGciOiJIUzI1NiJ9.eyJzdGM6bSI6IlMyNTYiLCJhdWQiOiI3MWI5NjNjMWI3YjZkMTE5IiwianRpIjoiNDE5MTY3MjMwNzMiLCJ0eXAiOiJzZXNzaW9uX3Rva2VuX2NvZGUiLCJpc3MiOiJodHRwczovL2FjY291bnRzLm5pbnRlbmRvLmNvbSIsImV4cCI6MTYzMTg0ODY2OSwic3ViIjoiYjJiZjFkZTI4NGZjNjI2MyIsInN0YzpjIjoiVEJHRDVRdi1BSzNnZXpIY1JkRVBTeGcxUHlLVk9FY1J6TFZ1bEpyS3J0TSIsImlhdCI6MTYzMTg0ODA2OSwic3RjOnNjcCI6WzAsOCw5LDE3LDIzXX0.J29_CabuI6Bk2-OPPnuIAwh0koASxIeR0uNBILEy_eY&state=Uv38ByGCZU8WP18PmmIdcpVmx00QA3xNe7sEB9HixkmBhVrY"
)

func TestLogIn(t *testing.T) {
	sessionToken, err := LogInFromCommandLine(strings.NewReader(userPickBtnURL + "\n"))
	assert.NoError(t, err)
	fmt.Printf("session token: %s\n", sessionToken)
}

func TestGetHashFromS2sApi(t *testing.T) {
	naIdToken := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjBiOTZiOWFkLTJkYjQtNDMwOS1iYmUwLTFlNTJmZWRhM2Y2ZiIsImprdSI6Imh0dHBzOi8vYWNjb3VudHMubmludGVuZG8uY29tLzEuMC4wL2NlcnRpZmljYXRlcyJ9.eyJ0eXAiOiJ0b2tlbiIsImlhdCI6MTYzMTgxMTI3MywianRpIjoiZTA5ZTQ3MDAtODYyOC00YTZhLTkxOWMtOTMxZDUyNzEyM2YzIiwiYWM6c2NwIjpbMCw4LDksMTcsMjNdLCJhYzpncnQiOjY0LCJleHAiOjE2MzE4MTIxNzMsImF1ZCI6IjcxYjk2M2MxYjdiNmQxMTkiLCJzdWIiOiJiMmJmMWRlMjg0ZmM2MjYzIiwiaXNzIjoiaHR0cHM6Ly9hY2NvdW50cy5uaW50ZW5kby5jb20ifQ.MARMutigW3UnvxNHqEfoUDIR9JrRuxvPEVPItShWYRRUxXekpNkb8SaNDq_y_SoXy3HwljC547ImOwiIK710bMoMsQEqeh3b6JP_78U8ptMwEbSNE6sJX8cLmQjRFtZF-zYVlkDw4oJpiM1kojnxo6M8fKlkoIK2HaKS0uWJlX6E-QIYRwvEPGSn3sQFksaqAHly7by_KFgTqQXVjQ8EDuyxd6FQBW3JNFhOLuuLkcZtftfB6x5p3OI_wwuhokHKagow29oNBfUufCq521S5VZAQbUlpvUO-ECkyT613lxd9R9gehnEReRBA4YgC-sHczn68LI7kCEt8J9UfCRLmeg"
	var timestamp int64 = 1631811270

	hash, err := getHashFromS2sApi(naIdToken, timestamp)
	assert.NoError(t, err)
	fmt.Println(hash)
}

func TestGenNewCookie(t *testing.T) {
	nickname, cookie, err := GenNewCookie(strings.NewReader(userPickBtnURL))
	assert.NoError(t, err)
	assert.NotEmpty(t, cookie)
	fmt.Printf("%s, you cookie is %s\n", nickname, cookie)
}

func TestGetCookie(t *testing.T) {
	sessionToken := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJiMmJmMWRlMjg0ZmM2MjYzIiwiYXVkIjoiNzFiOTYzYzFiN2I2ZDExOSIsInR5cCI6InNlc3Npb25fdG9rZW4iLCJpYXQiOjE2MzE3ODQzMzEsImp0aSI6IjYyMzM0NjU0MzMiLCJpc3MiOiJodHRwczovL2FjY291bnRzLm5pbnRlbmRvLmNvbSIsImV4cCI6MTY5NDg1NjMzMSwic3Q6c2NwIjpbMCw4LDksMTcsMjNdfQ.F73I-WOBCHTltge13GAfxlR3YheycWWQ0F4dz-5iwKk"
	nickname, cookie, err := GetCookie(sessionToken, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, nickname)
	assert.NotEmpty(t, cookie)
	fmt.Println(nickname, cookie)
}

func TestGetUserinfo(t *testing.T) {
	sessionToken := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJiMmJmMWRlMjg0ZmM2MjYzIiwiYXVkIjoiNzFiOTYzYzFiN2I2ZDExOSIsInR5cCI6InNlc3Npb25fdG9rZW4iLCJpYXQiOjE2MzE3ODQzMzEsImp0aSI6IjYyMzM0NjU0MzMiLCJpc3MiOiJodHRwczovL2FjY291bnRzLm5pbnRlbmRvLmNvbSIsImV4cCI6MTY5NDg1NjMzMSwic3Q6c2NwIjpbMCw4LDksMTcsMjNdfQ.F73I-WOBCHTltge13GAfxlR3YheycWWQ0F4dz-5iwKk"
	userLang := ""
	ui, err := GetUserInfo(sessionToken, userLang)
	assert.NoError(t, err)
	fmt.Println(StringifyUserInfo(ui))
}
