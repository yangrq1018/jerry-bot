package nintendo

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vmihailenco/msgpack/v5"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// network helpers

// httpRedirections tracks the history of requests in redirections from the final response resp
// in backwards order
// this is like python requests.Get("").history
func httpRedirections(resp *http.Response) (history []*http.Request) {
	for resp != nil {
		req := resp.Request
		history = append(history, req)
		resp = req.Response
	}
	return
}

// handy type for post body data
type postPayload map[string]interface{}

func (p postPayload) urlEncode() (io.Reader, error) {
	values := url.Values{}
	for k, v := range p {
		var vv string
		switch x := v.(type) {
		case string:
			vv = x
		case int64:
			vv = strconv.FormatInt(x, 10)
		default:
			return nil, fmt.Errorf("postHTTPURLEncodedForm cannot handle payload of type %T", v)
		}
		values.Set(k, vv)
	}
	return strings.NewReader(values.Encode()), nil
}

func makeHeader(m map[string]string) http.Header {
	h := make(http.Header)
	for k, v := range m {
		h.Set(k, v)
	}
	return h
}

func makeQueryValues(m map[string]string) url.Values {
	u := make(url.Values)
	for k, v := range m {
		u.Set(k, v)
	}
	return u
}

// post: body of multipart/form, text fields
func postHTTPMultipartForm(endpoint string, form multipartForm, header http.Header, cookies []*http.Cookie) (JSONResponse, error) {
	var body = bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	for k, v := range form {
		// only string key and string value allowed, int convert to string please
		err := writer.WriteField(k, v)
		if err != nil {
			return nil, err
		}
	}
	err := writer.Close()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", endpoint, body)
	if header != nil {
		// be careful of alias here
		setReqHeaderCopy(req, header)
	}
	// remember set content-type
	// like multipart/form; boundary=...
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return sendReqWithCookies(req, cookies)
}

func setReqHeaderCopy(req *http.Request, h http.Header) {
	req.Header = h.Clone()
}

// post: body of raw (JSON)
func postHTTPJson(endpoint string, payload postPayload, header http.Header, cookies []*http.Cookie) (JSONResponse, error) {
	var body = bytes.NewBuffer(nil)
	if payload != nil {
		err := json.NewEncoder(body).Encode(payload)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}
	if header != nil {
		setReqHeaderCopy(req, header)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return sendReqWithCookies(req, cookies)
}

var clientNoRedirect = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Timeout: 20 * time.Second,
}

// disable redirect
func postHTTPMsgPack(endpoint string, payload interface{}, header http.Header, client *http.Client) (*http.Response, error) {
	var body = bytes.NewBuffer(nil)
	mpEncoder := msgpack.NewEncoder(body)
	mpEncoder.SetCustomStructTag("json")
	err := mpEncoder.Encode(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}
	if header != nil {
		setReqHeaderCopy(req, header)
	}
	req.Header.Set("Content-Type", "application/x-msgpack")
	res, err := client.Do(req)
	return res, err
}

func sendReqWithCookies(req *http.Request, cookies []*http.Cookie) (JSONResponse, error) {
	for i := range cookies {
		req.AddCookie(cookies[i])
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		body, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("nintendo response: status %s, message %s", res.Status, string(body))
	}
	return extractJsonResponse(res)
}

func postHTTPURLEncodedForm(endpoint string, payload postPayload, header http.Header) (JSONResponse, error) {
	body, err := payload.urlEncode()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return nil, err
	}
	if header != nil {
		setReqHeaderCopy(req, header)
	}
	// See: https://github.com/frozenpandaman/splatnet2statink/wiki/api-docs
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return extractJsonResponse(res)
}

func extractJsonResponse(res *http.Response) (JSONResponse, error) {
	stream, err := autoDecode(res)
	if err != nil {
		return nil, err
	}
	var data JSONResponse
	err = json.NewDecoder(stream).Decode(&data)
	return data, err
}

// autoDecode handles response data transcoding automatically
func autoDecode(res *http.Response) (io.Reader, error) {
	var stream = res.Body
	var err error
	if res.Header.Get("Content-Encoding") == "gzip" {
		stream, err = gzip.NewReader(res.Body)
		return stream, err
	}
	return stream, nil
}

func getHTTPCookies(endpoint string, header http.Header) ([]*http.Cookie, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	setReqHeaderCopy(req, header)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return res.Cookies(), nil
}

func getHTTPUnmarshal(endpoint string, header http.Header, data interface{}) error {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}
	setReqHeaderCopy(req, header)
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	stream, err := autoDecode(res)
	if err != nil {
		return err
	}
	err = json.NewDecoder(stream).Decode(data)
	return err
}

func getHTTPJson(endpoint string, header http.Header) (JSONResponse, error) {
	var data JSONResponse
	err := getHTTPUnmarshal(endpoint, header, &data)
	return data, err
}

// JSONResponse represents json data sent from server
type JSONResponse map[string]interface{}

func (j JSONResponse) Error(key string) error {
	e, ok := j[key]
	if !ok {
		return nil
	}
	es, ok := e.(string)
	if !ok {
		return nil
	}
	return errors.New(es)
}

func (j JSONResponse) String(key string) string {
	if j[key] == nil {
		return ""
	}
	return j[key].(string)
}

// Json returns a field that is a JSON object.
// if such a key is missing or is nil, return an empty map
func (j JSONResponse) Json(key string) JSONResponse {
	// json unmarshal knows nothing about JSONResponse
	if j[key] == nil {
		return make(map[string]interface{})
	}
	return j[key].(map[string]interface{})
}

func errInRes(res JSONResponse) error {
	if es, ok := res["error"].(string); ok && es != "" {
		return fmt.Errorf("error: %s, description: %q", es, res.String("error_description"))
	}
	return nil
}
