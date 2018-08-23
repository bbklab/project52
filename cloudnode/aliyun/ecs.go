package aliyun

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

//
// Ecs Instance
//

// NewEcs create a new ecs instance with given settings
// note that Ecs is `Stopped` status after this call
func (sdk *SDK) NewEcs(req *CreateInstanceRequest) (string, error) {
	params := sdk.publicParameters()
	params.Set("Action", "CreateInstance") // must

	// verify the request params
	if err := req.Validate(); err != nil {
		return "", err
	}

	var (
		err      error
		regionID = req.RegionID
	)

	// create security group firstly
	sgid, err := sdk.NewSecurityGroup(&CreateSecurityGroupRequest{
		RegionID:          regionID,
		SecurityGroupName: req.InstanceName, // use instance name as security group name
		Description:       req.InstanceName, // use instance name as security group desc
	})
	if err != nil {
		return "", err
	}

	// if create ecs failed, clean up the newly created security group
	defer func() {
		if err != nil {
			log.Warnf("Ecs creation failed, clean up pre-created security group %s. [%v]", sgid, err)
			sdk.RemoveSecurityGroup(regionID, sgid)
		}
	}()

	// create security group authorization to allow all internet traffics
	err = sdk.AuthorizeSecurityGroup(&AuthorizeSecurityGroupRequest{
		SecurityGroupID: sgid,
		RegionID:        regionID,
		IPProtocol:      "all",
		PortRange:       "-1/-1",
		Policy:          "accept",
		NicType:         "internet",
		Priority:        "1",
		SourceCidrIP:    "0.0.0.0/0",
	})
	if err != nil {
		return "", err
	}

	// rewrite req with newly created SecurityGroup ID
	req.SecurityGroupID = sgid

	// append to request params
	req.AddToParameters(params)

	var resp *CreateInstanceResponse
	if err = sdk.apiCall(params, &resp); err != nil {
		return "", err
	}

	return resp.InstanceID, nil
}

// RemoveEcs remove the specified ecs instance
func (sdk *SDK) RemoveEcs(regionID, instanceID string) error {
	// firstly, insepct the target Ecs to obtain it's related security group ids
	info, err := sdk.InspectEcs(regionID, instanceID)
	if err != nil {
		return err
	}

	// remove ecs instance
	params := sdk.publicParameters()
	params.Set("Action", "DeleteInstance")
	params.Set("InstanceId", instanceID)

	var resp *EcsBaseResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return err
	}

	// remove related security gruops
	var (
		maxWait  = time.Second * 60
		interval = time.Second
		sgids    = info.SecurityGroupIDs.SecurityGroupID
	)
	for _, sgid := range sgids {
		// aliyun's inner status may delay few seconds, wait for a while to avoid error complains: DependencyViolation
		for goesby := int64(0); goesby <= int64(maxWait); goesby += int64(interval) {
			time.Sleep(interval)
			err = sdk.RemoveSecurityGroup(regionID, sgid)
			if err == nil {
				break
			}
			if strings.Contains(err.Error(), "DependencyViolation") {
				log.Warnf("aliyun ecs %s status not synced, retrying on security group removal ...", instanceID)
				continue
			}
			break
		}
	}

	return nil
}

// StartEcs start the specified ecs instance
// note that Ecs must be `Stopped` status, after this call, Ecs entering `Starting` status
func (sdk *SDK) StartEcs(id string) error {
	params := sdk.publicParameters()
	params.Set("Action", "StartInstance")
	params.Set("InstanceId", id)

	var resp *EcsBaseResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return err
	}

	return nil
}

// StopEcs stop the specified ecs instance
// note that Ecs must be `Running` status, after this call, Ecs entering `Stopping` status
func (sdk *SDK) StopEcs(id string) error {
	params := sdk.publicParameters()
	params.Set("Action", "StopInstance")
	params.Set("InstanceId", id)
	params.Set("ForceStop", "true")
	params.Set("ConfirmStop", "true")

	var resp *EcsBaseResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return err
	}

	return nil
}

// AssignEcsPublicIP assign a public ip to specified ecs instance
// note that Ecs must be `Running` or `Stopped` status
func (sdk *SDK) AssignEcsPublicIP(id string) (string, error) {
	params := sdk.publicParameters()
	params.Set("Action", "AllocatePublicIpAddress")
	params.Set("InstanceId", id)

	var resp *AllocatePublicIPAddressResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return "", err
	}

	return resp.IPAddress, nil
}

// InspectEcs show details of a given ecs instance
func (sdk *SDK) InspectEcs(regionID, instanceID string) (InstanceAttributesType, error) {
	params := sdk.publicParameters()
	params.Set("Action", "DescribeInstances")
	params.Set("RegionId", regionID)
	params.Set("InstanceIds", `["`+instanceID+`"]`)

	var resp *DescribeInstancesResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return InstanceAttributesType{}, err
	}

	if len(resp.Instances.Instance) == 0 {
		return InstanceAttributesType{}, errors.New("abnormal response, at least one instance should exists")
	}

	return resp.Instances.Instance[0], nil
}

// WaitEcs wait ecs instance status reached to expected status until maxWait timeout
func (sdk *SDK) WaitEcs(regionID, instanceID, expectStatus string, maxWait time.Duration) error {
	timeout := time.After(maxWait)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("wait ecs instance %s timeout in %s", expectStatus, maxWait)

		case <-ticker.C:
			info, err := sdk.InspectEcs(regionID, instanceID)
			if err != nil {
				return fmt.Errorf("WaitEcs.InspectEcs error: %v", err)
			}
			log.Debugf("Ecs Instance %s is %s ...", instanceID, info.Status)
			if info.Status == expectStatus {
				return nil
			}
		}
	}
}

// ListEcses show all of ecs instances for
// a specified region (regionID parameter set) or
// all of regions (regionID parameter empty)
func (sdk *SDK) ListEcses(regionID string) ([]InstanceAttributesType, error) {
	// obtain target regions firstly
	var queryRegions []string

	if regionID != "" {
		// use specified region id
		queryRegions = append(queryRegions, regionID)

	} else {
		// use all region ids
		regions, err := sdk.ListRegions()
		if err != nil {
			return nil, err
		}

		for _, reg := range regions {
			queryRegions = append(queryRegions, reg.RegionID)
		}
	}

	// query ecses list by each of region
	var (
		l   sync.Mutex
		ret []InstanceAttributesType
		wg  sync.WaitGroup
	)

	wg.Add(len(queryRegions))
	for _, regionID := range queryRegions {
		go func(regionID string) {
			defer wg.Done()
			params := sdk.publicParameters()
			params.Set("Action", "DescribeInstances")
			params.Set("RegionId", regionID)
			var resp *DescribeInstancesResponse
			if err := sdk.apiCall(params, &resp); err != nil {
				return
			}
			l.Lock()
			ret = append(ret, resp.Instances.Instance...)
			l.Unlock()
		}(regionID)
	}
	wg.Wait()

	return ret, nil
}

// ListRegions show all of regions aliyun supported
func (sdk *SDK) ListRegions() ([]RegionType, error) {
	params := sdk.publicParameters()
	params.Set("Action", "DescribeRegions")

	var resp *DescribeRegionsResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	return resp.Response.Regions, nil
}

// ListRegionZones show all of zones under a specified region.
// but the returned zone availble resources can't be used because the data is not exactly
// note: Deprecated
func (sdk *SDK) ListRegionZones(regionID string) ([]ZoneType, error) {
	params := sdk.publicParameters()
	params.Set("Action", "DescribeZones")
	params.Set("RegionId", regionID)

	var resp *DescribeRegionZonesResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	return resp.Zones.Zone, nil
}

// ListInstanceTypes show all of instance types aliyun supported
// support cpu / memory minimal/maximize filter parameters
func (sdk *SDK) ListInstanceTypes(minCPU, maxCPU, minMem, maxMem int) ([]InstanceTypeItemType, error) {
	params := sdk.publicParameters()
	params.Set("Action", "DescribeInstanceTypes")

	var resp *DescribeInstanceTypesResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	var ret []InstanceTypeItemType
	for _, typ := range resp.InstanceTypes.InstanceType {
		var cpus, mems = typ.CPUCoreCount, int(typ.MemorySize)
		if cpus <= maxCPU && cpus >= minCPU && mems <= maxMem && mems >= minMem {
			ret = append(ret, typ)
		}
	}

	return ret, nil
}

//
// Security Group
//

// NewSecurityGroup create a new security group with empty authorizations
func (sdk *SDK) NewSecurityGroup(req *CreateSecurityGroupRequest) (string, error) {
	params := sdk.publicParameters()
	params.Set("Action", "CreateSecurityGroup")

	// verify the request params
	if err := req.Validate(); err != nil {
		return "", err
	}

	// append to request params
	req.AddToParameters(params)

	var resp *CreateSecurityGroupResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return "", err
	}

	return resp.SecurityGroupID, nil
}

// RemoveSecurityGroup remove specified recurity group under given region
func (sdk *SDK) RemoveSecurityGroup(regionID, securityGroupID string) error {
	params := sdk.publicParameters()
	params.Set("Action", "DeleteSecurityGroup")
	params.Set("RegionId", regionID)
	params.Set("SecurityGroupId", securityGroupID)

	var resp *EcsBaseResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return err
	}

	return nil
}

// AuthorizeSecurityGroup create an authorization rule within specified security group
func (sdk *SDK) AuthorizeSecurityGroup(req *AuthorizeSecurityGroupRequest) error {
	params := sdk.publicParameters()
	params.Set("Action", "AuthorizeSecurityGroup")

	// verify the request params
	if err := req.Validate(); err != nil {
		return err
	}

	// append to request params
	req.AddToParameters(params)

	var resp *EcsBaseResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return err
	}

	return nil
}
