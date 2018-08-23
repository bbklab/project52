package aws

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	// DefaultAPIEndPoint is exported
	// nouse actually, Amazon VPC directs your request to the us-east-1
	DefaultAPIEndPoint = "https://ec2.amazonaws.com"
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
	cfg  *Config
	sess *session.Session
}

// Setup is exported
func Setup(cfg *Config) (*SDK, error) {
	var ret = new(SDK)
	if err := cfg.Valid(); err != nil {
		return nil, err
	}
	ret.cfg = cfg

	acfg := aws.NewConfig()
	acfg.WithCredentials(credentials.NewCredentials(&credentials.StaticProvider{
		Value: credentials.Value{
			AccessKeyID:     cfg.AccessID,
			SecretAccessKey: cfg.AccessKey,
		},
	}))

	ret.sess = session.New(acfg) // save session reference
	return ret, nil
}

// SwitchRegion switch to new region and return a new ec2.EC2 client
func (sdk *SDK) SwitchRegion(region string) *ec2.EC2 {
	if region == "" {
		region = "us-east-1" // setup default region
	}
	newsess := sdk.sess.Copy(&aws.Config{Region: aws.String(region)})
	return ec2.New(newsess)
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
// See:
// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html
func (sdk *SDK) Verify() error {
	_, err := sdk.SwitchRegion("").DescribeAccountAttributes(&ec2.DescribeAccountAttributesInput{})
	return err
}

func (sdk *SDK) isFatalError(err error) bool {
	if err == nil {
		return false
	}

	var (
		errMessage    = err.Error()
		fatalMessages = []string{
			"InvalidAMIID",
		}
	)

	for _, msg := range fatalMessages {
		if strings.Contains(errMessage, msg) {
			return true
		}
	}

	return false

}
