package cloudsvr

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"unicode"
)

// nolint
var (
	CLOUDFLAGKEY = "bbklab-cloudnode"
)

// Handler offers a common interface to access the cloud node service
type Handler interface {
	// Ping to verify the cloudsvr settings
	Ping() error

	// Type tell the cloudsvr type
	Type() string

	// ListNode list all cloud nodes with related labels key: CLOUDFLAGKEY
	ListNodes() ([]*CloudNode, error)

	// InspectNode show details of one cloud node
	// note: parameter regionOrZone is optional
	InspectNode(id, regionOrZone string) (interface{}, error)

	// NewNode create a new cloud node with prefered attributes
	// note: the returned attributes represents the actually used ones
	NewNode(prefer *PreferAttrs) (*CloudNode, *PreferAttrs, error)

	// RemoveNode remove a cloud node
	RemoveNode(node *CloudNode) error

	// ListCloudTypes list given region's cloud types
	// note: if no region given, will list all region's types
	ListCloudTypes(region string) ([]*CloudNodeType, error)

	// ListCloudRegions
	ListCloudRegions() ([]*CloudRegion, error)
}

type CloudNode struct {
	ID             string `json:"id" bson:"id"`                               // id of each cloud node (aliyun ecs id, tencent cvm id ...)
	RegionOrZoneID string `json:"region_or_zone_id" bson:"region_or_zone_id"` // aliyun called RegionID, qingcloud called ZoneID
	InstanceType   string `json:"instance_type" bson:"instance_type"`         // instance type
	CloudSvrType   string `json:"cloudsvr_type" bson:"cloudsvr_type"`         // cloudsvr type
	IPAddr         string `json:"ipaddr" bson:"ipaddr"`                       // public ipaddr
	Port           string `json:"port" bson:"port"`                           // ssh port
	User           string `json:"user" bson:"user"`                           // ssh user
	Password       string `json:"password" bson:"password"`                   // ssh password
	PrivKey        string `json:"privkey" bson:"privkey"`                     // private key text
	CreatTime      string `json:"create_time" bson:"create_time"`             // creation time
	Status         string `json:"status" bson:"status"`                       // current status
}

func (node *CloudNode) SSHAddr() string {
	return net.JoinHostPort(node.IPAddr, node.Port)
}

// PreferAttrs represents the prefered attributions of cloud node and the Handler
// will try to pick up this setting firstly while creating new cloud node
type PreferAttrs struct {
	RegionOrZone string `json:"region_or_zone"` // prefered region(aliyun) or zone(qingcloud)
	InstanceType string `json:"instance_type"`  // prefered instance type
}

func (pref *PreferAttrs) Valid() error {
	if reg, typ := pref.RegionOrZone, pref.InstanceType; reg == "" || typ == "" {
		return errors.New("invalid prefer cloud attributes, region & type required")
	}
	return nil
}

func (pref *PreferAttrs) Empty() bool {
	return pref.RegionOrZone == "" && pref.InstanceType == ""
}

type CloudRegion struct {
	ID       string `json:"id"`
	Location string `json:"location"` // readable
}

type CloudNodeType struct {
	ID       string `json:"id"`        // id
	Name     string `json:"name"`      // human readable text
	RegionID string `json:"region_id"` // optional, some of IAAS require this field, eg: qingcloud
	CPU      string `json:"cpu"`       // cpu count
	Memory   string `json:"memory"`    // by GB, eg: 0.5GB
	Disk     string `json:"disk"`      // by GB, eg: 20.0GB
}

type CloudNodeTypeSorter []*CloudNodeType

func (s CloudNodeTypeSorter) Len() int      { return len(s) }
func (s CloudNodeTypeSorter) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s CloudNodeTypeSorter) Less(i, j int) bool {
	mi := strings.TrimRightFunc(s[i].Memory, func(r rune) bool { return !unicode.IsNumber(r) })
	mj := strings.TrimRightFunc(s[j].Memory, func(r rune) bool { return !unicode.IsNumber(r) })
	miN, _ := strconv.ParseFloat(mi, 10)
	mjN, _ := strconv.ParseFloat(mj, 10)
	return miN < mjN
}
