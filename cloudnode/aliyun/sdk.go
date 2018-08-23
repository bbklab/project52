package aliyun

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/bbklab/inf/pkg/orderparam"
	"github.com/bbklab/inf/pkg/utils"
)

var (
	// DefaultAPIEndPoint is exported
	DefaultAPIEndPoint = "https://ecs.aliyuncs.com/"
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

// Verify verifies configs about aliyun endpoint & access id & access key
func (sdk *SDK) Verify() error {
	_, err := sdk.ListRegions()
	return err
}

func (sdk *SDK) apiCall(params *orderparam.Params, recv interface{}) error {

	// caculate string to be signed
	var (
		method                   = "GET"
		stringToSign             = method + "&" + orderparam.Escape("/") + "&"
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
	stringToSign += orderparam.Escape(canonicalizedQueryString)
	sig := sdk.sign(stringToSign)
	params.Set("Signature", orderparam.Escape(sig))

	// build our final request URL
	var (
		reqURL = sdk.APIEndPoint() + "?"
	)
	for idx, key := range params.Keys() {
		if idx == 0 {
			reqURL += key + "=" + params.Get(key)
		} else {
			reqURL += "&" + key + "=" + params.Get(key)
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

	if code < 400 {
		return sdk.bind(bs, &recv)
	}

	var errResp EcsErrorResponse
	if err := sdk.bind(bs, &errResp); err == nil {
		return fmt.Errorf("%s:%s", errResp.Code, errResp.Message)
	}
	return fmt.Errorf("%d - %s", code, string(bs))
}

func (sdk *SDK) bind(bs []byte, recv interface{}) error {
	return json.Unmarshal(bs, &recv)
}

// See More:
// https://help.aliyun.com/document_detail/25490.html?spm=5176.doc25492.6.790.N5VIVS
func (sdk *SDK) publicParameters() *orderparam.Params {
	params := orderparam.New()
	params.Set("Format", "JSON")
	params.Set("Version", "2014-05-26")
	params.Set("AccessKeyId", sdk.AccessID())
	params.Set("SignatureMethod", "HMAC-SHA1")
	params.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	params.Set("SignatureVersion", "1.0")
	params.Set("SignatureNonce", utils.RandomStringRange(32, append(utils.UpperAlpha, utils.Digits...))) // NOTE this must NOT be reused
	return params
	//params.Set("Signature", "")            // reset later
	//params.Set("ResourceOwnerAccount", "") // not required if not using AliYun-RAM
}

func (sdk *SDK) sign(message string) string {
	key := sdk.AccessKey() + "&" // NOTE
	hashfun := hmac.New(sha1.New, []byte(key))
	hashfun.Write([]byte(message))
	rawSignature := hashfun.Sum(nil)
	base64signature := base64.StdEncoding.EncodeToString(rawSignature)
	return base64signature
}

func (sdk *SDK) isFatalError(err error) bool {
	if err == nil {
		return false
	}

	var (
		errMessage    = err.Error()
		fatalMessages = []string{
			"Account.Arrearage",
			"InvalidAccountStatus.NotEnoughBalance",
			"InvalidParameter.ResourceOwnerAccount",
			"PaymentMethodNotFound",
			"InvalidPayMethod",
			"QuotaExceed.AfterpayInstance",
		}
	)

	for _, msg := range fatalMessages {
		if strings.Contains(errMessage, msg) {
			return true
		}
	}

	return false
}
