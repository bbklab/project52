package aws

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bbklab/inf/pkg/ptype"
)

// See: https://docs.aws.amazon.com/general/latest/gr/rande.html#ec2_region
// maybe need to periodic check out the latest version and update this mapping

var regmap = map[string]*Region{
	"us-east-1":      {"US East", "N. Virginia"},
	"us-east-2":      {"US East", "Ohio"},
	"us-west-1":      {"US West", "N. California"},
	"us-west-2":      {"US West", "Oregon"},
	"ap-northeast-1": {"Asia Pacific", "Tokyo"},
	"ap-northeast-2": {"Asia Pacific", "Seoul"},
	"ap-northeast-3": {"Asia Pacific", "Osaka-Local"},
	"ap-south-1":     {"Asia Pacific", "Mumbai"},
	"ap-southeast-1": {"Asia Pacific", "Singapore"},
	"ap-southeast-2": {"Asia Pacific", "Sydney"},
	"ca-central-1":   {"Canada", "Central"},
	"cn-north-1":     {"China", "Beijing"}, // fuck this off
	"cn-northwest-1": {"China", "Ningxia"}, // fuck this off
	"eu-central-1":   {"EU", "Frankfurt"},
	"eu-west-1":      {"EU", "Ireland"},
	"eu-west-2":      {"EU", "London"},
	"eu-west-3":      {"EU", "Paris"},
	"sa-east-1":      {"South America", "São Paulo"},
}

// mapping for region -> ami (Amazon Linux 2)
// See: https://aws.amazon.com/cn/amazon-linux-2/lts-candidate-2-release-notes/    ---> Amazon Linux 2 AMI IDs
// maybe need to periodic check out the latest version and update this mapping
//
// Amazon Linux 2 and the Amazon Linux AMI are supported and maintained Linux images provided by AWS.
// The following are some of the features of Amazon Linux 2 and Amazon Linux AMI:
//  - A stable, secure, and high-performance execution environment for applications running on Amazon EC2.
//  - Provided at no additional charge to Amazon EC2 users.
//  - yum based
var regamimap = map[string]string{
	"us-east-1":      "ami-b270a8cf",
	"us-east-2":      "ami-9bc0f1fe",
	"us-west-1":      "ami-62405102",
	"us-west-2":      "ami-d2f06baa",
	"ap-northeast-1": "ami-3dbcb441",
	"ap-northeast-2": "ami-deb916b0",
	"ap-northeast-3": "ami-2237395f", // not found this region
	"ap-south-1":     "ami-28560c47",
	"ap-southeast-1": "ami-4db5ec31",
	"ap-southeast-2": "ami-6b62ae09",
	"ca-central-1":   "ami-35ae2851",
	"cn-north-1":     "ami-7fd70e12", // fuck this off
	"cn-northwest-1": "ami-955d4af7", // fuck this off
	"eu-central-1":   "ami-104419fb",
	"eu-west-1":      "ami-86c695ff",
	"eu-west-2":      "ami-598b6a3e",
	"eu-west-3":      "ami-f22c9a8f",
	"sa-east-1":      "ami-a77225cb",
}

// See: https://aws.amazon.com/cn/ec2/instance-types/
// maybe need to periodic check out the latest version and update this mapping
//
var itmap = map[string]*InstanceTypeSpec{
	ec2.InstanceTypeT2Nano:      {1, 0.5},
	ec2.InstanceTypeT2Micro:     {1, 1},
	ec2.InstanceTypeT2Small:     {1, 2},
	ec2.InstanceTypeT2Medium:    {2, 4},
	ec2.InstanceTypeT2Large:     {2, 8},
	ec2.InstanceTypeT2Xlarge:    {4, 16},
	ec2.InstanceTypeT22xlarge:   {8, 32},
	ec2.InstanceTypeM5Large:     {2, 8},
	ec2.InstanceTypeM5Xlarge:    {4, 16},
	ec2.InstanceTypeM52xlarge:   {8, 32},
	ec2.InstanceTypeM54xlarge:   {16, 64},
	ec2.InstanceTypeM512xlarge:  {48, 192},
	ec2.InstanceTypeM524xlarge:  {96, 384},
	ec2.InstanceTypeM5dLarge:    {2, 8},
	ec2.InstanceTypeM5dXlarge:   {4, 16},
	ec2.InstanceTypeM5d2xlarge:  {8, 32},
	ec2.InstanceTypeM5d4xlarge:  {16, 64},
	ec2.InstanceTypeM5d12xlarge: {48, 192},
	ec2.InstanceTypeM5d24xlarge: {96, 384},
	ec2.InstanceTypeM4Large:     {2, 8},
	ec2.InstanceTypeM4Xlarge:    {4, 16},
	ec2.InstanceTypeM42xlarge:   {8, 32},
	ec2.InstanceTypeM44xlarge:   {16, 64},
	ec2.InstanceTypeM410xlarge:  {40, 160},
	ec2.InstanceTypeM416xlarge:  {64, 256},
	ec2.InstanceTypeC5Large:     {2, 4},
	ec2.InstanceTypeC5Xlarge:    {4, 8},
	ec2.InstanceTypeC52xlarge:   {8, 16},
	ec2.InstanceTypeC54xlarge:   {16, 32},
	ec2.InstanceTypeC59xlarge:   {36, 72},
	ec2.InstanceTypeC518xlarge:  {72, 144},
	ec2.InstanceTypeC5dLarge:    {2, 4},
	ec2.InstanceTypeC5dXlarge:   {4, 8},
	ec2.InstanceTypeC5d2xlarge:  {8, 16},
	ec2.InstanceTypeC5d4xlarge:  {16, 32},
	ec2.InstanceTypeC5d9xlarge:  {36, 72},
	ec2.InstanceTypeC5d18xlarge: {72, 144},
	ec2.InstanceTypeC4Large:     {2, 3.75},
	ec2.InstanceTypeC4Xlarge:    {4, 7.5},
	ec2.InstanceTypeC42xlarge:   {8, 15},
	ec2.InstanceTypeC44xlarge:   {16, 30},
	ec2.InstanceTypeC48xlarge:   {36, 60},
	ec2.InstanceTypeR4Large:     {2, 15.25},
	ec2.InstanceTypeR4Xlarge:    {4, 30.5},
	ec2.InstanceTypeR42xlarge:   {8, 61},
	ec2.InstanceTypeR44xlarge:   {16, 122},
	ec2.InstanceTypeR48xlarge:   {32, 244},
	ec2.InstanceTypeR416xlarge:  {64, 488},
	ec2.InstanceTypeR5Large:     {2, 16},
	ec2.InstanceTypeR5Xlarge:    {4, 32},
	ec2.InstanceTypeR52xlarge:   {8, 64},
	ec2.InstanceTypeR54xlarge:   {16, 128},
	ec2.InstanceTypeR512xlarge:  {48, 384},
	ec2.InstanceTypeR524xlarge:  {96, 768},
	ec2.InstanceTypeR5dLarge:    {2, 16},
	ec2.InstanceTypeR5dXlarge:   {4, 32},
	ec2.InstanceTypeR5d2xlarge:  {8, 64},
	ec2.InstanceTypeR5d4xlarge:  {16, 128},
	ec2.InstanceTypeR5d12xlarge: {48, 384},
	ec2.InstanceTypeR5d24xlarge: {96, 768},
	ec2.InstanceTypeX116xlarge:  {64, 976},
	ec2.InstanceTypeX132xlarge:  {128, 1952},
	ec2.InstanceTypeX1eXlarge:   {4, 122},
	ec2.InstanceTypeX1e2xlarge:  {8, 244},
	ec2.InstanceTypeX1e4xlarge:  {16, 488},
	ec2.InstanceTypeX1e8xlarge:  {32, 976},
	ec2.InstanceTypeX1e16xlarge: {64, 1952},
	ec2.InstanceTypeX1e32xlarge: {128, 3904},
	ec2.InstanceTypeZ1dLarge:    {2, 16},
	ec2.InstanceTypeZ1dXlarge:   {4, 32},
	ec2.InstanceTypeZ1d2xlarge:  {8, 64},
	ec2.InstanceTypeZ1d3xlarge:  {12, 96},
	ec2.InstanceTypeZ1d6xlarge:  {24, 192},
	ec2.InstanceTypeZ1d12xlarge: {48, 34},
	ec2.InstanceTypeI3Large:     {2, 15.25},
	ec2.InstanceTypeI3Xlarge:    {4, 30.5},
	ec2.InstanceTypeI32xlarge:   {8, 61},
	ec2.InstanceTypeI34xlarge:   {16, 122},
	ec2.InstanceTypeI38xlarge:   {32, 244},
	ec2.InstanceTypeI316xlarge:  {64, 488},
	ec2.InstanceTypeG34xlarge:   {16, 122},
	ec2.InstanceTypeG38xlarge:   {32, 244},
	ec2.InstanceTypeG316xlarge:  {64, 488},
	ec2.InstanceTypeP2Xlarge:    {4, 61},
	ec2.InstanceTypeP28xlarge:   {32, 488},
	ec2.InstanceTypeP216xlarge:  {64, 732},
	ec2.InstanceTypeP32xlarge:   {8, 61},
	ec2.InstanceTypeP38xlarge:   {32, 244},
	ec2.InstanceTypeP316xlarge:  {64, 488},
	ec2.InstanceTypeD2Xlarge:    {4, 30.5},
	ec2.InstanceTypeD22xlarge:   {8, 61},
	ec2.InstanceTypeD24xlarge:   {16, 122},
	ec2.InstanceTypeD28xlarge:   {36, 244},
	ec2.InstanceTypeF12xlarge:   {8, 122},
	ec2.InstanceTypeF116xlarge:  {64, 976},
	ec2.InstanceTypeH12xlarge:   {8, 32},
	ec2.InstanceTypeH14xlarge:   {16, 64},
	ec2.InstanceTypeH18xlarge:   {32, 128},
	ec2.InstanceTypeH116xlarge:  {64, 256},
}

// InstanceTypeSpec is exported
type InstanceTypeSpec struct {
	CPU    int     `json:"cpu"`
	Memory float64 `json:"memory"` // by GB
}

// Region is exported
type Region struct {
	Location string `json:"location"`
	City     string `json:"city"`
}

// aws zone is always be the form as "region{id}"
// the {id} always be: a,b,c,d,e, such as
// ap-southeast-1a ap-southeast-1e
// trim the zone suffix {id} then we got the region
func regionOfZone(zone string) string {
	return strings.TrimRightFunc(zone, func(r rune) bool {
		return r >= 'a' && r <= 'z'
	})
}

// ImageSorter sort the ec2.Image by CreationDate
//
//
type ImageSorter []*ec2.Image

func (s ImageSorter) Len() int      { return len(s) }
func (s ImageSorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ImageSorter) Less(i, j int) bool {
	ictime, _ := time.Parse(time.RFC3339, ptype.StringV(s[i].CreationDate))
	jctime, _ := time.Parse(time.RFC3339, ptype.StringV(s[j].CreationDate))
	return ictime.After(jctime)
}
