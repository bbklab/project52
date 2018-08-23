package qingcloud

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/bbklab/inf/pkg/orderparam"
)

var (
	// DefaultAPIEndPoint is exported
	DefaultAPIEndPoint = "https://api.qingcloud.com/iaas" // note: no trailing /
)

// Config is exported
type Config struct {
	EndPoint  string `json:"endpoint,omitempty"`
	AccessID  string `json:"access_id"`
	AccessKey string `json:"access_key"`
}

// Valid is exported
func (cfg *Config) Valid() error {
	if cfg.AccessID == "" {
		return errors.New("access id required")
	}
	if cfg.AccessKey == "" {
		return errors.New("access key required")
	}
	return nil
}

// SDK is an implement of CloudSvr
type SDK struct {
	cfg *Config
}

// Setup is exported
func Setup(cfg *Config) (*SDK, error) {
	if err := cfg.Valid(); err != nil {
		return nil, err
	}
	return &SDK{cfg: cfg}, nil
}

// APIEndPoint is exported
func (sdk *SDK) APIEndPoint() string {
	if addr := sdk.cfg.EndPoint; addr != "" {
		return addr
	}
	return DefaultAPIEndPoint
}

// AccessID is exported
func (sdk *SDK) AccessID() string {
	return sdk.cfg.AccessID
}

// AccessKey is exported
func (sdk *SDK) AccessKey() string {
	return sdk.cfg.AccessKey
}

// Verify verifies configs about qingcloud endpoint & access id & access key
func (sdk *SDK) Verify() error {
	_, err := sdk.ListZones()
	return err
}

func (sdk *SDK) apiCall(params *orderparam.Params, recv interface{}) error {

	// caculate string to be signed
	var (
		method                   = "GET"
		stringToSign             = method + "\n" + "/iaas/" + "\n"
		canonicalizedQueryString string
	)

	for _, key := range params.Keys() {
		if canonicalizedQueryString == "" {
			canonicalizedQueryString += orderparam.Escape(key) + "=" + orderparam.Escape(params.Get(key))
		} else {
			canonicalizedQueryString += "&" + orderparam.Escape(key) + "=" + orderparam.Escape(params.Get(key))
		}
	}

	// sign the query params and append to query params
	stringToSign += canonicalizedQueryString
	sig := sdk.sign(stringToSign)
	params.Set("signature", sig)

	// build our final request URL
	var (
		reqURL = sdk.APIEndPoint() + "/?" // note: here
	)
	for idx, key := range params.Keys() {
		if idx == 0 {
			reqURL += key + "=" + orderparam.Escape(params.Get(key))
		} else {
			reqURL += "&" + key + "=" + orderparam.Escape(params.Get(key))
		}
	}

	// send the request
	resp, err := http.Get(reqURL)
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

	var errResp BaseResponse
	if err := sdk.bind(bs, &errResp); err != nil {
		return fmt.Errorf("%d - %s", code, string(bs))
	}

	if errResp.RetCode != 0 {
		return &QcAPIError{errResp.RetCode, errResp.Message}
	}

	return sdk.bind(bs, recv)
}

// QcAPIError is exported
type QcAPIError struct {
	Code    int
	Message string
}

// Error implement error interface
func (e *QcAPIError) Error() string {
	return fmt.Sprintf("%d:%s", e.Code, e.Message)
}

func (sdk *SDK) bind(bs []byte, recv interface{}) error {
	return json.Unmarshal(bs, &recv)
}

// See More:
// https://docs.qingcloud.com/product/api/common/parameters
func (sdk *SDK) publicParameters() *orderparam.Params {
	params := orderparam.New()
	params.Set("time_stamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	params.Set("access_key_id", sdk.AccessID())
	params.Set("version", "1")
	params.Set("signature_method", "HmacSHA256")
	params.Set("signature_version", "1")
	return params
	// params.Set("signature", "") // must, set later
	// params.Set("action", "")    // must, set laster
	// params.Set("zone", "")      // optional, set on request
}

// See More:
// https://docs.qingcloud.com/product/api/common/signature
func (sdk *SDK) sign(message string) string {
	key := sdk.AccessKey() // key := "SECRETACCESSKEY" // for debug
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(message))
	signature := strings.TrimSpace(base64.StdEncoding.EncodeToString(h.Sum(nil)))
	signature = strings.Replace(signature, " ", "+", -1)
	return signature
}

func (sdk *SDK) isTemporarilyError(err error) bool {
	errv, ok := err.(*QcAPIError)
	if !ok {
		return false
	}

	msg := strings.ToLower(errv.Message)
	if strings.Contains(msg, "lease info not ready yet, please try later") {
		return true
	}

	return false
}

func (sdk *SDK) isFatalError(err error) bool {
	errv, ok := err.(*QcAPIError)
	if !ok {
		return false
	}

	switch errv.Code {
	case 2400, 5000, 5100, 5200, 5300:
		return true
	}
	return false
}

// DebugSignature is used for debug qingcloud request signature ...
func (sdk *SDK) DebugSignature() {
	pmap := map[string]string{
		"count":             "1",
		"vxnets.1":          "vxnet-0",
		"zone":              "pek1",
		"instance_type":     "small_b",
		"signature_version": "1",
		"signature_method":  "HmacSHA256",
		"instance_name":     "demo",
		"image_id":          "centos64x86a",
		"login_mode":        "passwd",
		"login_passwd":      "QingCloud20130712",
		"version":           "1",
		"access_key_id":     "QYACCESSKEYIDEXAMPLE",
		"action":            "RunInstances",
		"time_stamp":        "2013-08-27T14:30:10Z",
	}

	params := orderparam.New()
	for key, val := range pmap {
		params.Set(key, val)
	}

	var ret interface{}
	fmt.Println(sdk.apiCall(params, &ret))
}
