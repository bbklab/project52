package linode

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
)

//
// Ecs Instance
//

// NewEcs create a new ecs instance with given settings
func (sdk *SDK) NewEcs(req *CreateInstancesInput) (int, error) {
	var resp Instance
	err := sdk.apiCall("POST", "/linode/instances", req, &resp)
	if err != nil {
		return -1, err
	}
	return resp.ID, err
}

// ListEcses show all of ecs instances
func (sdk *SDK) ListEcses() ([]*Instance, error) {
	var resp DescribeInstancesOutput
	err := sdk.apiCall("GET", "/linode/instances", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// InspectEcs show details of a given ecs instance
func (sdk *SDK) InspectEcs(id int) (*Instance, error) {
	var resp *Instance
	err := sdk.apiCall("GET", fmt.Sprintf("/linode/instances/%d", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RemoveEcs remove the specified ecs instance
func (sdk *SDK) RemoveEcs(id int) error {
	return sdk.apiCall("DELETE", fmt.Sprintf("/linode/instances/%d", id), nil, nil)
}

// WaitEcs wait ecs instance status reached to expected status until maxWait timeout
func (sdk *SDK) WaitEcs(id int, expectStatus string, maxWait time.Duration) error {
	timeout := time.After(maxWait)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("wait ecs instance %d -> %s timeout in %s", id, expectStatus, maxWait)

		case <-ticker.C:
			info, err := sdk.InspectEcs(id)
			if err != nil {
				return fmt.Errorf("WaitEcs.InspectEcs %d error: %v", id, err)
			}
			logrus.Printf("linode ecs instance %d is %s ...", id, info.Status)
			if info.Status == expectStatus {
				return nil
			}
		}
	}
}

// ListRegions show all of regions linode supported
// note: public api, do NOT require auth
func (sdk *SDK) ListRegions() ([]*Region, error) {
	var resp DescribeRegionsOutput
	err := sdk.apiCall("GET", "/regions", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Data, err
}

// ListInstanceTypes show all of instance types under given zone
// support cpu / memory minimal/maximize filter parameters
// note: public api, do NOT require auth
func (sdk *SDK) ListInstanceTypes(minCPU, maxCPU, minMem, maxMem int) ([]*InstanceType, error) {
	var resp DescribeTypesOutput
	err := sdk.apiCall("GET", "/linode/types", nil, &resp)
	if err != nil {
		return nil, err
	}

	var ret []*InstanceType
	for _, typ := range resp.Data {
		var cpus, mems = typ.VCpus, typ.Memory
		if cpus <= maxCPU && cpus >= minCPU && mems <= maxMem && mems >= minMem {
			ret = append(ret, typ)
		}
	}

	return ret, nil
}
