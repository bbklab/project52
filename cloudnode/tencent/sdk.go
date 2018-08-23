package tencent

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bbklab/inf/pkg/orderparam"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
)

var (
	// DefaultAPIEndPoint is exported
	DefaultAPIEndPoint = "https://cvm.tencentcloudapi.com"
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

// APIHost is exported
func (sdk *SDK) APIHost() string {
	u, err := url.Parse(sdk.APIEndPoint())
	if err != nil {
		return "cvm.tencentcloudapi.com"
	}
	return u.Host
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
		method       = "GET"
		stringToSign = method + sdk.APIHost() + "/" + "?"
	)

	for _, key := range params.Keys() {
		stringToSign += fmt.Sprintf("%s=%s&", key, params.Get(key))
	}
	stringToSign = strings.TrimSuffix(stringToSign, "&") // remove tailing &

	// sign the query params and append to query params
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

	// for debug
	// json.NewEncoder(os.Stdout).Encode(params)

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

	// for debug
	// fmt.Println(string(bs))

	// resp status code always be 200
	if code := resp.StatusCode; code != 200 {
		return fmt.Errorf("%d - %s", code, string(bs))
	}

	// error response
	var errResp tchttp.ErrorResponse
	if err := sdk.bind(bs, &errResp); err == nil && errResp.Response.Error.Code != "" {
		return fmt.Errorf("%s - %s", errResp.Response.Error.Code, errResp.Response.Error.Message)
	}

	// expected response
	return sdk.bind(bs, &recv)
}

func (sdk *SDK) bind(bs []byte, recv interface{}) error {
	return json.Unmarshal(bs, &recv)
}

// See More:
// https://cloud.tencent.com/document/api/213/15692
func (sdk *SDK) publicParameters() *orderparam.Params {
	params := orderparam.New()
	params.Set("Timestamp", strconv.FormatInt(time.Now().Unix()+rand.Int63n(300), 10))
	params.Set("Nonce", strconv.Itoa(rand.Int()))
	params.Set("SecretId", sdk.AccessID())
	params.Set("SignatureMethod", "HmacSHA1")
	params.Set("Version", "2017-03-12")
	return params
	//params.Set("Signature", "")            // reset later
}

func (sdk *SDK) sign(message string) string {
	hashfun := hmac.New(sha1.New, []byte(sdk.AccessKey()))
	hashfun.Write([]byte(message))
	rawSignature := hashfun.Sum(nil)
	base64signature := base64.StdEncoding.EncodeToString(rawSignature)
	return base64signature
}

// See More:
// https://cloud.tencent.com/document/api/213/15694
func (sdk *SDK) isFatalError(err error) bool {
	if err == nil {
		return false
	}

	var (
		errMessage    = err.Error()
		fatalMessages = []string{
			"UnauthorizedOperation",
			"InvalidAction",
			// "UnsupportedRegion",
			// "LimitExceeded",
			// "ResourceInUse",
			// "AuthFailure",
			// "UnknownParameter",
			// "MissingParameter",
			// "InvalidParameterValue",
			// "InvalidParameter",
		}
	)

	for _, msg := range fatalMessages {
		if strings.Contains(errMessage, msg) {
			return true
		}
	}

	return false
}
