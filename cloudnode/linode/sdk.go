package linode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	// DefaultAPIEndPoint is exported
	DefaultAPIEndPoint = "https://api.linode.com/v4"
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

// Verify verifies configs about linode endpoint & access id & access key
func (sdk *SDK) Verify() error {
	_, err := sdk.ListEcses()
	return err
}

func (sdk *SDK) sendRequest(method, path string, data interface{}) (*http.Response, error) {
	// rewrite as full path
	path = sdk.APIEndPoint() + path

	// fill up the body buffer
	var (
		buf io.ReadWriter // should send nil instead of an empty io.Reader if no data would be sent
	)
	if data != nil {
		buf = bytes.NewBuffer(nil) // request Body holder
		if err := json.NewEncoder(buf).Encode(data); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, path, buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sdk.AccessKey()))

	return sdk.client.Do(req)
}

func (sdk *SDK) apiCall(method, path string, data interface{}, recv interface{}) error {
	// send request
	resp, err := sdk.sendRequest(method, path, data)
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
		return sdk.bind(bs, recv)
	}

	var errResp ErrorResponse
	if err := sdk.bind(bs, &errResp); err != nil {
		return fmt.Errorf("%d - %s", code, string(bs))
	}

	if len(errResp.Errors) == 0 {
		return fmt.Errorf("%d - %s", code, string(bs))
	}

	return &LnAPIError{code, errResp.Errors[0].Field, errResp.Errors[0].Reason}
}

// LnAPIError is exported
type LnAPIError struct {
	Code   int
	Field  string
	Reason string
}

// Error implement error interface
func (e *LnAPIError) Error() string {
	return fmt.Sprintf("LnAPIError: %d - %s - %s", e.Code, e.Field, e.Reason)
}

// TODO
func (sdk *SDK) isFatalError(err error) bool {
	return false
}

func (sdk *SDK) bind(bs []byte, recv interface{}) error {
	return json.Unmarshal(bs, &recv)
}
