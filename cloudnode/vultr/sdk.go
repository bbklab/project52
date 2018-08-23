package vultr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	// DefaultAPIEndPoint is exported
	DefaultAPIEndPoint = "https://api.vultr.com/v1"
)

// Config is exported
type Config struct {
	EndPoint  string `json:"endpoint,omitempty"`
	AccessKey string `json:"access_key"`
}

// Valid is exported
func (cfg *Config) Valid() error {
	if cfg.AccessKey == "" {
		return errors.New("access key required")
	}
	return nil
}

// SDK is an implement of CloudSvr
type SDK struct {
	cfg    *Config
	client *http.Client // http client
}

// Setup is exported
func Setup(cfg *Config) (*SDK, error) {
	if err := cfg.Valid(); err != nil {
		return nil, err
	}
	return &SDK{cfg, &http.Client{Timeout: time.Second * 30}}, nil
}

// APIEndPoint is exported
func (sdk *SDK) APIEndPoint() string {
	if addr := sdk.cfg.EndPoint; addr != "" {
		return addr
	}
	return DefaultAPIEndPoint
}

// AccessKey is exported
func (sdk *SDK) AccessKey() string {
	return sdk.cfg.AccessKey
}

// Verify verifies configs about vultr endpoint & access id & access key
func (sdk *SDK) Verify() error {
	_, err := sdk.AccountInfo()
	return err
}

func (sdk *SDK) sendRequest(method, path string, values url.Values) (*http.Response, error) {
	// rewrite as full path
	path = sdk.APIEndPoint() + path

	// request body holder
	buf := bytes.NewBuffer(nil)
	if values != nil {
		buf.WriteString(values.Encode())
	}

	req, err := http.NewRequest(method, path, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("API-Key", sdk.AccessKey())
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded") // be posted content-type
	} else {
		req.Header.Set("Accept", "application/json") // expect to received content-type
	}

	return sdk.client.Do(req)
}

func (sdk *SDK) apiCall(method, path string, values url.Values, recv interface{}) error {
	var retried = 1

ApiCall:
	// send request
	resp, err := sdk.sendRequest(method, path, values)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// obtain response and bind to the reciving objects pointer
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	code := resp.StatusCode
	if code == 200 {
		if recv == nil {
			return nil
		}
		// while empty list, vultr return [] instead of {}, but it shouldn't, make fit wit it
		if string(bs) == `[]` {
			recv = nil
			return nil
		}
		return sdk.bind(bs, recv)
	}

	// if reached api request rate limit, backoff and retry
	if code == 503 && strings.Contains(string(bs), "Rate limit reached") {
		log.Warnf("backoff delay and retry as we hit the vultr api rate limit: %s - %s", method, path)
		retried++
		if retried >= 10 {
			retried = 10
		}
		time.Sleep(time.Second * time.Duration(retried))
		goto ApiCall
	}

	return fmt.Errorf("%d - %s", code, string(bs))
}

func (sdk *SDK) bind(bs []byte, recv interface{}) error {
	return json.Unmarshal(bs, &recv)
}

func (sdk *SDK) isFatalError(err error) bool {
	return false
}

// AccountInfo is exported
func (sdk *SDK) AccountInfo() (*Account, error) {
	var resp *Account
	err := sdk.apiCall("GET", "/account/info", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, err
}
