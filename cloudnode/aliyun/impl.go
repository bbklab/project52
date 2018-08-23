// Package aliyun ...
// this file rewrap existing methods to implement the cloudsvr.Handler interface
package aliyun

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bbklab/inf/pkg/cloudsvr"
	"github.com/bbklab/inf/pkg/utils"
)

var (
	// CloudType is exported
	CloudType = "aliyun"
)

var (
	// OsImage is exported to make pass golint
	OsImage = "centos_7_03_64_40G_alibase_20170710.vhd"
	// NodeName is exported to make pass golint
	NodeName = "inf-agent-aliyun-node"
	// NodeLabels is exported to make pass golint
	NodeLabels = map[string]string{cloudsvr.CLOUDFLAGKEY: CloudType}
)

// Ping verify the cloudsvr settings could be working fine
func (sdk *SDK) Ping() error {
	return sdk.Verify()
}

// Type implement Handler
func (sdk *SDK) Type() string {
	return CloudType
}

// ListCloudRegions implement cloudsvr.Handler
func (sdk *SDK) ListCloudRegions() ([]*cloudsvr.CloudRegion, error) {
	regs, err := sdk.ListRegions()
	if err != nil {
		return nil, err
	}

	var ret []*cloudsvr.CloudRegion
	for _, reg := range regs {
		ret = append(ret, &cloudsvr.CloudRegion{
			ID:       reg.RegionID,
			Location: reg.LocalName,
		})
	}

	return ret, nil
}

// ListCloudTypes implement cloudsvr.Handler
func (sdk *SDK) ListCloudTypes(region string) ([]*cloudsvr.CloudNodeType, error) {
	types, err := sdk.ListInstanceTypes(-1, math.MaxInt32, -1, math.MaxInt32)
	if err != nil {
		return nil, err
	}

	var ret []*cloudsvr.CloudNodeType
	for _, typ := range types {
		ret = append(ret, &cloudsvr.CloudNodeType{
			ID:       typ.InstanceTypeID,
			Name:     "-",
			RegionID: "-",
			CPU:      fmt.Sprintf("%d", typ.CPUCoreCount),
			Memory:   fmt.Sprintf("%0.1fGB", typ.MemorySize),
			Disk:     "-",
		})
	}

	return ret, nil
}

// ListNodes list all aliyun ecs instances with labels key: CLOUDFLAGKEY
// the nodes listed does NOT have any auth fields `User` `Password`
// note: aliyun's api is much slower, use concurrency to speed up
func (sdk *SDK) ListNodes() ([]*cloudsvr.CloudNode, error) {
	// query all regions
	regions, err := sdk.ListRegions()
	if err != nil {
		log.Errorf("sdk.ListNodes.ListRegions() error: %v", err)
		return nil, err
	}

	var (
		ret []*cloudsvr.CloudNode
		mux sync.Mutex
		wg  sync.WaitGroup
	)

	// query each region's ecs instances
	wg.Add(len(regions))
	for _, reg := range regions {
		go func(reg RegionType) {
			defer wg.Done()
			ecses, err := sdk.ListEcses(reg.RegionID)
			if err != nil {
				log.Errorf("sdk.ListNodes.ListEcses() on %s error: %v", reg.RegionID, err)
				return
			}
			for _, ecsAttr := range ecses {
				tags := ecsAttr.Tags.Tag
				if len(tags) == 0 {
					continue
				}

				var ipaddr string
				if ipaddrs := ecsAttr.PublicIPAddress.IPAddress; len(ipaddrs) > 0 {
					ipaddr = ipaddrs[0]
				}

				key, val := tags[0].TagKey, tags[0].TagValue
				if key == cloudsvr.CLOUDFLAGKEY && val == CloudType {
					mux.Lock()
					ret = append(ret, &cloudsvr.CloudNode{
						ID:             ecsAttr.InstanceID,
						RegionOrZoneID: ecsAttr.RegionID,
						InstanceType:   ecsAttr.InstanceType,
						CloudSvrType:   sdk.Type(),
						IPAddr:         ipaddr,
						CreatTime:      ecsAttr.CreationTime,
						Status:         ecsAttr.Status,
					})
					mux.Unlock()
				}
			}
		}(reg)
	}
	wg.Wait()

	return ret, nil
}

// InspectNode show details of one given ecs instance
func (sdk *SDK) InspectNode(id, regionOrZone string) (interface{}, error) {
	return sdk.InspectEcs(regionOrZone, id)
}

// RemoveNode remove aliyun ecs instance, ignore IncorrectInstanceStatus error and retry until succeed or timeout
// 1. stop ecs
// 2. wait ecs stopped
// 3. remove ecs (ignore ...)
func (sdk *SDK) RemoveNode(node *cloudsvr.CloudNode) error {
	var (
		maxWait  = time.Second * 60
		interval = time.Second
		regionID = node.RegionOrZoneID
		ecsID    = node.ID
		err      error
	)

	if regionID == "" {
		return errors.New("aliyun node removal require region ID")
	}

	log.Printf("removing aliyun ecs %s at %s ...", ecsID, regionID)

	// inspect ecs firstly to obtain it's current status
	info, err := sdk.InspectEcs(regionID, ecsID)
	if err != nil {
		return fmt.Errorf("RemoveNode.InspectEcs error: %v", err)
	}

	// stop ecs if necessary
	if info.Status != "Stopped" {
		// stop ecs
		if err = sdk.StopEcs(ecsID); err != nil {
			return fmt.Errorf("stop aliyun ecs %s error: %v", ecsID, err)
		}

		// wait ecs to be stopped
		if err = sdk.WaitEcs(regionID, ecsID, "Stopped", time.Second*300); err != nil {
			return fmt.Errorf("aliyun ecs %s waitting to be stopped failed: %v", ecsID, err)
		}
	}

	log.Printf("aliyun ecs %s stopped", ecsID)

	// aliyun's inner status may delay few seconds, wait for a while to avoid error complains: IncorrectInstanceStatus
	for goesby := int64(0); goesby <= int64(maxWait); goesby += int64(interval) {
		time.Sleep(interval)
		err = sdk.RemoveEcs(regionID, ecsID)
		if err == nil {
			break
		}
		if strings.Contains(err.Error(), "IncorrectInstanceStatus") {
			log.Warnf("aliyun ecs %s status not synced [%v], retrying on ecs removal ...", ecsID, err)
			continue
		}
		break
	}

	if err != nil {
		log.Errorf("aliyun ecs %s failed to be removed: %v", ecsID, err)
		return err
	}

	log.Printf("aliyun ecs %s removed", ecsID)
	return nil
}

// NewNode create aliyun ecs instance, try to use prefered attributes firstly
func (sdk *SDK) NewNode(prefer *cloudsvr.PreferAttrs) (*cloudsvr.CloudNode, *cloudsvr.PreferAttrs, error) {

	var (
		password, _ = utils.GenPassword(24)
		req         = &CreateInstanceRequest{
			ImageID:                 OsImage,
			Password:                password,
			InstanceName:            NodeName,
			InstanceChargeType:      "PostPaid",     // require RMB 100+
			SecurityGroupID:         "whatever",     // will be automatic rewrite
			InternetChargeType:      "PayByTraffic", // traffic payment
			InternetMaxBandwidthOut: "100",          // 100M
			Labels:                  NodeLabels,
		}
	)

	// if prefered attributes set, use prefer region & instance-type
	if prefer != nil && prefer.Valid() == nil {
		var (
			reg = prefer.RegionOrZone
			typ = prefer.InstanceType
		)
		log.Printf("create aliyun ecs by using prefered region %s, instance type %s ...", reg, typ)

		req.RegionID = reg     // cn-beijing
		req.InstanceType = typ // ecs.n4.large

		created, err := sdk.createNode(req)
		if err != nil {
			return nil, nil, err
		}

		log.Printf("created prefered aliyun ecs succeed: %s", created.ID)
		return created, prefer, nil
	}

	log.Infoln("creating aliyun ecs by trying all regions & types ...")

	// if prefered created failed, or without prefer region & instance-type
	// try best on all region & instance-types to create the new aliyun ecs
	var (
		regions []RegionType           // all of aliyun regions
		types   []InstanceTypeItemType // all of instance types within given range of mems & cpus
		err     error
		created *cloudsvr.CloudNode
	)

	// list all regions
	regions, err = sdk.ListRegions()
	if err != nil {
		log.Errorf("sdk.NewNode.ListRegions() error: %v", err)
		return nil, nil, err
	}

	// list specified range of instance types
	types, err = sdk.ListInstanceTypes(2, 4, 2, 8) // TODO range of given cpus/mems ranges
	if err != nil {
		log.Errorf("sdk.NewNode.ListInstanceTypes() error: %v", err)
		return nil, nil, err
	}

	var (
		useRegionID, useInsType string
	)
	// range all regions & types to try to create ecs instance
	for _, reg := range regions {
		for _, typ := range types {
			req.RegionID = reg.RegionID           // cn-beijing
			req.InstanceType = typ.InstanceTypeID // ecs.n4.large

			// if created succeed, directly return
			created, err = sdk.createNode(req)
			if err == nil {
				useRegionID, useInsType = reg.RegionID, typ.InstanceTypeID
				goto END
			}

			if sdk.isFatalError(err) {
				log.Errorf("create aliyun ecs got fatal error, stop retry: %v", err)
				return nil, nil, err
			}

			log.Warnf("create aliyun ecs failed: %v, will retry another region or type", err)
		}
	}

END:
	if err != nil {
		return nil, nil, err
	}

	log.Printf("created aliyun ecs %s at %s and type is %s", created.ID, useRegionID, useInsType)
	return created, &cloudsvr.PreferAttrs{RegionOrZone: useRegionID, InstanceType: useInsType}, nil
}

// createNode actually create a new ecs with given region & instance-type
// 1. create ecs ->
// 2. assign ecs public ip ->
// 3. start ecs ->
// 4. wait ecs running
func (sdk *SDK) createNode(req *CreateInstanceRequest) (*cloudsvr.CloudNode, error) {
	var err error

	// create ecs firstly
	ecsID, err := sdk.NewEcs(req)
	if err != nil {
		return nil, fmt.Errorf("aliyun ecs create failed: %v", err)
	}
	log.Printf("aliyun ecs %s created at %s", ecsID, req.RegionID)

	// if create succeed, but other operations failed, clean up the newly created ecs instance to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("aliyun cloud node creation failed, clean up the newly created ecs instance %s. [%v]", ecsID, err)
			sdk.RemoveNode(&cloudsvr.CloudNode{ID: ecsID, RegionOrZoneID: req.RegionID})
		}
	}()

	// now ecs is stopped, we assign an public ip to it
	ip, err := sdk.AssignEcsPublicIP(ecsID)
	if err != nil {
		return nil, fmt.Errorf("aliyun ecs %s assign public ip failed: %v", ecsID, err)
	}
	log.Printf("aliyun ecs %s assgined public ipaddress %s", ecsID, ip)

	// start ecs
	if err = sdk.StartEcs(ecsID); err != nil {
		return nil, fmt.Errorf("aliyun ecs %s start failed: %v", ecsID, err)
	}
	log.Printf("aliyun ecs %s starting", ecsID)

	// wait ecs to be running
	if err = sdk.WaitEcs(req.RegionID, ecsID, "Running", time.Second*300); err != nil {
		return nil, fmt.Errorf("aliyun ecs %s waitting to be running failed: %v", ecsID, err)
	}
	log.Printf("aliyun ecs %s is Running now", ecsID)

	return &cloudsvr.CloudNode{
		ID:             ecsID,
		RegionOrZoneID: req.RegionID,
		InstanceType:   req.InstanceType,
		CloudSvrType:   sdk.Type(),
		IPAddr:         ip,
		Port:           "22",
		User:           "root",
		Password:       req.Password,
	}, nil
}
