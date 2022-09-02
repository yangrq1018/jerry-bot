package nintendo

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
