// Package qingcloud ...
// Most borrowed from:
//   github.com/yunify/qingcloud-sdk-go/service
//
package qingcloud

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bbklab/inf/pkg/orderparam"
	"github.com/bbklab/inf/pkg/ptype"
)

// BaseResponse is exported
//
type BaseResponse struct {
	Message string `json:"message"`
	Action  string `json:"action"`
	RetCode int    `json:"ret_code"`
}

//
// Zone
//

// DescribeZonesInput is exported
type DescribeZonesInput struct {
	Status []*string `json:"status"`
	Zones  []*string `json:"zones"`
}

// Validate is exported
func (v *DescribeZonesInput) Validate() error {
	return nil
}

// DescribeZonesOutput is exported
type DescribeZonesOutput struct {
	BaseResponse
	TotalCount *int    `json:"total_count"`
	ZoneSet    []*Zone `json:"zone_set"`
}

// Zone is exported
type Zone struct {
	Status *string `json:"status"` // active, faulty, defunct
	ZoneID *string `json:"zone_id"`
}

//
// Instance Types
//

// DescribeInstanceTypesOutput is exported
type DescribeInstanceTypesOutput struct {
	BaseResponse
	TotalCount      *int            `json:"total_count"`
	InstanceTypeSet []*InstanceType `json:"instance_type_set"`
}

// InstanceType is exported
type InstanceType struct {
	Description      *string `json:"description"`
	InstanceTypeID   *string `json:"instance_type_id"`
	InstanceTypeName *string `json:"instance_type_name"`
	MemoryCurrent    *int    `json:"memory_current"` // by MiB
	Status           *string `json:"status"`         // available, deprecated
	VCPUsCurrent     *int    `json:"vcpus_current"`
	ZoneID           *string `json:"zone_id"`
}

//
// Instance
//

// RunInstancesInput is exported
type RunInstancesInput struct {
	Count         *int      `json:"count"`         // must
	ImageID       *string   `json:"image_id"`      // must
	LoginMode     *string   `json:"login_mode"`    // must: keypair, passwd
	InstanceType  *string   `json:"instance_type"` // optional
	CPU           *int      `json:"cpu"`           // must if InstanceType empty: 1, 2, 4, 8, 16
	Memory        *int      `json:"memory"`        // must if InstanceType empty: 1024, 2048, 4096, 6144, 8192, 12288, 16384, 24576, 32768
	LoginPasswd   *string   `json:"login_passwd"`  // must if LoginMode==passwd
	LoginKeyPair  *string   `json:"login_keypair"` // must if LoginMode==keypair
	Hostname      *string   `json:"hostname"`
	InstanceClass *int      `json:"instance_class"` // 0, 1
	InstanceName  *string   `json:"instance_name"`
	NeedNewSID    *int      `json:"need_newsid"`   // 0, 1
	NeedUserdata  *int      `json:"need_userdata"` // 0, 1
	SecurityGroup *string   `json:"security_group"`
	UIType        *string   `json:"ui_type"`
	UserdataFile  *string   `json:"userdata_file"`
	UserdataPath  *string   `json:"userdata_path"`
	UserdataType  *string   `json:"userdata_type"` // plain, exec, tar
	UserdataValue *string   `json:"userdata_value"`
	Volumes       []*string `json:"volumes"`
	VxNets        []*string `json:"vxnets"`
}

// RunInstancesOutput is exported
type RunInstancesOutput struct {
	BaseResponse
	Instances []string `json:"instances"`
	JobID     string   `json:"job_id"`
}

// AddToParameters should be called after Validate verify checking
func (v *RunInstancesInput) AddToParameters(params *orderparam.Params) {
	params.Set("count", strconv.Itoa(ptype.IntV(v.Count)))
	params.Set("image_id", ptype.StringV(v.ImageID))
	params.Set("login_mode", ptype.StringV(v.LoginMode))
	params.SetIgnoreNull("instance_type", ptype.StringV(v.InstanceType))
	if ptype.StringV(v.InstanceType) == "" {
		params.Set("cpu", strconv.Itoa(ptype.IntV(v.CPU)))
		params.Set("memory", strconv.Itoa(ptype.IntV(v.Memory)))
	}
	if ptype.StringV(v.LoginMode) == "passwd" {
		params.Set("login_passwd", ptype.StringV(v.LoginPasswd))
	} else {
		params.Set("login_keypair", ptype.StringV(v.LoginKeyPair))
	}
	params.SetIgnoreNull("hostname", ptype.StringV(v.Hostname))
	params.SetIgnoreNull("instance_class", strconv.Itoa(ptype.IntV(v.InstanceClass)))
	params.SetIgnoreNull("instance_name", ptype.StringV(v.InstanceName))
	params.SetIgnoreNull("need_newsid", strconv.Itoa(ptype.IntV(v.NeedNewSID)))
	params.SetIgnoreNull("need_userdata", strconv.Itoa(ptype.IntV(v.NeedUserdata)))
	params.SetIgnoreNull("security_group", ptype.StringV(v.SecurityGroup))
	params.SetIgnoreNull("ui_type", ptype.StringV(v.UIType))
	params.SetIgnoreNull("userdata_file", ptype.StringV(v.UserdataFile))
	params.SetIgnoreNull("userdata_path", ptype.StringV(v.UserdataPath))
	params.SetIgnoreNull("userdata_type", ptype.StringV(v.UserdataType))
	params.SetIgnoreNull("userdata_value", ptype.StringV(v.UserdataValue))

	var idx int
	for _, vxnet := range ptype.StringSliceV(v.VxNets) {
		idx++
		params.Set(fmt.Sprintf("vxnets.%d", idx), vxnet)
	}
}

// Validate is exported
func (v *RunInstancesInput) Validate() error {
	if v.Count == nil {
		return ParameterRequiredError{
			ParameterName: "Count",
			ParentName:    "RunInstancesInput",
		}
	}

	if *v.Count <= 0 {
		return errors.New("parameter Count must be positive")
	}

	if v.ImageID == nil {
		return ParameterRequiredError{
			ParameterName: "ImageID",
			ParentName:    "RunInstancesInput",
		}
	}

	if v.LoginMode == nil {
		return ParameterRequiredError{
			ParameterName: "LoginMode",
			ParentName:    "RunInstancesInput",
		}
	}

	loginModeValidValues := []string{"keypair", "passwd"}
	loginModeParameterValue := fmt.Sprint(*v.LoginMode)
	loginModeIsValid := false
	for _, value := range loginModeValidValues {
		if value == loginModeParameterValue {
			loginModeIsValid = true
		}
	}
	if !loginModeIsValid {
		return ParameterValueNotAllowedError{
			ParameterName:  "LoginMode",
			ParameterValue: loginModeParameterValue,
			AllowedValues:  loginModeValidValues,
		}
	}

	if v.CPU != nil {
		cpuValidValues := []string{"1", "2", "4", "8", "16"}
		cpuParameterValue := fmt.Sprint(*v.CPU)

		cpuIsValid := false
		for _, value := range cpuValidValues {
			if value == cpuParameterValue {
				cpuIsValid = true
			}
		}

		if !cpuIsValid {
			return ParameterValueNotAllowedError{
				ParameterName:  "CPU",
				ParameterValue: cpuParameterValue,
				AllowedValues:  cpuValidValues,
			}
		}
	}

	if v.InstanceClass != nil {
		instanceClassValidValues := []string{"0", "1"}
		instanceClassParameterValue := fmt.Sprint(*v.InstanceClass)

		instanceClassIsValid := false
		for _, value := range instanceClassValidValues {
			if value == instanceClassParameterValue {
				instanceClassIsValid = true
			}
		}

		if !instanceClassIsValid {
			return ParameterValueNotAllowedError{
				ParameterName:  "InstanceClass",
				ParameterValue: instanceClassParameterValue,
				AllowedValues:  instanceClassValidValues,
			}
		}
	}

	if v.Memory != nil {
		memoryValidValues := []string{"1024", "2048", "4096", "6144", "8192", "12288", "16384", "24576", "32768"}
		memoryParameterValue := fmt.Sprint(*v.Memory)

		memoryIsValid := false
		for _, value := range memoryValidValues {
			if value == memoryParameterValue {
				memoryIsValid = true
			}
		}

		if !memoryIsValid {
			return ParameterValueNotAllowedError{
				ParameterName:  "Memory",
				ParameterValue: memoryParameterValue,
				AllowedValues:  memoryValidValues,
			}
		}
	}

	if v.NeedNewSID != nil {
		needNewSIDValidValues := []string{"0", "1"}
		needNewSIDParameterValue := fmt.Sprint(*v.NeedNewSID)

		needNewSIDIsValid := false
		for _, value := range needNewSIDValidValues {
			if value == needNewSIDParameterValue {
				needNewSIDIsValid = true
			}
		}

		if !needNewSIDIsValid {
			return ParameterValueNotAllowedError{
				ParameterName:  "NeedNewSID",
				ParameterValue: needNewSIDParameterValue,
				AllowedValues:  needNewSIDValidValues,
			}
		}
	}

	if v.NeedUserdata != nil {
		needUserdataValidValues := []string{"0", "1"}
		needUserdataParameterValue := fmt.Sprint(*v.NeedUserdata)

		needUserdataIsValid := false
		for _, value := range needUserdataValidValues {
			if value == needUserdataParameterValue {
				needUserdataIsValid = true
			}
		}

		if !needUserdataIsValid {
			return ParameterValueNotAllowedError{
				ParameterName:  "NeedUserdata",
				ParameterValue: needUserdataParameterValue,
				AllowedValues:  needUserdataValidValues,
			}
		}
	}

	if v.UserdataType != nil {
		userdataTypeValidValues := []string{"plain", "exec", "tar"}
		userdataTypeParameterValue := fmt.Sprint(*v.UserdataType)

		userdataTypeIsValid := false
		for _, value := range userdataTypeValidValues {
			if value == userdataTypeParameterValue {
				userdataTypeIsValid = true
			}
		}

		if !userdataTypeIsValid {
			return ParameterValueNotAllowedError{
				ParameterName:  "UserdataType",
				ParameterValue: userdataTypeParameterValue,
				AllowedValues:  userdataTypeValidValues,
			}
		}
	}

	return nil
}

// DescribeInstancesOutput is exported
type DescribeInstancesOutput struct {
	BaseResponse
	InstanceSet []*Instance `json:"instance_set"`
	TotalCount  *int        `json:"total_count"`
}

// InstanceWrapper is exported
type InstanceWrapper struct {
	*Instance
	Zone string `json:"zone"`
}

// Instance is exported
type Instance struct {
	AlarmStatus      *string         `json:"alarm_status"`
	CPUTopology      *string         `json:"cpu_topology"`
	CreateTime       *time.Time      `json:"create_time"`
	Description      *string         `json:"description"`
	Device           *string         `json:"device"`
	EIP              *EIP            `json:"eip"`
	GraphicsPasswd   *string         `json:"graphics_passwd"`
	GraphicsProtocol *string         `json:"graphics_protocol"`
	Image            Image           `json:"image"`
	ImageID          *string         `json:"image_id"`
	InstanceClass    *int            `json:"instance_class"`
	InstanceID       *string         `json:"instance_id"`
	InstanceName     *string         `json:"instance_name"`
	InstanceType     *string         `json:"instance_type"`
	KeyPairIDs       []string        `json:"keypair_ids"`
	MemoryCurrent    *int            `json:"memory_current"`
	PrivateIP        *string         `json:"private_ip"`
	SecurityGroup    SecurityGroup   `json:"security_group"`
	Status           *string         `json:"status"` // pending, running, stopped, suspended, terminated, ceased
	StatusTime       time.Time       `json:"status_time"`
	SubCode          *int            `json:"sub_code"`
	Tags             []Tag           `json:"tags"`
	TransitionStatus *string         `json:"transition_status"` // creating, starting, stopping, restarting, suspending, resuming, terminating, recovering, resetting
	VCPUsCurrent     *int            `json:"vcpus_current"`
	VolumeIDs        []*string       `json:"volume_ids"`
	Volumes          []Volume        `json:"volumes"`
	VxNets           []InstanceVxNet `json:"vxnets"`
}

// StartInstancesOutput is exported
type StartInstancesOutput struct {
	BaseResponse
	JobID *string `json:"job_id"`
}

// StopInstancesOutput is exported
type StopInstancesOutput struct {
	BaseResponse
	JobID *string `json:"job_id"`
}

// TerminateInstancesOutput is exported
type TerminateInstancesOutput struct {
	BaseResponse
	JobID *string `json:"job_id"`
}

//
// SecurityGroup
//

// SecurityGroup is exported
type SecurityGroup struct {
	CreateTime        *time.Time  `json:"create_time"`
	Description       *string     `json:"description"`
	IsApplied         *int        `json:"is_applied"`
	IsDefault         *int        `json:"is_default"`
	Resources         []*Resource `json:"resources"`
	SecurityGroupID   *string     `json:"security_group_id"`
	SecurityGroupName *string     `json:"security_group_name"`
	Tags              []*Tag      `json:"tags"`
}

//
// Volume
//

// Volume is exported
type Volume struct {
	CreateTime         *time.Time  `json:"create_time"`
	Description        *string     `json:"description"`
	Device             *string     `json:"device"`
	Instance           *Instance   `json:"instance"`
	Instances          []*Instance `json:"instances"`
	LatestSnapshotTime *time.Time  `json:"latest_snapshot_time"`
	Owner              *string     `json:"owner"`
	PlaceGroupID       *string     `json:"place_group_id"`
	Size               *int        `json:"size"`
	// Status's available values: pending, available, in-use, suspended, deleted, ceased
	Status     *string    `json:"status"`
	StatusTime *time.Time `json:"status_time"`
	SubCode    *int       `json:"sub_code"`
	Tags       []*Tag     `json:"tags"`
	// TransitionStatus's available values: creating, attaching, detaching, suspending, resuming, deleting, recovering
	TransitionStatus *string `json:"transition_status"`
	VolumeID         *string `json:"volume_id"`
	VolumeName       *string `json:"volume_name"`
	// VolumeType's available values: 0, 1, 2, 3
	VolumeType *int `json:"volume_type"`
}

//
// Tag
//

// Tag is exported
type Tag struct {
	Color             *string              `json:"color"`
	CreateTime        *time.Time           `json:"create_time"`
	Description       *string              `json:"description"`
	Owner             *string              `json:"owner"`
	ResourceCount     *int                 `json:"resource_count"`
	ResourceTagPairs  []*ResourceTagPair   `json:"resource_tag_pairs"`
	ResourceTypeCount []*ResourceTypeCount `json:"resource_type_count"`
	TagID             *string              `json:"tag_id"`
	TagKey            *string              `json:"tag_key"`
	TagName           *string              `json:"tag_name"`
}

//
// Image
//

// Image is exported
type Image struct {
	AppBillingID  *string    `json:"app_billing_id"`
	Architecture  *string    `json:"architecture"`
	BillingID     *string    `json:"billing_id"`
	CreateTime    *time.Time `json:"create_time"`
	DefaultPasswd *string    `json:"default_passwd"`
	DefaultUser   *string    `json:"default_user"`
	Description   *string    `json:"description"`
	FResetpwd     *int       `json:"f_resetpwd"`
	Feature       *int       `json:"feature"`
	Features      *int       `json:"features"`
	Hypervisor    *string    `json:"hypervisor"`
	ImageID       *string    `json:"image_id"`
	ImageName     *string    `json:"image_name"`
	InstanceIDs   []*string  `json:"instance_ids"`
	OSFamily      *string    `json:"os_family"`
	Owner         *string    `json:"owner"`
	// Platform's available values: linux, windows
	Platform *string `json:"platform"`
	// ProcessorType's available values: 64bit, 32bit
	ProcessorType *string `json:"processor_type"`
	// Provider's available values: system, self
	Provider        *string `json:"provider"`
	RecommendedType *string `json:"recommended_type"`
	RootID          *string `json:"root_id"`
	Size            *int    `json:"size"`
	// Status's available values: pending, available, deprecated, suspended, deleted, ceased
	Status     *string    `json:"status"`
	StatusTime *time.Time `json:"status_time"`
	SubCode    *int       `json:"sub_code"`
	// TransitionStatus's available values: creating, suspending, resuming, deleting, recovering
	TransitionStatus *string `json:"transition_status"`
	UIType           *string `json:"ui_type"`
	// Visibility's available values: public, private
	Visibility *string `json:"visibility"`
}

//
// VxNet
//

// VxNet is exported
type VxNet struct {
	AvailableIPCount *int       `json:"available_ip_count"`
	CreateTime       *time.Time `json:"create_time"`
	Description      *string    `json:"description"`
	InstanceIDs      []*string  `json:"instance_ids"`
	Owner            *string    `json:"owner"`
	Router           *Router    `json:"router"`
	Tags             []*Tag     `json:"tags"`
	VpcRouterID      *string    `json:"vpc_router_id"`
	VxNetID          *string    `json:"vxnet_id"`
	VxNetName        *string    `json:"vxnet_name"`
	// VxNetType's available values: 0, 1
	VxNetType *int `json:"vxnet_type"`
}

// InstanceVxNet is exported
type InstanceVxNet struct {
	NICID     *string `json:"nic_id"`
	PrivateIP *string `json:"private_ip"`
	Role      *int    `json:"role"`
	VxNetID   *string `json:"vxnet_id"`
	VxNetName *string `json:"vxnet_name"`
	// VxNetType's available values: 0, 1
	VxNetType *int `json:"vxnet_type"`
}

//
// Nic
//

// DescribeNicsOutput is exported
type DescribeNicsOutput struct {
	BaseResponse
	NICSet     []*NIC `json:"nic_set"`
	TotalCount *int   `json:"total_count"`
}

// NIC is exported
type NIC struct {
	CreateTime    *time.Time `json:"create_time"`
	InstanceID    *string    `json:"instance_id"`
	NICID         *string    `json:"nic_id"`
	NICName       *string    `json:"nic_name"`
	Owner         *string    `json:"owner"`
	PrivateIP     *string    `json:"private_ip"`
	Role          *int       `json:"role"`
	RootUserID    *string    `json:"root_user_id"`
	SecurityGroup *string    `json:"security_group"`
	Sequence      *int       `json:"sequence"`
	Status        *string    `json:"status"` // avaliable, in-use
	StatusTime    *time.Time `json:"status_time"`
	Tags          []*Tag     `json:"tags"`
	VxNetID       *string    `json:"vxnet_id"`
}

// CreateNicsOutput is exported
type CreateNicsOutput struct {
	BaseResponse
	Nics []*NICIP `json:"nics"`
}

// NICIP is exported
type NICIP struct {
	NICID     *string `json:"nic_id"`
	PrivateIP *string `json:"private_ip"`
}

//
// Router
//

// Router is exported
type Router struct {
	CreateTime  *time.Time `json:"create_time"`
	Description *string    `json:"description"`
	DYNIPEnd    *string    `json:"dyn_ip_end"`
	DYNIPStart  *string    `json:"dyn_ip_start"`
	EIP         *EIP       `json:"eip"`
	IPNetwork   *string    `json:"ip_network"`
	// IsApplied's available values: 0, 1
	IsApplied  *int    `json:"is_applied"`
	ManagerIP  *string `json:"manager_ip"`
	Mode       *int    `json:"mode"`
	PrivateIP  *string `json:"private_ip"`
	RouterID   *string `json:"router_id"`
	RouterName *string `json:"router_name"`
	// RouterType's available values: 1
	RouterType      *int    `json:"router_type"`
	SecurityGroupID *string `json:"security_group_id"`
	// Status's available values: pending, active, poweroffed, suspended, deleted, ceased
	Status     *string    `json:"status"`
	StatusTime *time.Time `json:"status_time"`
	Tags       []*Tag     `json:"tags"`
	// TransitionStatus's available values: creating, updating, suspending, resuming, poweroffing, poweroning, deleting
	TransitionStatus *string  `json:"transition_status"`
	VxNets           []*VxNet `json:"vxnets"`
}

//
// Resource
//

// Resource is exported
type Resource struct {
	ResourceID   *string `json:"resource_id"`
	ResourceName *string `json:"resource_name"`
	ResourceType *string `json:"resource_type"`
}

// Validate is exported
func (v *Resource) Validate() error {
	return nil
}

// ResourceTagPair is exported
type ResourceTagPair struct {
	ResourceID   *string    `json:"resource_id"`
	ResourceType *string    `json:"resource_type"`
	Status       *string    `json:"status"`
	StatusTime   *time.Time `json:"status_time"`
	TagID        *string    `json:"tag_id"`
}

// Validate is exported
func (v *ResourceTagPair) Validate() error {
	return nil
}

// ResourceTypeCount is exported
type ResourceTypeCount struct {
	Count        *int    `json:"count"`
	ResourceType *string `json:"resource_type"`
}

//
// EIP
//

// DescribeEIPsOutput is exported
type DescribeEIPsOutput struct {
	BaseResponse
	EIPSet     []*EIP `json:"eip_set"`
	TotalCount *int   `json:"total_count"`
}

// EIP is exported
type EIP struct {
	AlarmStatus   *string `json:"alarm_status"`
	AssociateMode *int    `json:"associate_mode"`
	Bandwidth     *int    `json:"bandwidth"`
	// BillingMode's available values: bandwidth, traffic
	BillingMode *string      `json:"billing_mode"`
	CreateTime  *time.Time   `json:"create_time"`
	Description *string      `json:"description"`
	EIPAddr     *string      `json:"eip_addr"`
	EIPGroup    *EIPGroup    `json:"eip_group"`
	EIPID       *string      `json:"eip_id"`
	EIPName     *string      `json:"eip_name"`
	ICPCodes    *string      `json:"icp_codes"`
	NeedICP     *int         `json:"need_icp"`
	Resource    *EIPResource `json:"resource"`
	// Status's available values: pending, available, associated, suspended, released, ceased
	Status     *string    `json:"status"`
	StatusTime *time.Time `json:"status_time"`
	SubCode    *int       `json:"sub_code"`
	Tags       []*Tag     `json:"tags"`
	// TransitionStatus's available values: associating, dissociating, suspending, resuming, releasing
	TransitionStatus *string `json:"transition_status"`
}

// EIPGroup is exported
type EIPGroup struct {
	EIPGroupID   *string `json:"eip_group_id"`
	EIPGroupName *string `json:"eip_group_name"`
}

// EIPResource is exported
type EIPResource struct {
	ResourceID   *string `json:"resource_id"`
	ResourceName *string `json:"resource_name"`
	ResourceType *string `json:"resource_type"`
}

// AllocateEIPsOutput is exported
type AllocateEIPsOutput struct {
	BaseResponse
	EIPs []*string `json:"eips"`
}

// ReleaseEIPsOutput is exported
type ReleaseEIPsOutput struct {
	BaseResponse
	JobID *string `json:"job_id"`
}

// AssociateEIPOutput is exported
type AssociateEIPOutput struct {
	BaseResponse
	JobID *string `json:"job_id"`
}

// DissociateEIPsOutput is exported
type DissociateEIPsOutput struct {
	BaseResponse
	JobID *string `json:"job_id"`
}

//
// Job
//

// DescribeJobsOutput is exported
type DescribeJobsOutput struct {
	BaseResponse
	JobSet     []*Job `json:"job_set"`
	TotalCount *int   `json:"total_count"`
}

// Job is exported
type Job struct {
	CreateTime  *time.Time `json:"create_time"`
	JobAction   *string    `json:"job_action"`
	JobID       *string    `json:"job_id"`
	Owner       *string    `json:"owner"`
	ResourceIDs *string    `json:"resource_ids"`
	Status      *string    `json:"status"` //  pending, working, failed, successful, done with failure
	StatusTime  *time.Time `json:"status_time"`
}

//
// Error Types
//

// ParameterRequiredError indicates that the required parameter is missing.
type ParameterRequiredError struct {
	ParameterName string
	ParentName    string
}

// Error returns the description of ParameterRequiredError.
func (e ParameterRequiredError) Error() string {
	return fmt.Sprintf(`"%s" is required in "%s"`, e.ParameterName, e.ParentName)
}

// ParameterValueNotAllowedError indicates that the parameter value is not allowed.
type ParameterValueNotAllowedError struct {
	ParameterName  string
	ParameterValue string
	AllowedValues  []string
}

// Error returns the description of ParameterValueNotAllowedError.
func (e ParameterValueNotAllowedError) Error() string {
	allowedValues := []string{}
	for _, value := range e.AllowedValues {
		allowedValues = append(allowedValues, "\""+value+"\"")
	}
	return fmt.Sprintf(
		`"%s" value "%s" is not allowed, should be one of %s`,
		e.ParameterName,
		e.ParameterValue,
		strings.Join(allowedValues, ", "))
}
