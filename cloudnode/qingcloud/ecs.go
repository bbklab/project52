package qingcloud

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/bbklab/inf/pkg/ptype"
)

//
// Ecs Instance
//

// NewEcs create a new ecs instance with given settings
func (sdk *SDK) NewEcs(zone string, req *RunInstancesInput) (string, error) {
	params := sdk.publicParameters()
	params.Set("action", "RunInstances") // must
	params.Set("zone", zone)             // must

	// verify the request params
	if err := req.Validate(); err != nil {
		return "", err
	}

	// append to request params
	req.AddToParameters(params)

	var resp *RunInstancesOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return "", err // note: maybe *QcApiError, do NOT wrap any more
	}

	if len(resp.Instances) == 0 {
		return "", errors.New("abnormal response, without any instance id given")
	}
	return resp.Instances[0], nil
}

// ListEcses show all of ecs instances under given zone
// if empty zone parameter, will query all zones
func (sdk *SDK) ListEcses(zone string) ([]*InstanceWrapper, error) {

	// target zones
	var (
		queryZones []string
	)

	if zone != "" { // use specified zone id
		queryZones = append(queryZones, zone)

	} else { // use all zone ids

		zones, err := sdk.ListZones()
		if err != nil {
			return nil, err
		}
		for _, zone := range zones {
			queryZones = append(queryZones, ptype.StringV(zone.ZoneID))
		}
	}

	// query ecses list by each of zone
	// NOTE: exclude removed instances: `terminated` `ceased`
	var ret []*InstanceWrapper

	for _, zone := range queryZones {
		params := sdk.publicParameters()
		params.Set("action", "DescribeInstances")
		params.Set("verbose", "1")
		params.Set("zone", zone)
		params.Set("status.1", "pending")
		params.Set("status.2", "running")
		params.Set("status.3", "stopped")
		params.Set("status.4", "suspended")
		params.Set("limit", "100") // TODO
		var resp *DescribeInstancesOutput
		if err := sdk.apiCall(params, &resp); err != nil {
			return nil, err
		}
		for _, ins := range resp.InstanceSet {
			ret = append(ret, &InstanceWrapper{ins, zone})
		}
	}

	return ret, nil
}

// InspectEcs show details of a given ecs instance
func (sdk *SDK) InspectEcs(zone, instanceID string) (*Instance, error) {
	params := sdk.publicParameters()
	params.Set("action", "DescribeInstances")
	params.Set("verbose", "1")
	params.Set("zone", zone)
	params.Set("instances.1", instanceID)

	var resp *DescribeInstancesOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	if len(resp.InstanceSet) == 0 {
		return nil, errors.New("abnormal response, without any instance details given")
	}

	return resp.InstanceSet[0], nil
}

// WaitEcs wait ecs instance status reached to expected status until maxWait timeout
func (sdk *SDK) WaitEcs(zone, instanceID, expectStatus string, maxWait time.Duration) error {
	timeout := time.After(maxWait)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("wait ecs instance %s:%s -> %s timeout in %s", zone, instanceID, expectStatus, maxWait)

		case <-ticker.C:
			info, err := sdk.InspectEcs(zone, instanceID)
			if err != nil {
				return fmt.Errorf("WaitEcs.InspectEcs %s:%s error: %v", zone, instanceID, err)
			}
			logrus.Printf("qingcloud ecs instance %s:%s is %s ...", zone, instanceID, info.Status)
			if ptype.StringV(info.Status) == expectStatus {
				return nil
			}
		}
	}
}

// StartEcs start the specified ecs instance
// note: ecs must be `stopped` status
func (sdk *SDK) StartEcs(zone, instanceID string) error {
	params := sdk.publicParameters()
	params.Set("action", "StartInstances")
	params.Set("instances.1", instanceID)

	var resp *StartInstancesOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return err
	}

	jid := ptype.StringV(resp.JobID)
	if err := sdk.WaitJob(zone, jid, "successful", time.Second*30); err != nil {
		return fmt.Errorf("waitting for StartInstances job %s: %v", jid, err)
	}

	return nil
}

// StopEcs stop the specified ecs instance
// note: ecs must be `running` status
func (sdk *SDK) StopEcs(zone, instanceID string) error {
	params := sdk.publicParameters()
	params.Set("action", "StopInstances")
	params.Set("instances.1", instanceID)

	var resp *StopInstancesOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return err
	}

	jid := ptype.StringV(resp.JobID)
	if err := sdk.WaitJob(zone, jid, "successful", time.Second*30); err != nil {
		return fmt.Errorf("waitting for StopInstances job %s: %v", jid, err)
	}

	return nil
}

// RemoveEcs remove the specified ecs instance
// note: maybe met temporarily error like followings:
//  1400:PermissionDenied, resource [i-hdqves5i] lease info not ready yet, please try later
// if met, we will retry removal until succeed.
func (sdk *SDK) RemoveEcs(zone, instanceID string) error {
	// insepct the target Ecs to obtain it's eip
	info, err := sdk.InspectEcs(zone, instanceID)
	if err != nil {
		return err
	}

	// remove ecs instance
	params := sdk.publicParameters()
	params.Set("action", "TerminateInstances")
	params.Set("direct_cease", "1")
	params.Set("zone", zone)
	params.Set("instances.1", instanceID)

ApiRemove:
	var resp *TerminateInstancesOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		if !sdk.isTemporarilyError(err) {
			return err
		}
		logrus.Warnf("RemoveEcs met temporarily error: %v", err)
		params.Del("signature") // note: remove previous signatur appended by apiCall() ...
		time.Sleep(time.Second * 3)
		goto ApiRemove
	}

	jid := ptype.StringV(resp.JobID)
	if err := sdk.WaitJob(zone, jid, "successful", time.Second*30); err != nil {
		return fmt.Errorf("waitting for TerminateInstances job %s: %v", jid, err)
	}

	// remove related eip
	if info.EIP == nil {
		return nil
	}

	return sdk.RemoveEip(zone, ptype.StringV(info.EIP.EIPID))
}

// ListZones show all of zones qingcloud supported
func (sdk *SDK) ListZones() ([]*Zone, error) {
	params := sdk.publicParameters()
	params.Set("action", "DescribeZones")

	var resp *DescribeZonesOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	return resp.ZoneSet, nil
}

// ListInstanceTypes show all of instance types under given zone
// support cpu / memory minimal/maximize filter parameters
func (sdk *SDK) ListInstanceTypes(zone string, minCPU, maxCPU, minMem, maxMem int) ([]*InstanceType, error) {
	params := sdk.publicParameters()
	params.Set("action", "DescribeInstanceTypes")
	params.Set("zone", zone)

	var resp *DescribeInstanceTypesOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	var ret []*InstanceType
	for _, typ := range resp.InstanceTypeSet {
		var cpus, mems = ptype.IntV(typ.VCPUsCurrent), ptype.IntV(typ.MemoryCurrent)
		if cpus <= maxCPU && cpus >= minCPU && mems <= maxMem && mems >= minMem {
			ret = append(ret, typ)
		}
	}

	return ret, nil
}

//
// Nics
//

// ListNics show all of nics under given zone
// note: no use currently
func (sdk *SDK) ListNics(zone string) ([]*NIC, error) {
	params := sdk.publicParameters()
	params.Set("action", "DescribeNics")
	params.Set("zone", zone)

	var resp *DescribeNicsOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	return resp.NICSet, nil
}

//
// Eips
//

// NewEip create one eip under given zone
func (sdk *SDK) NewEip(zone string, bandwidth int) (string, string, error) {
	params := sdk.publicParameters()
	params.Set("action", "AllocateEips")
	params.Set("zone", zone)
	params.Set("bandwidth", strconv.Itoa(bandwidth))

	var resp *AllocateEIPsOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return "", "", err // note: maybe *QcApiError, do NOT wrap any more
	}

	if len(resp.EIPs) == 0 {
		return "", "", errors.New("abnormal response, without any eip id given")
	}

	var (
		eipID   = ptype.StringSliceV(resp.EIPs)[0]
		eipAddr string
	)
	eipinfo, err := sdk.InspectEip(zone, eipID)
	if err != nil {
		return "", "", fmt.Errorf("pick up the newly created eip failed: %v", err)
	}
	eipAddr = ptype.StringV(eipinfo.EIPAddr)

	return eipID, eipAddr, nil
}

// InspectEip is exported
func (sdk *SDK) InspectEip(zone, eipID string) (*EIP, error) {
	params := sdk.publicParameters()
	params.Set("action", "DescribeEips")
	params.Set("zone", zone)
	params.Set("eips.1", eipID)

	var resp *DescribeEIPsOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	if len(resp.EIPSet) == 0 {
		return nil, errors.New("abnormal response, without any eip details given")
	}

	return resp.EIPSet[0], nil
}

// ListEips show all of given status of eips under given zone
// pending, available, associated, suspended, released, ceased
func (sdk *SDK) ListEips(zone, status string) ([]*EIP, error) {
	params := sdk.publicParameters()
	params.Set("action", "DescribeEips")
	params.Set("zone", zone)
	if status == "" {
		status = "available"
	}
	params.Set("status.1", status) // pending, available, associated, suspended, released, ceased

	var resp *DescribeEIPsOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	return resp.EIPSet, nil
}

// RemoveEip remove one given eip under given zone
// note: maybe met temporarily error like followings:
//  1400:PermissionDenied, resource [eip-df098ifi] lease info not ready yet, please try later
// if met, we will retry removal until succeed.
func (sdk *SDK) RemoveEip(zone, eipID string) error {
	params := sdk.publicParameters()
	params.Set("action", "ReleaseEips")
	params.Set("zone", zone)
	params.Set("eips.1", eipID)

ApiRemove:
	var resp *ReleaseEIPsOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		if !sdk.isTemporarilyError(err) {
			return err
		}
		logrus.Warnf("RemoveEip met temporarily error: %v", err)
		params.Del("signature") // note: remove previous signatur appended by apiCall() ...
		time.Sleep(time.Second * 3)
		goto ApiRemove
	}

	jid := ptype.StringV(resp.JobID)
	if err := sdk.WaitJob(zone, jid, "successful", time.Second*30); err != nil {
		return fmt.Errorf("waitting for ReleaseEips job %s: %v", jid, err)
	}

	return nil
}

// AssignEip assign one given eip to one given ecs
func (sdk *SDK) AssignEip(zone, instanceID, eipID string) error {
	params := sdk.publicParameters()
	params.Set("action", "AssociateEip")
	params.Set("zone", zone)
	params.Set("instance", instanceID)
	params.Set("eip", eipID)

	var resp *AssociateEIPOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return err
	}

	jid := ptype.StringV(resp.JobID)
	if err := sdk.WaitJob(zone, jid, "successful", time.Second*30); err != nil {
		return fmt.Errorf("waitting for AssociateEip job %s: %v", jid, err)
	}

	return nil
}

// UnAssignEip unassign one given eip
func (sdk *SDK) UnAssignEip(zone, eipID string) error {
	params := sdk.publicParameters()
	params.Set("action", "DissociateEips")
	params.Set("zone", zone)
	params.Set("eips.1", eipID)

	var resp *DissociateEIPsOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return err
	}

	jid := ptype.StringV(resp.JobID)
	if err := sdk.WaitJob(zone, jid, "successful", time.Second*30); err != nil {
		return fmt.Errorf("waitting for DissociateEips job %s: %v", jid, err)
	}

	return nil
}

//
// Jobs
//

// InspectJob show details of a given async job
func (sdk *SDK) InspectJob(zone, jobID string) (*Job, error) {
	params := sdk.publicParameters()
	params.Set("action", "DescribeJobs")
	params.Set("zone", zone)
	params.Set("jobs.1", jobID)

	var resp *DescribeJobsOutput
	if err := sdk.apiCall(params, &resp); err != nil {
		return nil, err
	}

	if len(resp.JobSet) == 0 {
		return nil, errors.New("abnormal response, without any job details given")
	}

	return resp.JobSet[0], nil
}

// WaitJob wait job status reached to expected status until maxWait timeout
func (sdk *SDK) WaitJob(zone, jobID, expectStatus string, maxWait time.Duration) error {
	timeout := time.After(maxWait)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("wait async job %s:%s -> %s timeout in %s", zone, jobID, expectStatus, maxWait)

		case <-ticker.C:
			info, err := sdk.InspectJob(zone, jobID)
			if err != nil {
				return fmt.Errorf("WaitJob.InspectJob %s:%s error: %v", zone, jobID, err)
			}
			logrus.Debugf("Async Job %s:%s is %s ...", zone, jobID, info.Status)
			switch v := ptype.StringV(info.Status); v {
			case expectStatus, "successful":
				return nil
			case "failed", "done with failure":
				return errors.New("job failed")
			case "pending", "working":
				continue
			}

		}
	}
}
