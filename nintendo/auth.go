package nintendo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

var (
	defaultNintendo = NewClient("")
)

type Client struct {
	iksmSession string
}

func NewClient(session string) *Client {
	return &Client{
		iksmSession: session,
	}
}

func iksmCookie(cookie string) *http.Cookie {
	return &http.Cookie{
		Name:  "iksm_session",
		Value: cookie,
	}
}

func (n *Client) IksmCookie() *http.Cookie {
	return iksmCookie(n.iksmSession)
}

func (n *Client) AuthReq(method string, url *url.URL) (*http.Request, error) {
	req, err := http.NewRequest(method, "", nil)
	req.URL = url
	if err != nil {
		return nil, err
	}
	req.AddCookie(iksmCookie(n.iksmSession))
	return req, nil
}

type gameService interface {
	Domain() string
	Scheme() string
}

func newURL(s gameService) *url.URL {
	u := new(url.URL)
	u.Host = s.Domain()
	u.Scheme = s.Scheme()
	return u
}

type webService struct {
	s      gameService
	client *Client
}

func (w *webService) fullURL(endpoint string) *url.URL {
	u := newURL(w.s)
	u.Path = endpoint
	return u
}

func (w *webService) authReq(method, endpoint string) (*http.Request, error) {
	return w.client.AuthReq(method, w.fullURL(endpoint))
}

func (w *webService) Do(method, endpoint string) (*http.Response, error) {
	req, err := w.authReq(method, endpoint)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (w *webService) Api(method, endpoint string, result interface{}) error {
	res, err := w.Do(method, endpoint)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusForbidden:
		// token will expire
		return fmt.Errorf("SplatNet API error: %q, please reinitialize your token", res.Status)
	case http.StatusNotFound:
		return fmt.Errorf("SplatNet API error: %q", res.Status)
	}

	encoder := json.NewDecoder(res.Body)
	encoder.UseNumber()
	err = encoder.Decode(result)
	return err
}

func (n *Client) Splatoon() *SplatoonService {
	s := &SplatoonService{}
	s.webService = webService{
		client: n,
		s:      s, // tricky here, SplatoonService implement the gameService interface
	}
	return s
}
