package nintendo

import (
	"bufio"
	"crypto"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// DefaultSessionToken can get from mobile sniff apps like Stream
// look for POST https://accounts.nintendo.com/connect/1.0.0/api/token.
// This serves as the seed for renew cookie
func DefaultSessionToken() string {
	return os.Getenv("NINTENDO_SESSION_TOKEN")
}

// Ses also: github.com/frozenpandaman/splatnet2statink

const (
	endpointAuthorize            = "https://accounts.nintendo.com/connect/1.0.0/authorize"
	endpointSessionToken         = "https://accounts.nintendo.com/connect/1.0.0/api/session_token"
	endpointApiToken             = "https://accounts.nintendo.com/connect/1.0.0/api/token"
	endpointUserMe               = "https://api.accounts.nintendo.com/2.0.0/users/me"
	endpointNintendoLogin        = "https://api-lp1.znc.srv.nintendo.net/v1/Account/Login"
	endpointWebserviceToken      = "https://api-lp1.znc.srv.nintendo.net/v2/Game/GetWebServiceToken"
	endpointSplatNet             = "https://app.splatoon2.nintendo.net/?lang=%s"
	endpointSplatNetShareResult  = "https://app.splatoon2.nintendo.net/api/share/results/%s"
	endpointSplatNetShareProfile = "https://app.splatoon2.nintendo.net/api/share/profile"

	// statink endpoint
	endpointStatinkUploadBattle = "https://stat.ink/api/v2/battle"
)

var client = http.Client{
	Timeout: 20 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	},
	Jar: nil,
}

// uRandom generates a random slice of bytes of length n
func uRandom(n int) []byte {
	var data = make([]byte, n)
	rand.Read(data)
	return data
}

// LogInFromCommandLine Logs in to a Nintendo Account and returns a session token
func LogInFromCommandLine(in io.Reader) (string, error) {
	postLoginURL, authCode, err := GeneratePostLogin()
	if err != nil {
		return "", err
	}
	// 提示用户在浏览器请求req，浏览器内的账号数据会触发展示账号选择列表，而不是直接302到login
	fmt.Println("Navigate to this URL in your browser:")
	fmt.Println(postLoginURL)
	fmt.Println("登录，右键\"选择此人\"，拷贝链接地址，粘贴到这里")
	var userAccountURL string
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		userAccountURL = scanner.Text()
		break
	}
	return ExtractSessionToken(userAccountURL, authCode)
}

func GeneratePostLogin() (*url.URL, string, error) {
	// 随机状态
	authState := base64.RawURLEncoding.EncodeToString(uRandom(36))
	authCodeVerifier := base64.RawURLEncoding.EncodeToString(uRandom(32))
	authCVHash := crypto.SHA256.New()
	authCVHash.Write([]byte(authCodeVerifier)) // 因为authCodeVerifier用的是RawEncoding所以不需要replace "=" (no padding)
	authCodeChallenge := base64.RawURLEncoding.EncodeToString(authCVHash.Sum(nil))

	values := makeQueryValues(map[string]string{
		"state":                               authState,
		"redirect_uri":                        "npf71b963c1b7b6d119://auth",
		"client_id":                           "71b963c1b7b6d119",
		"scope":                               "openid user user.birthday user.mii user.screenName",
		"response_type":                       "session_token_code", // ask for session token code
		"session_token_code_challenge":        authCodeChallenge,
		"session_token_code_challenge_method": "S256", // SHA256
		"theme":                               "login_form",
	})

	req, err := http.NewRequest("GET", endpointAuthorize, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header = makeHeader(map[string]string{
		"Host":                      "accounts.nintendo.com",
		"Connection":                "keep-alive",
		"Cache-Control":             "max-age=0",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                "Mozilla/5.0 (Linux; Android 7.1.2; Pixel Build/NJH47D; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/59.0.3071.125 Mobile Safari/537.36",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8n",
		"DNT":                       "1",
		"Accept-Encoding":           "gzip,deflate,br",
	})
	req.URL.RawQuery = values.Encode()
	res, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}

	// 在没有浏览器的Cookie信息情况下，服务器返回302重定向是登录，输入用户名密码
	// 在真实浏览器下，请求直接返回账号选择界面，可以在按钮的链接里找到服务器返回的session_token_code
	redirect := httpRedirections(res)

	// 这个骚操作是因为python requests弱化了*http.Request对象，所以原作者调用的形式是
	// r = session.get(url, headers=..., params=...)
	// 只能从r.history中还原最开始的请求
	// postLogin.URL == req.URL
	postLogin := redirect[len(redirect)-1] // i.e. req
	return postLogin.URL, authCodeVerifier, nil
}

func ExtractSessionToken(userAccountURL string, authCodeVerifier string) (string, error) {
	// parse
	match := regexp.MustCompile(`.*://auth#(.*)`).FindStringSubmatch(userAccountURL)
	if len(match) < 2 {
		return "", fmt.Errorf("invalid url: %q", userAccountURL)
	}
	queryString := match[1]
	values, err := url.ParseQuery(queryString)
	if err != nil {
		return "", err
	}
	sessionTokenCode := values.Get("session_token_code")
	if sessionTokenCode == "" {
		return "", fmt.Errorf("session_token_code not found")
	}
	sessionToken, err := getSessionToken(sessionTokenCode, authCodeVerifier)
	if err != nil {
		return "", err
	}
	return sessionToken, nil
}

// UserInfo 账号信息
type UserInfo struct {
	CreatedAt                 int64  `json:"createdAt"`
	IsChild                   bool   `json:"isChild"`
	AnalyticsOptedIn          bool   `json:"analyticsOptedIn"`
	AnalyticsOptedInUpdatedAt int64  `json:"analyticsOptedInUpdatedAt"`
	Nickname                  string `json:"nickname"`
	ID                        string `json:"id"`
	Gender                    string `json:"gender"`
	EmailVerified             bool   `json:"emailVerified"`
	EmailOptedIn              bool   `json:"emailOptedIn"`
	UpdatedAt                 int64  `json:"updatedAt"`
	ScreenName                string `json:"screenName"` // 邮箱
	ClientFriendsOptedIn      int64  `json:"clientFriendsOptedInUpdatedAt"`
	Birthday                  string `json:"birthday"`
	Language                  string `json:"language"`
	Country                   string `json:"country"`
	Timezone                  struct {
		UTCOffset        string `json:"utcOffset"`
		UTCOffsetSeconds int    `json:"utcOffsetSeconds"`
		ID               string `json:"id"`
		Name             string `json:"name"`
	} `json:"timezone"`
}

func StringifyUserInfo(user *UserInfo) string {
	return fmt.Sprintf(`账户信息
昵称: %s
账号: %s
创建日期: %s  
出生日期: %s
国家: %s
时区: %s
语言: %s
ID: %s
性别: %s
是否儿童: %s
`,
		user.Nickname, user.ScreenName,
		StringUnixDate(user.CreatedAt),
		user.Birthday, user.Country,
		user.Timezone.Name,
		user.Language, user.ID,
		user.Gender, chineseBool(user.IsChild),
	)
}

func chineseBool(b bool) string {
	if b {
		return "是"
	} else {
		return "否"
	}
}

func getIdToken(sessionToken, userLang string) (string, error) {
	appHead := makeHeader(map[string]string{
		"Host":            "accounts.nintendo.com",
		"Accept-Encoding": "gzip",
		"Content-Type":    "application/json; charset=utf-8",
		"Accept-Language": userLang,
		"Content-Length":  "439",
		"Accept":          "application/json",
		"Connection":      "Keep-Alive",
		"User-Agent":      "OnlineLounge/" + getNSOAppVersion() + " NASDKAPI Android",
	})

	var body = postPayload{
		"client_id":     "71b963c1b7b6d119", // Splatoon 2 service
		"session_token": sessionToken,
		"grant_type":    "urn:ietf:params:oauth:grant-type:jwt-bearer-session-token",
	}

	// different access token
	idResponse, err := postHTTPJson(endpointApiToken, body, appHead, nil)
	if err != nil {
		return "", err
	}

	// get user info
	idToken := idResponse.String("access_token")
	return idToken, nil
}

func GetUserInfo(sessionToken, userLang string) (*UserInfo, error) {
	idToken, err := getIdToken(sessionToken, userLang)
	if err != nil {
		return nil, err
	}
	userInfo, err := getUserInfo(idToken, userLang)
	if err != nil {
		return nil, err
	}
	return userInfo, nil
}

// GetCookie returns a new cookie (iksm_session) provided the session_token
func GetCookie(sessionToken, lang string) (string, string, error) {
	idToken, err := getIdToken(sessionToken, lang)
	if err != nil {
		return "", "", err
	}

	userInfo, err := getUserInfo(idToken, lang)
	if err != nil {
		return "", "", err
	}

	// 	get access token
	appHead := makeHeader(map[string]string{
		"Host":             "api-lp1.znc.srv.nintendo.net",
		"Accept-Language":  lang,
		"User-Agent":       "com.nintendo.znca/" + getNSOAppVersion() + " (Android/7.1.2)",
		"Accept":           "application/json",
		"X-ProductVersion": getNSOAppVersion(),
		"Content-Type":     "application/json; charset=utf-8",
		"Connection":       "Keep-Alive",
		"Authorization":    "Bearer",
		//"Content-Length":   "1036",
		"X-Platform":      "Android",
		"Accept-Encoding": "gzip",
	})

	timestamp := strconv.Itoa(int(time.Now().Unix()))
	guid := uuid.New().String()
	f, err := callImink(idToken, guid, timestamp, "1")
	if err != nil {
		return "", "", err
	}
	parameter := postPayload{
		"f":          f,
		"naIdToken":  idToken,
		"timestamp":  timestamp,
		"requestId":  guid,
		"naCountry":  userInfo.Country,
		"naBirthday": userInfo.Birthday,
		"language":   userInfo.Language,
	}
	body := postPayload{"parameter": parameter}
	splatoonToken, err := postHTTPJson(endpointNintendoLogin, body, appHead, nil)
	if err != nil {
		return "", "", err
	}

	// ugly way to handle undeclared deep json
	idToken = splatoonToken.Json("result").Json("webApiServerCredential").String("accessToken")
	f, err = callImink(idToken, guid, timestamp, "2") // call the second time
	if err != nil {
		return "", "", err
	}

	appHead = makeHeader(map[string]string{
		"Host":             "api-lp1.znc.srv.nintendo.net",
		"User-Agent":       "com.nintendo.znca/" + getNSOAppVersion() + " (Android/7.1.2)",
		"Accept":           "application/json",
		"X-ProductVersion": getNSOAppVersion(),
		"Content-Type":     "application/json; charset=utf-8",
		"Connection":       "Keep-Alive",
		"Authorization":    "Bearer " + idToken,
		"Content-Length":   "37",
		"X-Platform":       "Android",
		"Accept-Encoding":  "gzip",
	})

	body = postPayload{
		"parameter": postPayload{
			"id":                5741031244955648,
			"f":                 f,
			"registrationToken": idToken,
			"timestamp":         timestamp,
			"requestId":         guid,
		},
	}
	res, err := postHTTPJson(endpointWebserviceToken, body, appHead, nil)
	if err != nil {
		return "", "", err
	}
	splatoonAccessToken := res.Json("result").String("accessToken")
	appHead = makeHeader(map[string]string{
		"Host":                    "app.splatoon2.nintendo.net",
		"X-IsAppAnalyticsOptedIn": "false",
		"Accept":                  "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Encoding":         "gzip,deflate",
		"X-GameWebToken":          splatoonAccessToken,
		"Accept-Language":         lang,
		"X-IsAnalyticsOptedIn":    "false",
		"Connection":              "keep-alive",
		"DNT":                     "0",
		"User-Agent":              "Mozilla/5.0 (Linux; Android 7.1.2; Pixel Build/NJH47D; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/59.0.3071.125 Mobile Safari/537.36",
		"X-Requested-With":        "com.nintendo.znca",
	})

	cookies, err := getHTTPCookies(fmt.Sprintf(endpointSplatNet, lang), appHead)
	if err != nil {
		return "", "", err
	}
	var iksmSession string
	for _, cookie := range cookies {
		if cookie.Name == "iksm_session" {
			iksmSession = cookie.Value
		}
	}
	return userInfo.Nickname, iksmSession, nil
}

// GenNewCookie wait for manual paste from browser into in.
// Hints are sent to out
func GenNewCookie(in io.Reader) (string, string, error) {
	token, err := LogInFromCommandLine(in)
	if err != nil {
		return "", "", err
	}
	return GetCookie(token, "")
}

// helper function for login, obtain session_token from session_token_code
// sessionTokenCode中已经包含了加密的认证账户信息 (从浏览器cookie提取)
func getSessionToken(sessionTokenCode, authCodeVerifier string) (string, error) {
	var appHead = makeHeader(map[string]string{
		"User-Agent":      "OnlineLounge/" + getNSOAppVersion() + " NASDKAPI Android",
		"Accept-Language": "en-US",
		"Accept":          "application/json",
		"Content-Type":    "application/x-www-form-urlencoded",
		"Content-Length":  "540",
		"Host":            "accounts.nintendo.com",
		"Connection":      "Keep-Alive",
		"Accept-Encoding": "gzip",
	})
	body := map[string]interface{}{
		"client_id":                   "71b963c1b7b6d119", // Splatoon 2 service
		"session_token_code":          sessionTokenCode,
		"session_token_code_verifier": authCodeVerifier,
	}

	res, err := postHTTPJson(endpointSessionToken, body, appHead, nil)
	if err != nil {
		return "", err
	}
	if err = errInRes(res); err != nil {
		return "", err
	}
	return res.String("session_token"), nil
}

func callImink(idToken, guid, timestamp, step string) (string, error) {
	header := makeHeader(map[string]string{
		"User-Agent":   "splatnet2statink/" + agentVersion,
		"Content-Type": "application/json; charset=utf-8",
	})
	body := postPayload{
		"timestamp":  timestamp,
		"requestId":  guid,
		"hashMethod": step,
		"token":      idToken,
	}
	res, err := postHTTPJson("https://api.imink.app/f", body, header, nil)
	if err != nil {
		return "", err
	}
	return res.String("f"), nil
}

// accessToken should be the token returned by getIdToken
func getUserInfo(accessToken string, userLang string) (*UserInfo, error) {
	appHead := makeHeader(map[string]string{
		"User-Agent":      "OnlineLounge/" + getNSOAppVersion() + " NASDKAPI Android",
		"Accept-Language": userLang,
		"Accept":          "application/json",
		"Authorization":   "Bearer " + accessToken, // Bearer OAuth
		"Host":            "api.accounts.nintendo.com",
		"Connection":      "Keep-Alive",
		"Accept-Encoding": "gzip", // handle zip decode
	})
	var user UserInfo
	err := getHTTPUnmarshal(endpointUserMe, appHead, &user)
	return &user, err
}
