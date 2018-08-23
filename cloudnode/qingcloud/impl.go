// Package qingcloud ...
// this file rewrap existing methods to implement the cloudsvr.Handler interface
//
package qingcloud

import (
	"errors"
	"fmt"
	"math"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/bbklab/inf/pkg/cloudsvr"
	"github.com/bbklab/inf/pkg/ptype"
	"github.com/bbklab/inf/pkg/utils"
)

var (
	// CloudType is exported
	CloudType = "qingcloud"
)

var (
	// OsImage is exported
	OsImage = "centos73x64"
	// NodeName is exported
	NodeName = "inf-agent-qingcloud-node"
)

// Ping verify the cloudsvr settings could be working fine
func (sdk *SDK) Ping() error {
	return sdk.Verify()
}

// Type implement cloudsvr.Handler
func (sdk *SDK) Type() string {
	return CloudType
}

// ListCloudRegions implement cloudsvr.Handler
func (sdk *SDK) ListCloudRegions() ([]*cloudsvr.CloudRegion, error) {
	regs, err := sdk.ListZones()
	if err != nil {
		return nil, err
	}

	var ret []*cloudsvr.CloudRegion
	for _, reg := range regs {
		ret = append(ret, &cloudsvr.CloudRegion{
			ID:       ptype.StringV(reg.ZoneID),
			Location: "",
		})
	}

	return ret, nil
}

// ListCloudTypes implement cloudsvr.Handler
func (sdk *SDK) ListCloudTypes(region string) ([]*cloudsvr.CloudNodeType, error) {
	zones, err := sdk.ListZones()
	if err != nil {
		return nil, err
	}

	var regs = make([]*Zone, 0, 0)
	if region != "" {
		for _, zone := range zones {
			if ptype.StringV(zone.ZoneID) == region {
				regs = append(regs, zone)
				break
			}
		}
	} else {
		regs = zones
	}

	var ret []*cloudsvr.CloudNodeType
	for _, reg := range regs {
		types, err := sdk.ListInstanceTypes(ptype.StringV(reg.ZoneID), -1, math.MaxInt32, -1, math.MaxInt32)
		if err != nil {
			return nil, err
		}
		for _, typ := range types {
			ret = append(ret, &cloudsvr.CloudNodeType{
				ID:       ptype.StringV(typ.InstanceTypeID),
				Name:     ptype.StringV(typ.InstanceTypeName),
				RegionID: ptype.StringV(reg.ZoneID),
				CPU:      fmt.Sprintf("%d", ptype.IntV(typ.VCPUsCurrent)),
				Memory:   fmt.Sprintf("%0.1fGB", float64(ptype.IntV(typ.MemoryCurrent))/float64(1024)),
				Disk:     "-",
			})
		}
	}

	return ret, nil
}

// ListNodes list all qingcloud ecs instances with InstanceName == NodeName
// the nodes listed does NOT have any auth fields `User` `Password`
func (sdk *SDK) ListNodes() ([]*cloudsvr.CloudNode, error) {

	ecses, err := sdk.ListEcses("")
	if err != nil {
		log.Errorf("sdk.ListNodes.ListEcses() on all zones error: %v", err)
		return nil, err
	}

	var ret []*cloudsvr.CloudNode

	for _, ecs := range ecses {
		var ipaddr string
		if ecs.EIP != nil {
			ipaddr = ptype.StringV(ecs.EIP.EIPAddr)
		}
		if ptype.StringV(ecs.InstanceName) == NodeName {
			ret = append(ret, &cloudsvr.CloudNode{
				ID:             ptype.StringV(ecs.InstanceID),
				RegionOrZoneID: ecs.Zone,
				InstanceType:   ptype.StringV(ecs.InstanceType),
				CloudSvrType:   sdk.Type(),
				IPAddr:         ipaddr,
				CreatTime:      ptype.TimeV(ecs.CreateTime).Format(time.RFC3339),
				Status:         ptype.StringV(ecs.Status),
			})
		}
	}

	return ret, nil
}

// InspectNode show details of one given ecs instance
func (sdk *SDK) InspectNode(id, regionOrZone string) (interface{}, error) {
	return sdk.InspectEcs(regionOrZone, id)
}

// RemoveNode remove qingcloud ecs instance
func (sdk *SDK) RemoveNode(node *cloudsvr.CloudNode) error {
	var (
		zone = node.RegionOrZoneID
		id   = node.ID
	)
	if zone == "" {
		return errors.New("qingcloud node removal require zone ID")
	}
	return sdk.RemoveEcs(zone, id)
}

// NewNode create qingcloud ecs instance, try to use prefered attributes firstly
func (sdk *SDK) NewNode(prefer *cloudsvr.PreferAttrs) (*cloudsvr.CloudNode, *cloudsvr.PreferAttrs, error) {
	var (
		password, _ = utils.GenPassword(24)
		req         = &RunInstancesInput{
			ImageID:       ptype.String(OsImage),
			LoginMode:     ptype.String("passwd"), // fixed
			LoginPasswd:   ptype.String(password),
			InstanceClass: ptype.Int(0),
			InstanceName:  ptype.String(NodeName),
			Count:         ptype.Int(1),
			VxNets:        ptype.StringSlice([]string{"vxnet-0"}), // fixed: base vxnet
		}
	)

	// if prefered attributes set, use prefer zone & instance-type
	if prefer != nil && prefer.Valid() == nil {

		log.Printf("create qingcloud ecs by using prefered zone %s, instance type %s ...", prefer.RegionOrZone, prefer.InstanceType)

		req.InstanceType = ptype.String(prefer.InstanceType)

		created, err := sdk.createNode(prefer.RegionOrZone, req)
		if err != nil {
			return nil, nil, err
		}

		log.Printf("created prefered qingcloud ecs succeed: %s", created.ID)
		return created, prefer, nil
	}

	log.Infoln("creating qingcloud ecs by trying all zones & types ...")

	// if prefered created failed, or without prefer zone & instance-type
	// try best on all zone & instance-types to create the new qingcloud ecs
	var (
		zones   []*Zone // all of qingcloud zones
		err     error
		created *cloudsvr.CloudNode
	)

	// list all zones
	zones, err = sdk.ListZones()
	if err != nil {
		log.Errorf("sdk.NewNode.ListZones() error: %v", err)
		return nil, nil, err
	}

	var (
		useZone, useType string
	)

	// range all zones & types to try to create ecs instance
	for _, zone := range zones {

		// list specified range of instance types
		types, err := sdk.ListInstanceTypes(ptype.StringV(zone.ZoneID), 1, 2, 1024, 4096) // TODO range of given cpus/mems ranges
		if err != nil {
			log.Errorf("sdk.NewNode.ListInstanceTypes() on zone %s error: %v", zone, err)
			continue
		}

		for _, typ := range types {
			req.InstanceType = typ.InstanceTypeID

			// if created succeed, directly return
			created, err = sdk.createNode(ptype.StringV(zone.ZoneID), req)
			if err == nil {
				useZone, useType = ptype.StringV(zone.ZoneID), ptype.StringV(typ.InstanceTypeID)
				goto END
			}

			if sdk.isFatalError(err) {
				log.Errorf("create qingcloud ecs got fatal error, stop retry: %v", err)
				return nil, nil, err
			}

			log.Warnf("create qingcloud ecs failed: %v, will retry another region or type", err)
		}
	}

END:
	if err != nil {
		return nil, nil, err
	}

	log.Printf("created qingcloud ecs %s at %s and type is %s", created.ID, useZone, useType)
	return created, &cloudsvr.PreferAttrs{RegionOrZone: useZone, InstanceType: useType}, nil

}

// createNode actually create a new ecs with given region & instance-type
// 1. create ecs ->
// 2. wait ecs running ->
// 3. allocate eip and assign to ecs
func (sdk *SDK) createNode(zone string, req *RunInstancesInput) (*cloudsvr.CloudNode, error) {
	var err error

	// create ecs
	ecsID, err := sdk.NewEcs(zone, req)
	if err != nil {
		return nil, err // note: maybe *QcApiError, do NOT wrap any more
	}
	log.Printf("qingcloud ecs %s created at %s", ecsID, zone)

	// if failed , clean up the newly created ecs instance to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("qingcloud cloud node creation failed, clean up the newly created ecs instance %s. [%v]", ecsID, err)
			sdk.RemoveEcs(zone, ecsID)
		}
	}()

	// wait ecs to be running
	err = sdk.WaitEcs(zone, ecsID, "running", time.Second*120)
	if err != nil {
		return nil, fmt.Errorf("qingcloud ecs %s waitting to be running failed: %v", ecsID, err)
	}
	log.Printf("qingcloud ecs %s is running now", ecsID)

	// allocated & assign an public ip to it
	eipID, eipAddr, err := sdk.NewEip(zone, 10) // 10M
	if err != nil {
		return nil, err // note: maybe *QcApiError, do NOT wrap any more
	}
	log.Printf("qingcloud eip %s:%s allocated at %s", eipID, eipAddr, zone)

	// if failed, clean up the newly created eip to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("qingcloud cloud node creation failed, clean up the newly created eip %s:%s. [%v]", eipID, eipAddr, err)
			sdk.RemoveEip(zone, eipID)
		}
	}()

	err = sdk.AssignEip(zone, ecsID, eipID)
	if err != nil {
		return nil, fmt.Errorf("qingcloud ecs %s assign public ip failed: %v", ecsID, err)
	}
	log.Printf("qingcloud ecs %s assgined public ipaddress %s", ecsID, eipAddr)

	// return the final
	return &cloudsvr.CloudNode{
		ID:             ecsID,
		RegionOrZoneID: zone,
		InstanceType:   ptype.StringV(req.InstanceType),
		CloudSvrType:   sdk.Type(),
		IPAddr:         eipAddr,
		Port:           "22",
		User:           "root",
		Password:       ptype.StringV(req.LoginPasswd),
	}, nil
}
