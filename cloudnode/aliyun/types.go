// Package aliyun ...
//
// Most borrowed from:
//   github.com/ChangjunZhao/aliyun-api-golang/ecs
//
package aliyun

import (
	"fmt"

	"github.com/bbklab/inf/pkg/orderparam"
)

var (
	fieldCanNotNullErrMsg = "%s can not null."
)

// EcsBaseResponse is exported
type EcsBaseResponse struct {
	RequestID string `json:"RequestId"`
}

// EcsErrorResponse is exported
type EcsErrorResponse struct {
	EcsBaseResponse
	HostID  string `json:"HostId"`
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

// RegionType is exported
type RegionType struct {
	RegionID  string `json:"RegionId"`
	LocalName string `json:"LocalName"`
}

// Regions is exported
type Regions struct {
	Regions []RegionType `json:"Region"`
}

// DescribeRegionsResponse is exported
type DescribeRegionsResponse struct {
	EcsBaseResponse
	Response Regions `json:"Regions"`
}

// Instances is exported
type Instances struct {
	Instance []InstanceAttributesType `json:"Instance"`
}

// InstanceAttributesType is exported
type InstanceAttributesType struct {
	InstanceID              string                  `json:"InstanceId"`
	InstanceName            string                  `json:"InstanceName"`
	Description             string                  `json:"Description"`
	ImageID                 string                  `json:"ImageId"`
	RegionID                string                  `json:"RegionId"`
	ZoneID                  string                  `json:"ZoneId"`
	InstanceType            string                  `json:"InstanceType"`
	HostName                string                  `json:"HostName"`
	Status                  string                  `json:"Status"`
	SecurityGroupIDs        SecurityGroupIDSetType  `json:"SecurityGroupIds"`
	InnerIPAddress          IPAddressSetType        `json:"InnerIpAddress"`
	PublicIPAddress         IPAddressSetType        `json:"PublicIpAddress"`
	InternetMaxBandwidthIn  int                     `json:"InternetMaxBandwidthIn"`
	InternetMaxBandwidthOut int                     `json:"InternetMaxBandwidthOut"`
	InternetChargeType      string                  `json:"InternetChargeType"`
	CreationTime            string                  `json:"CreationTime"`
	VpcAttributes           VpcAttributesType       `json:"VpcAttributes"`
	EipAddress              EipAddressAssociateType `json:"EipAddress"`
	InstanceNetworkType     string                  `json:"InstanceNetworkType"`
	OperationLocks          OperationLocksType      `json:"OperationLocks"`
	Tags                    Tags                    `json:"Tags"` // extended
}

// Tags is exported
type Tags struct {
	Tag []TagType `json:"tag"`
}

// TagType is exported
type TagType struct {
	TagKey   string `json:"TagKey"`
	TagValue string `json:"TagValue"`
}

// DescribeInstancesResponse is exported
type DescribeInstancesResponse struct {
	EcsBaseResponse
	TotalCount int       `json:"TotalCount"`
	PageNumber int       `json:"PageNumber"`
	PageSize   int       `json:"PageSize"`
	Instances  Instances `json:"Instances"`
}

// CreateInstanceRequest is exported
// See More:
// https://help.aliyun.com/document_detail/25499.html?spm=5176.doc25506.6.798.eeuP5g
type CreateInstanceRequest struct {
	RegionID                string // must
	ZoneID                  string
	ImageID                 string // must
	InstanceType            string // must: https://help.aliyun.com/document_detail/25378.html?spm=5176.doc25499.2.10.dzAw7Z
	SecurityGroupID         string // must
	Password                string // not required, if setup, must call Api via https
	InstanceName            string
	Description             string
	InternetChargeType      string
	InternetMaxBandwidthIn  string
	InternetMaxBandwidthOut string
	HostName                string
	IoOptimized             string
	SystemDiskCategory      string
	SystemDiskDiskName      string
	SystemDiskDescription   string
	VSwitchID               string
	PrivateIPAddress        string
	InstanceChargeType      string            // PrePaid, PostPaid(default, require RMB 100+)
	Period                  int               // required if InstanceChargeType == PrePaid
	Labels                  map[string]string // max 5
}

// Validate is exported
func (r *CreateInstanceRequest) Validate() error {
	if r.RegionID == "" {
		return fmt.Errorf(fieldCanNotNullErrMsg, "RegionId")
	}
	if r.ImageID == "" {
		return fmt.Errorf(fieldCanNotNullErrMsg, "ImageId")
	}
	if r.InstanceType == "" {
		return fmt.Errorf(fieldCanNotNullErrMsg, "InstanceType")
	}
	if r.SecurityGroupID == "" {
		return fmt.Errorf(fieldCanNotNullErrMsg, "SecurityGroupId")
	}
	if r.Password == "" {
		return fmt.Errorf(fieldCanNotNullErrMsg, "Password")
	}
	if r.InstanceChargeType == "PrePaid" && r.Period == 0 {
		return fmt.Errorf(fieldCanNotNullErrMsg, "Period")
	}
	if len(r.Labels) > 5 {
		return fmt.Errorf("maximum five labels allowed")
	}
	return nil
}

// AddToParameters should be called after Validate verify checking
func (r *CreateInstanceRequest) AddToParameters(params *orderparam.Params) {
	params.Set("RegionId", r.RegionID)
	params.Set("ImageId", r.ImageID)
	params.Set("InstanceType", r.InstanceType)
	params.Set("SecurityGroupId", r.SecurityGroupID)
	params.Set("Password", r.Password)
	params.SetIgnoreNull("ZoneId", r.ZoneID)
	params.SetIgnoreNull("InstanceName", r.InstanceName)
	params.SetIgnoreNull("Description", r.Description)
	params.SetIgnoreNull("InternetChargeType", r.InternetChargeType)
	if r.InternetChargeType == "PayByBandwidth" {
		params.SetIgnoreNull("InternetMaxBandwidthIn", r.InternetMaxBandwidthIn)
		params.SetIgnoreNull("InternetMaxBandwidthOut", r.InternetMaxBandwidthOut)
	} else {
		params.SetIgnoreNull("InternetMaxBandwidthOut", r.InternetMaxBandwidthOut)
	}
	params.SetIgnoreNull("HostName", r.HostName)
	params.SetIgnoreNull("IoOptimized", r.IoOptimized)
	params.SetIgnoreNull("SystemDisk.Category", r.SystemDiskCategory)
	params.SetIgnoreNull("SystemDisk.DiskName", r.SystemDiskDiskName)
	params.SetIgnoreNull("SystemDisk.Description", r.SystemDiskDescription)
	params.SetIgnoreNull("VSwitchId", r.VSwitchID)
	params.SetIgnoreNull("PrivateIpAddress", r.PrivateIPAddress)
	params.SetIgnoreNull("InstanceChargeType", r.InstanceChargeType)

	var n int
	for key, val := range r.Labels {
		n++
		if key != "" && val != "" {
			params.Set(fmt.Sprintf("Tag.%d.Key", n), key)
			params.Set(fmt.Sprintf("Tag.%d.Value", n), val)
		}
	}
}

// CreateInstanceResponse is exported
type CreateInstanceResponse struct {
	EcsBaseResponse
	InstanceID string `json:"InstanceId"`
}

// InstanceMonitorDataType is exported
type InstanceMonitorDataType struct {
	InstanceID        string
	CPU               int
	IntranetRX        int
	IntranetTX        int
	IntranetBandwidth int
	InternetRX        int
	InternetTX        int
	InternetBandwidth int
	IOPSRead          int
	IOPSWrite         int
	BPSRead           int
	BPSWrite          int
	TimeStamp         string
}

// InstanceTypeItemType is exported
type InstanceTypeItemType struct {
	InstanceTypeID       string  `json:"InstanceTypeId"`
	CPUCoreCount         int     `json:"CpuCoreCount"`
	MemorySize           float64 `json:"MemorySize"`
	GPUSpec              string  `json:"GPUSpec"`
	GPUAmount            int     `json:"GPUAmount"`
	InstanceTypeFamily   string  `json:"InstanceTypeFamily"`
	LocalStorageCategory string  `json:"LocalStorageCategory"`
}

// InstanceTypes is exported
type InstanceTypes struct {
	InstanceType []InstanceTypeItemType `json:"InstanceType"`
}

// DescribeInstanceTypesResponse is exported
type DescribeInstanceTypesResponse struct {
	EcsBaseResponse
	InstanceTypes InstanceTypes `json:"InstanceTypes"`
}

// SecurityGroupIDSetType is exported
//
type SecurityGroupIDSetType struct {
	SecurityGroupID []string `json:"SecurityGroupId"`
}

// CreateSecurityGroupRequest is exported
// See More:
// https://help.aliyun.com/document_detail/25553.html?spm=5176.doc25499.6.857.CzabIR
type CreateSecurityGroupRequest struct {
	RegionID          string // must
	SecurityGroupName string
	Description       string
	VpcID             string
}

// Validate is exported
func (r *CreateSecurityGroupRequest) Validate() error {
	if r.RegionID == "" {
		return fmt.Errorf(fieldCanNotNullErrMsg, "RegionId")
	}
	return nil
}

// AddToParameters should be called after Validate verify checking
func (r *CreateSecurityGroupRequest) AddToParameters(params *orderparam.Params) {
	params.Set("RegionId", r.RegionID)
	params.SetIgnoreNull("SecurityGroupName", r.SecurityGroupName)
	params.SetIgnoreNull("Description", r.Description)
	params.SetIgnoreNull("VpcId", r.VpcID)
}

// CreateSecurityGroupResponse is exported
type CreateSecurityGroupResponse struct {
	EcsBaseResponse
	SecurityGroupID string `json:"SecurityGroupId"`
}

// AuthorizeSecurityGroupRequest is exported
type AuthorizeSecurityGroupRequest struct {
	SecurityGroupID         string // must
	RegionID                string // must
	IPProtocol              string // must tcp/udp/icmp/gre/all
	PortRange               string // must 1-100 / -1/-1
	Policy                  string // accept, drop
	NicType                 string // internet(default), intranet
	Priority                string // [1-100]
	SourceCidrIP            string // 0.0.0.0/0 (default)
	SourceGroupID           string
	SourceGroupOwnerAccount string
	DestCidrIP              string
	DestGroupID             string
	DestGroupOwnerAccount   string
}

// Validate is exported
func (r *AuthorizeSecurityGroupRequest) Validate() error {
	if r.SecurityGroupID == "" {
		return fmt.Errorf(fieldCanNotNullErrMsg, "SecurityGroupId")
	}
	if r.RegionID == "" {
		return fmt.Errorf(fieldCanNotNullErrMsg, "RegionId")
	}
	if r.IPProtocol == "" {
		return fmt.Errorf(fieldCanNotNullErrMsg, "IpProtocol")
	}
	if r.PortRange == "" {
		return fmt.Errorf(fieldCanNotNullErrMsg, "PortRange")
	}
	return nil
}

// AddToParameters is exported
func (r *AuthorizeSecurityGroupRequest) AddToParameters(params *orderparam.Params) {
	params.Set("SecurityGroupId", r.SecurityGroupID)
	params.Set("RegionId", r.RegionID)
	params.Set("IpProtocol", r.IPProtocol)
	params.Set("PortRange", r.PortRange)
	params.SetIgnoreNull("Policy", r.Policy)
	params.SetIgnoreNull("NicType", r.NicType)
	params.SetIgnoreNull("Priority", r.Priority)
	params.SetIgnoreNull("SourceCidrIp", r.SourceCidrIP)
	params.SetIgnoreNull("SourceGroupId", r.SourceGroupID)
	params.SetIgnoreNull("SourceGroupOwnerAccount", r.SourceGroupOwnerAccount)
	params.SetIgnoreNull("DestCidrIp", r.DestCidrIP)
	params.SetIgnoreNull("DestGroupId", r.DestGroupID)
	params.SetIgnoreNull("DestGroupOwnerAccount", r.DestGroupOwnerAccount)
}

// DescribeRegionZonesResponse is exported
//
// NOTE: no use because the returned data is not exactly
type DescribeRegionZonesResponse struct {
	EcsBaseResponse
	Zones Zones `json:"Zones"`
}

// Zones is exported
type Zones struct {
	Zone []ZoneType `json:"Zone"`
}

// ZoneType is exported
type ZoneType struct {
	ZoneID                    string                        `json:"ZoneId"`
	LocalName                 string                        `json:"LocalName"`
	AvailableResources        AvailableResources            `json:"AvailableResources"`
	AvailableInstanceTypes    AvailableInstanceTypes        `json:"AvailableInstanceTypes"`
	AvailableResourceCreation AvailableResourceCreationType `json:"AvailableResourceCreation"`
	AvailableDiskCategories   AvailableDiskCategoriesType   `json:"AvailableDiskCategories"`
	AvailableVolumeCategories AvailableVolumeCategories     `json:"AvailableVolumeCategories"`
}

// AvailableResources is exported
type AvailableResources struct {
	// contains all of supported DiskCategory / NetworkCategory / InstanceType / SystemDiskCategory
	// but the returned data is not exactly, so we do NOT actually using this
	ResourcesInfo []interface{} `json:"ResourcesInfo"`
}

// AvailableInstanceTypes is exported
type AvailableInstanceTypes struct {
	InstanceTypes []string `json:"InstanceTypes"`
}

// AvailableResourceCreationType is exported
type AvailableResourceCreationType struct {
	ResourceTypes []string `json:"ResourceTypes"` // Instance, Disk, VSwitch
}

// AvailableDiskCategoriesType is exported
type AvailableDiskCategoriesType struct {
	DiskCategories []string `json:"DiskCategories"` // cloud, ephemeral, ephemeral_ssd
}

// AvailableVolumeCategories is exported
type AvailableVolumeCategories struct {
	VolumeCategories []string `json:"VolumeCategories"`
}

// ClusterType is exported
//
type ClusterType struct {
	ClusterID string `json:"ClusterId"`
}

// SnapshotType is exported
//
type SnapshotType struct {
	SnapshotID     string `json:"SnapshotId"`
	SnapshotName   string `json:"SnapshotName"`
	Description    string `json:"Description"`
	Progress       string `json:"Progress"`
	SourceDiskID   string `json:"SourceDiskId"`
	SourceDiskSize int    `json:"SourceDiskSize"`
	SourceDiskType string `json:"SourceDiskType"`
	ProductCode    string `json:"ProductCode"`
	CreationTime   string `json:"CreationTime"`
}

//
// IPRange is exported
//

// IPRangeSetType is exported
type IPRangeSetType struct {
	IPAddress string // CIDR
	NicType   string // internet, intranet
}

// IPAddressSetType is exported
type IPAddressSetType struct {
	IPAddress []string `json:"IpAddress"`
}

//
// VPC
//

// VpcAttributesType is exported
type VpcAttributesType struct {
	VpcID            string           `json:"VpcId"`
	VSwitchID        string           `json:"VSwitchId"`
	PrivateIPAddress IPAddressSetType `json:"PrivateIpAddress"`
	NatIPAddress     string           `json:"NatIpAddress"`
}

// VpcSetType is exported
type VpcSetType struct {
	VpcID        string
	RegionID     string
	Status       string
	VpcName      string
	VSwitchIDs   string
	CidrBlock    string
	VRouterID    string
	Description  string
	CreationTime string
}

//
// VRouter
//

// VRouterSetType is exported
type VRouterSetType struct {
	VRouterID     string
	RegionID      string
	VpcID         string
	RouteTableIDs string
	VRouterName   string
	Description   string
	CreationTime  string
}

// RouteTableSetType is exported
type RouteTableSetType struct {
	VRouterID      string
	RouteTableID   string
	RouteEntrys    []RouteEntrySetType
	RouteTableType string
	CreationTime   string
}

// RouteEntrySetType is exported
type RouteEntrySetType struct {
	RouteTableID         string
	DestinationCidrBlock string
	Type                 string
	NextHopID            string
	Status               string
}

//
// VSwitch
//

// VSwitchSetType is exported
type VSwitchSetType struct {
	VSwitchID               string
	VpcID                   string
	Status                  string
	CidrBlock               string
	ZoneID                  string
	AvailableIPAddressCount int
	Description             string
	VSwitchName             string
	CreationTime            string
}

//
// EIP
//

// EipAddressAssociateType is exported
type EipAddressAssociateType struct {
	AllocationID       string `json:"AllocationId"`
	IPAddress          string `json:"IpAddress"`
	Bandwidth          int    `json:"Bandwidth"`
	InternetChargeType string `json:"InternetChargeType"`
}

// EipAddressSetType is exported
type EipAddressSetType struct {
	RegionID           string
	IPAddress          string
	AllocationID       string
	Status             string
	InstanceID         string
	Bandwidth          int
	InternetChargeType string
	OperationLocks     OperationLocksType
	AllocationTime     string
}

// EipMonitorDataType is exported
type EipMonitorDataType struct {
	EipRX        int
	EipTX        int
	EipFlow      int
	EipBandwidth int
	EipPackets   int
	TimeStamp    string
}

//
// AutoSnapshot
//

// AutoSnapshotPolicyType is exported
type AutoSnapshotPolicyType struct {
	SystemDiskPolicyEnabled           string
	SystemDiskPolicyTimePeriod        int
	SystemDiskPolicyRetentionDays     int
	SystemDiskPolicyRetentionLastWeek string
	DataDiskPolicyEnabled             string
	DataDiskPolicyTimePeriod          int
	DataDiskPolicyRetentionDays       int
	DataDiskPolicyRetentionLastWeek   string
}

// AutoSnapshotExecutionStatusType is exported
type AutoSnapshotExecutionStatusType struct {
	SystemDiskExecutionStatus string
	DataDiskExecutionStatus   string
}

//
// OperationLock
//

// OperationLocksType is exported
type OperationLocksType struct {
	LockReason []string `json:"LockReason"` // financial, security
}

//
// Disk
//

// DiskItemType is exported
type DiskItemType struct {
	DiskID             string
	RegionID           string
	ZoneID             string
	DiskName           string
	Description        string
	Type               string
	Category           string
	Size               int
	ImageID            string
	SourceSnapshotID   string
	ProductCode        string
	Portable           string
	Status             string
	OperationLocks     OperationLocksType
	InstanceID         string
	Device             string
	DeleteWithInstance string
	DeleteAutoSnapshot string
	EnableAutoSnapshot string
	CreationTime       string
	AttachedTime       string
	DetachedTime       string
}

// DiskSetType is exported
type DiskSetType struct {
	Disk DiskItemType
}

// DiskDeviceMapping is exported
type DiskDeviceMapping struct {
	SnapshotID string
	Size       string
	Device     string
}

// DiskMonitorDataType is exported
type DiskMonitorDataType struct {
	DiskID    string
	IOPSRead  int
	IOPSWrite int
	IOPSTotal int
	BPSRead   int
	BPSWrite  int
	BPSTotal  int
	TimeStamp string
}

//
// Image
//

// ImageType is exported
type ImageType struct {
	ImageID            string
	ImageVersion       string
	Architecture       string
	ImageName          string
	Description        string
	Size               int
	ImageOwnerAlias    string
	OSName             string
	DiskDeviceMappings DiskDeviceMapping
	ProductCode        string
	IsSubscribed       string
	Progress           string
	Status             string
	CreationTime       string
}

//
// Account
//

// AccountType is exported
type AccountType struct {
	AliyunID string
}

//
// ShareGroup
//

// ShareGroupType is exported
type ShareGroupType struct {
	Group string
}

//
// PublichIP
//

// AllocatePublicIPAddressResponse is exported
type AllocatePublicIPAddressResponse struct {
	EcsBaseResponse
	IPAddress string `json:"IpAddress"`
}
