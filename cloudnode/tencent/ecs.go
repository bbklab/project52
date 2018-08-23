package tencent

import (
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"

	"github.com/bbklab/inf/pkg/ptype"
)

//
// Ecs Instance
//

// NewEcs create a new ecs instance with given settings
func (sdk *SDK) NewEcs(region string, req *cvm.RunInstancesRequest) (string, error) {
	params := sdk.publicParameters()
	params.Set("Action", "RunInstances") // must
	params.Set("Region", region)         // must

	// verify the request params
	if err := validateRunInstanceReq(req); err != nil {
		return "", err
	}

	// append to request params
	mergeRunInstanceReq(params, req)

	var resp *cvm.RunInstancesResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return "", err
	}

	if len(resp.Response.InstanceIdSet) == 0 {
		return "", errors.New("NewEcs() got abnormal response, at least one instance should exists")
	}
	return ptype.StringSliceV(resp.Response.InstanceIdSet)[0], nil
}

// RemoveEcs remove the specified ecs instance
func (sdk *SDK) RemoveEcs(region, ecsID string) error {
	params := sdk.publicParameters()
	params.Set("Action", "TerminateInstances") // must
	params.Set("Region", region)               // must
	params.Set("InstanceIds.0", ecsID)         // given ecs id

	var resp = new(cvm.TerminateInstancesResponse)
	if err := sdk.apiCall(params, &resp); err != nil {
		return err
	}

	// wait until ecs indeed removed
	timeout := time.After(time.Second * 120)
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("Remove Ecs Instance %s timeout", ecsID)
		case <-ticker.C:
			info, err := sdk.InspectEcs(region, ecsID)
			if err != nil {
				goto END
			}
			// still exists, retry
			status := ptype.StringV(info.InstanceState)
			log.Debugf("Ecs Instance %s is %s ...", ecsID, status)
		}
	}

END:
	return nil
}

// InspectEcs show details of a given ecs instance
func (sdk *SDK) InspectEcs(region, ecsID string) (*cvm.Instance, error) {
	params := sdk.publicParameters()
	params.Set("Action", "DescribeInstances")
	params.Set("Region", region)       // must
	params.Set("InstanceIds.0", ecsID) // given ecs id

	var resp = new(cvm.DescribeInstancesResponse)
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	if len(resp.Response.InstanceSet) == 0 {
		return nil, errors.New("InspectEcs() got abnormal response, at least one instance should exists")
	}

	return resp.Response.InstanceSet[0], nil
}

// WaitEcs wait ecs instance status reached to expected status until maxWait timeout
// expectStatus could be:
//  - PENDING
//  - LAUNCH_FAILED
//  - RUNNING
//  - STOPPED
//  - STARTING
//  - STOPPING
//  - REBOOTING
//  - SHUTDOWN
//  - TERMINATING
func (sdk *SDK) WaitEcs(region, ecsID, expectStatus string, maxWait time.Duration) error {
	timeout := time.After(maxWait)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("wait ecs instance %s timeout in %s", expectStatus, maxWait)

		case <-ticker.C:
			info, err := sdk.InspectEcs(region, ecsID)
			if err != nil {
				return fmt.Errorf("WaitEcs.InspectEcs error: %v", err)
			}
			status := ptype.StringV(info.InstanceState)
			log.Printf("tencent ecs instance %s is %s ...", ecsID, status)
			if status == expectStatus {
				return nil
			}
		}
	}
}

// ListEcses show all of ecs instances for
// a specified region (regionID parameter set) or
// all of regions (regionID parameter empty)
func (sdk *SDK) ListEcses(region string, lbs map[string]string) ([]*cvm.Instance, error) {
	// obtain target regions firstly
	var queryRegions []string

	if region != "" {
		// use specified region id
		queryRegions = append(queryRegions, region)

	} else {
		// use all region ids
		regions, err := sdk.ListRegions()
		if err != nil {
			return nil, err
		}
		for _, reg := range regions {
			regid := ptype.StringV(reg.Region)
			switch regid {
			case "ap-shanghai-fsi", "ap-shenzhen-fsi": // 华东地区(上海金融), 华南地区(深圳金融)
				continue
			}
			queryRegions = append(queryRegions, regid)
		}
	}

	// construct label filters
	var filters = make([][2]string, 0)
	for key, val := range lbs {
		filters = append(filters, [2]string{key, val})
	}

	// query ecses list by each of region
	var (
		l   sync.Mutex
		ret []*cvm.Instance
		wg  sync.WaitGroup
	)

	wg.Add(len(queryRegions))
	for _, reg := range queryRegions {
		go func(reg string) {
			defer wg.Done()
			params := sdk.publicParameters()
			params.Set("Action", "DescribeInstances")
			params.Set("Region", reg)
			for idx, kv := range filters {
				params.Set(fmt.Sprintf("Filters.%d.Name", idx), fmt.Sprintf("tag:%s", kv[0]))
				params.Set(fmt.Sprintf("Filters.%d.Values.0", idx), kv[1])
			}
			var resp = new(cvm.DescribeInstancesResponse)
			if err := sdk.apiCall(params, &resp); err != nil {
				return
			}
			l.Lock()
			ret = append(ret, resp.Response.InstanceSet...)
			l.Unlock()
		}(reg)
	}
	wg.Wait()

	return ret, nil
}

// ListRegions show all of regions tencent supported
func (sdk *SDK) ListRegions() ([]*cvm.RegionInfo, error) {
	params := sdk.publicParameters()
	params.Set("Action", "DescribeRegions")

	var resp *cvm.DescribeRegionsResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	return resp.Response.RegionSet, nil
}

// ListZones show all of zones under given region
func (sdk *SDK) ListZones(region string) ([]*cvm.ZoneInfo, error) {
	params := sdk.publicParameters()
	params.Set("Action", "DescribeZones")
	params.Set("Region", region)

	var resp *cvm.DescribeZonesResponse
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	return resp.Response.ZoneSet, nil
}

// PickupZone pick up the first avaliable zone under a specified region
func (sdk *SDK) PickupZone(region string) string {
	defaultZone := fmt.Sprintf("%s-1", region)

	zones, err := sdk.ListZones(region)
	if err != nil {
		return defaultZone
	}

	for _, zone := range zones {
		if ptype.StringV(zone.ZoneState) == "AVAILABLE" {
			return ptype.StringV(zone.Zone)
		}
	}

	return defaultZone
}

// ListInstanceTypes show all of instance types tencent supported
// support cpu / memory minimal/maximize filter parameters
func (sdk *SDK) ListInstanceTypes(minCPU, maxCPU, minMem, maxMem int) ([]*cvm.InstanceTypeConfig, error) {
	regs, err := sdk.ListRegions()
	if err != nil {
		return nil, err
	}

	var (
		wg   sync.WaitGroup
		l    sync.Mutex // protect followins two
		seen = make(map[string]bool)
		ret  []*cvm.InstanceTypeConfig
	)

	// list all regions instance types by concurrency
	wg.Add(len(regs))
	for _, reg := range regs {
		go func(reg *cvm.RegionInfo) {
			defer wg.Done()

			var (
				regid   = ptype.StringV(reg.Region)
				regname = ptype.StringV(reg.RegionName)
			)

			// followings two tencent regions are always can't be listed
			switch regid {
			case "ap-shanghai-fsi", "ap-shenzhen-fsi": // 华东地区(上海金融), 华南地区(深圳金融)
				return
			}

			params := sdk.publicParameters()
			params.Set("Action", "DescribeInstanceTypeConfigs")
			params.Set("Region", regid)

			var resp = new(cvm.DescribeInstanceTypeConfigsResponse)
			if err := sdk.apiCall(params, &resp); err != nil {
				log.Errorf("ListInstanceTypes() on tencent region %s:%s error: %v", regid, regname, err)
				return
			}

			for _, typ := range resp.Response.InstanceTypeConfigSet {
				id := ptype.StringV(typ.InstanceType)
				l.Lock()
				if _, ok := seen[id]; !ok {
					seen[id] = true
					ret = append(ret, typ)
				}
				l.Unlock()
			}
		}(reg)
	}
	wg.Wait()

	filtered := ret[:0]
	for _, typ := range ret {
		var cpus, mems = ptype.Int64V(typ.CPU), ptype.Int64V(typ.Memory)
		if cpus <= int64(maxCPU) && cpus >= int64(minCPU) && mems <= int64(maxMem) && mems >= int64(minMem) {
			filtered = append(filtered, typ)
		}
	}
	return filtered, nil
}
