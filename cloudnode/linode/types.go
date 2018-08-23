package linode

import "errors"

// BaseResponse is exported
//
type BaseResponse struct {
	Results int `json:"results"`
	Pages   int `json:"pages"`
	Page    int `json:"page"`
}

// ErrorResponse is exported
type ErrorResponse struct {
	Errors []*ErrResponse `json:"errors"`
}

// ErrResponse is exported
type ErrResponse struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

//
// Region
//

// DescribeRegionsOutput is exported
type DescribeRegionsOutput struct {
	BaseResponse
	Data []*Region `json:"data"`
}

// Region is exported
type Region struct {
	ID      string `json:"id"`
	Country string `json:"country"`
}

//
// Instance Types
//

// DescribeTypesOutput is exported
type DescribeTypesOutput struct {
	BaseResponse
	Data []*InstanceType `json:"data"`
}

// InstanceType is exported
type InstanceType struct {
	ID         string `json:"id"`
	Memory     int    `json:"memory"` // by MiB
	VCpus      int    `json:"vcpus"`
	Disk       int    `json:"disk"`        // by MiB
	Transfer   int    `json:"transfer"`    // outbound transfer in MB
	NetworkOut int    `json:"network_out"` // Mbits outbound bandwidth
	Label      string `json:"label"`
	Class      string `json:"class"`
	// Price      Price  `json:"price"`
	// Addons     Addons `json:"addons"`
}

//
// Instance
//

// CreateInstancesInput is exported
type CreateInstancesInput struct {
	Region       string `json:"region"` // must
	Type         string `json:"type"`   // must
	Label        string `json:"label"`  // must be uniq,  length: [3-32]
	Distribution string `json:"distribution"`
	RootPass     string `json:"root_pass"`
	Booted       bool   `json:"booted"`
}

// Validate is exported
func (req *CreateInstancesInput) Validate() error {
	if req.Region == "" {
		return errors.New("region required")
	}
	if req.Type == "" {
		return errors.New("instance type required")
	}
	if n := len(req.Label); n < 3 || n > 32 {
		return errors.New("label length must between [3-32]")
	}
	if req.Distribution == "" {
		return errors.New("distribution required")
	}
	if req.RootPass == "" {
		return errors.New("root password required")
	}
	return nil
}

// DescribeInstancesOutput is exported
type DescribeInstancesOutput struct {
	BaseResponse
	Data []*Instance `json:"data"`
}

// Instance is exported
type Instance struct {
	ID           int      `json:"id"`
	Region       string   `json:"region"`
	Image        string   `json:"image"`
	Group        string   `json:"group"`
	Distribution string   `json:"distribution"`
	IPv4         []string `json:"ipv4"`
	IPv6         string   `json:"ipv6"`
	Label        string   `json:"label"`
	Type         string   `json:"type"`       // instance type
	Status       string   `json:"status"`     // offline,booting,running,shutting_down,rebooting,provisioning,deleting,migrating
	Hypervisor   string   `json:"hypervisor"` // kvm, xen
	Created      string   `json:"created"`
	Updated      string   `json:"updated"`
	Specs        *Spec    `json:"specs"`
	// Alerts       *LinodeAlert
	// Backups      *LinodeBackup
	// Snapshot     *LinodeBackup
}

// Spec is exported
type Spec struct {
	Disk     int `json:"disk"`
	Memory   int `json:"memory"`
	Vcpus    int `json:"vcpus"`
	Transfer int `json:"transfer"`
}
