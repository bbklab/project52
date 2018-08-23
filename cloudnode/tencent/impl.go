// Package tencent ...
// this file rewrap existing methods to implement the cloudsvr.Handler interface
package tencent

import (
	"errors"
	"fmt"
	"math"
	"time"

	log "github.com/Sirupsen/logrus"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"

	"github.com/bbklab/inf/pkg/cloudsvr"
	"github.com/bbklab/inf/pkg/ptype"
	"github.com/bbklab/inf/pkg/utils"
)

var (
	// CloudType is exported
	CloudType = "tencent"
)

var (
	// OsImage is exported to make pass golint
	OsImage = "img-dkwyg6sr"
	// NodeName is exported to make pass golint
	NodeName = "inf-agent-tencent-node"
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
			ID:       ptype.StringV(reg.Region),
			Location: ptype.StringV(reg.RegionName),
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
			ID:       ptype.StringV(typ.InstanceType),
			Name:     "-",
			RegionID: ptype.StringV(typ.Zone) + " (zone)",
			CPU:      fmt.Sprintf("%d", ptype.Int64V(typ.CPU)),
			Memory:   fmt.Sprintf("%dGB", ptype.Int64V(typ.Memory)),
			Disk:     "-",
		})
	}

	return ret, nil
}

// ListNodes list all tencent ecs instances with labels key: CLOUDFLAGKEY
// the nodes listed does NOT have any auth fields `User` `Password`
func (sdk *SDK) ListNodes() ([]*cloudsvr.CloudNode, error) {
	ecses, err := sdk.ListEcses("", NodeLabels)
	if err != nil {
		return nil, err
	}

	var ret []*cloudsvr.CloudNode
	for _, ecs := range ecses {
		var ipaddr string
		if ipaddrs := ptype.StringSliceV(ecs.PublicIpAddresses); len(ipaddrs) > 0 {
			ipaddr = ipaddrs[0]
		}
		ret = append(ret, &cloudsvr.CloudNode{
			ID:             ptype.StringV(ecs.InstanceId),
			RegionOrZoneID: regionOfZone(ptype.StringV(ecs.Placement.Zone)),
			InstanceType:   ptype.StringV(ecs.InstanceType),
			CloudSvrType:   sdk.Type(),
			IPAddr:         ipaddr,
			CreatTime:      ptype.StringV(ecs.CreatedTime),
			Status:         ptype.StringV(ecs.InstanceState),
		})
	}

	return ret, nil
}

// InspectNode show details of one given ecs instance
func (sdk *SDK) InspectNode(id, regionOrZone string) (interface{}, error) {
	return sdk.InspectEcs(regionOrZone, id)
}

// RemoveNode remove tencent ecs instance
func (sdk *SDK) RemoveNode(node *cloudsvr.CloudNode) error {
	var (
		region = node.RegionOrZoneID
		ecsID  = node.ID
	)

	if region == "" {
		return errors.New("tencent node removal require region ID")
	}

	err := sdk.RemoveEcs(region, ecsID)
	if err != nil {
		log.Errorf("remove tencent ecs %s error: %v", ecsID, err)
		return err
	}

	log.Printf("tencent ecs %s removed", ecsID)
	return nil
}

// NewNode create tencent ecs instance, try to use prefered attributes firstly
// Note: only 20-30 `POSTPAID_BY_HOUR` cvm instances could be bought per-zone / per-month / per-user
// See: https://cloud.tencent.com/document/product/213/2664
func (sdk *SDK) NewNode(prefer *cloudsvr.PreferAttrs) (*cloudsvr.CloudNode, *cloudsvr.PreferAttrs, error) {
	var (
		password, _ = utils.GenPassword(16) // See: https://cloud.tencent.com/document/api/213/15753#LoginSettings
		req         = &cvm.RunInstancesRequest{
			InstanceChargeType: ptype.String("POSTPAID_BY_HOUR"), // post paid
			ImageId:            ptype.String(OsImage),            // centos7.3 64bit
			InternetAccessible: &cvm.InternetAccessible{
				InternetChargeType:      ptype.String("TRAFFIC_POSTPAID_BY_HOUR"), // traffic payment
				InternetMaxBandwidthOut: ptype.Int64(100),                         // 100M traffic
				PublicIpAssigned:        ptype.Bool(true),                         // free public ipaddress
			},
			InstanceCount: ptype.Int64(1),
			InstanceName:  ptype.String(NodeName),
			LoginSettings: &cvm.LoginSettings{
				Password: ptype.String(password),
			},
			EnhancedService: &cvm.EnhancedService{
				SecurityService: &cvm.RunSecurityServiceEnabled{Enabled: ptype.Bool(false)},
				MonitorService:  &cvm.RunMonitorServiceEnabled{Enabled: ptype.Bool(false)},
			},
			ClientToken: ptype.String(utils.RandomString(16)),
			TagSpecification: []*cvm.TagSpecification{
				{
					ResourceType: ptype.String("instance"), // fixed
					Tags: []*cvm.Tag{
						{
							Key:   ptype.String(cloudsvr.CLOUDFLAGKEY),
							Value: ptype.String(CloudType),
						},
					},
				},
			},
		}
	)

	// if prefered attributes set, use prefer region & instance-type
	if prefer != nil && prefer.Valid() == nil {
		var (
			reg = prefer.RegionOrZone
			typ = prefer.InstanceType
		)
		log.Printf("create tencent ecs by using prefered region %s, instance type %s ...", reg, typ)

		zone := sdk.PickupZone(reg)
		req.Placement = &cvm.Placement{
			Zone: ptype.String(zone),
		}
		req.InstanceType = ptype.String(typ)

		created, err := sdk.createNode(reg, req)
		if err != nil {
			return nil, nil, err
		}

		log.Printf("created prefered tencent ecs succeed: %s", created.ID)
		return created, prefer, nil
	}

	log.Infoln("creating tencent ecs by trying all regions & types ...")

	// if prefered created failed, or without prefer region & instance-type
	// try best on all region & instance-types to create the new tencent ecs
	var (
		err     error
		created *cloudsvr.CloudNode
	)

	// list all regions
	regions, err := sdk.ListRegions()
	if err != nil {
		log.Errorf("sdk.NewNode.ListRegions() error: %v", err)
		return nil, nil, err
	}

	// list specified range of instance types
	types, err := sdk.ListInstanceTypes(1, 4, 2, 4) // TODO range of given cpus/mems ranges
	if err != nil {
		log.Errorf("sdk.NewNode.ListInstanceTypes() error: %v", err)
		return nil, nil, err
	}

	var (
		useRegionID, useInsType string
	)
	// range all regions & types to try to create ecs instance
	for _, reg := range regions {
		regid := ptype.StringV(reg.Region)

		for _, typ := range types {
			typid := ptype.StringV(typ.InstanceType)

			zone := sdk.PickupZone(regid)
			req.Placement = &cvm.Placement{
				Zone: ptype.String(zone),
			}
			req.InstanceType = ptype.String(typid)

			// if created succeed, directly return
			created, err = sdk.createNode(regid, req)
			if err == nil {
				useRegionID, useInsType = regid, typid
				goto END
			}

			if sdk.isFatalError(err) {
				log.Errorf("create tencent ecs got fatal error, stop retry: %v", err)
				return nil, nil, err
			}

			log.Warnf("create tencent ecs failed: %v, will retry another region or type", err)
			time.Sleep(time.Millisecond * 200)
		}
	}

END:
	if err != nil {
		return nil, nil, err
	}

	log.Printf("created tencent ecs %s at %s and type is %s", created.ID, useRegionID, useInsType)
	return created, &cloudsvr.PreferAttrs{RegionOrZone: useRegionID, InstanceType: useInsType}, nil
}

// createNode actually create a new ecs with given region & instance-type
func (sdk *SDK) createNode(region string, req *cvm.RunInstancesRequest) (*cloudsvr.CloudNode, error) {
	var err error

	// create ecs firstly
	ecsID, err := sdk.NewEcs(region, req)
	if err != nil {
		return nil, fmt.Errorf("tencent ecs create failed: %v", err)
	}
	log.Printf("tencent ecs %s created at %s", ecsID, region)

	// if create succeed, but other operations failed, clean up the newly created ecs instance to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("tencent cloud node creation failed, clean up the newly created ecs instance %s. [%v]", ecsID, err)
			sdk.RemoveNode(&cloudsvr.CloudNode{ID: ecsID, RegionOrZoneID: region})
		}
	}()

	// wait ecs to be running
	err = sdk.WaitEcs(region, ecsID, "RUNNING", time.Second*300)
	if err != nil {
		return nil, fmt.Errorf("tencent ecs %s waitting to be running failed: %v", ecsID, err)
	}
	log.Printf("tencent ecs %s is Running now", ecsID)

	// get ecs public ip address
	info, err := sdk.InspectEcs(region, ecsID)
	if err != nil {
		return nil, fmt.Errorf("pick up the newly created tencent ecs %s failed: %v", ecsID, err)
	}
	ips := ptype.StringSliceV(info.PublicIpAddresses)
	if len(ips) == 0 {
		return nil, fmt.Errorf("the newly created tencent ecs %s has no public ip addresses", ecsID)
	}

	return &cloudsvr.CloudNode{
		ID:             ecsID,
		RegionOrZoneID: region,
		InstanceType:   ptype.StringV(info.InstanceType),
		CloudSvrType:   sdk.Type(),
		IPAddr:         ips[0],
		Port:           "22",
		User:           "root",
		Password:       ptype.StringV(req.LoginSettings.Password),
	}, nil
}
