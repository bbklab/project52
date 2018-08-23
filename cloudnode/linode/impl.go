// Package linode ...
// this file rewrap existing methods to implement the cloudsvr.Handler interface
//
package linode

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/bbklab/inf/pkg/cloudsvr"
	"github.com/bbklab/inf/pkg/utils"
)

var (
	// CloudType is exported
	CloudType = "linode"
)

var (
	// OsImage is exported
	OsImage = "linode/centos7"
	// NodeLabelPrefix is exported
	NodeLabelPrefix = "inf-agent-linode-node"
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
	regs, err := sdk.ListRegions()
	if err != nil {
		return nil, err
	}

	var ret []*cloudsvr.CloudRegion
	for _, reg := range regs {
		ret = append(ret, &cloudsvr.CloudRegion{
			ID:       reg.ID,
			Location: reg.Country,
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
			ID:       typ.ID,
			Name:     "-",
			RegionID: "-",
			CPU:      fmt.Sprintf("%d", typ.VCpus),
			Memory:   fmt.Sprintf("%0.1fGB", float64(typ.Memory)/float64(1024)),
			Disk:     fmt.Sprintf("%0.1fGB", float64(typ.Disk)/float64(1024)),
		})
	}

	return ret, nil
}

// ListNodes list all linode ecs instances with label prefix: NodeLabelPrefix
// the nodes listed does NOT have any auth fields `User` `Password`
func (sdk *SDK) ListNodes() ([]*cloudsvr.CloudNode, error) {

	ecses, err := sdk.ListEcses()
	if err != nil {
		log.Errorf("sdk.ListNodes.ListEcses() on all regions error: %v", err)
		return nil, err
	}

	var ret []*cloudsvr.CloudNode

	for _, ecs := range ecses {
		if strings.HasPrefix(ecs.Label, NodeLabelPrefix) {
			var ipaddr string
			if len(ecs.IPv4) > 0 {
				ipaddr = ecs.IPv4[0]
			}
			ret = append(ret, &cloudsvr.CloudNode{
				ID:             strconv.Itoa(ecs.ID),
				RegionOrZoneID: ecs.Region,
				InstanceType:   ecs.Type,
				CloudSvrType:   sdk.Type(),
				IPAddr:         ipaddr,
				CreatTime:      ecs.Created,
				Status:         ecs.Status,
			})
		}
	}

	return ret, nil
}

// InspectNode show details of one given ecs instance
func (sdk *SDK) InspectNode(id, regionOrZone string) (interface{}, error) {
	idN, _ := strconv.Atoi(id)
	return sdk.InspectEcs(idN)
}

// RemoveNode remove linode ecs instance
func (sdk *SDK) RemoveNode(node *cloudsvr.CloudNode) error {
	var (
		id, _ = strconv.Atoi(node.ID)
	)
	return sdk.RemoveEcs(id)
}

// NewNode create linode ecs instance, try to use prefered attributes firstly
func (sdk *SDK) NewNode(prefer *cloudsvr.PreferAttrs) (*cloudsvr.CloudNode, *cloudsvr.PreferAttrs, error) {
	var (
		password, _ = utils.GenPassword(24)
		req         = &CreateInstancesInput{
			Label:        fmt.Sprintf("%s-%s", NodeLabelPrefix, utils.RandomString(6)), // with label prefix, (24+1+6 < 32)
			Distribution: OsImage,
			RootPass:     password,
			Booted:       true,
		}
	)

	// if prefered attributes set, use prefer region & instance-type
	if prefer != nil && prefer.Valid() == nil {

		log.Printf("create linode ecs by using prefered region %s, instance type %s ...", prefer.RegionOrZone, prefer.InstanceType)

		req.Region = prefer.RegionOrZone
		req.Type = prefer.InstanceType

		created, err := sdk.createNode(req)
		if err != nil {
			return nil, nil, err
		}

		log.Printf("created prefered linode ecs succeed: %s", created.ID)
		return created, prefer, nil
	}

	log.Infoln("creating linode ecs by trying all regions & types ...")

	// if prefered created failed, or without prefer region & instance-type
	// try best on all region & instance-types to create the new linode ecs
	var (
		regions []*Region // all of linode regions
		err     error
		created *cloudsvr.CloudNode
	)

	// list all regions
	regions, err = sdk.ListRegions()
	if err != nil {
		log.Errorf("sdk.NewNode.ListRegions() error: %v", err)
		return nil, nil, err
	}

	// list all instance types
	types, err := sdk.ListInstanceTypes(1, 2, 1024, 4096) // TODO range of given cpus/mems ranges
	if err != nil {
		log.Errorf("sdk.NewNode.ListInstanceTypes() on all regions error: %v", err)
		return nil, nil, err
	}

	var (
		useRegion, useType string
	)

	// range all regions & types to try to create ecs instance
	for _, region := range regions {

		for _, typ := range types {

			req.Region = region.ID
			req.Type = typ.ID

			// if created succeed, directly return
			created, err = sdk.createNode(req)
			if err == nil {
				useRegion, useType = region.ID, typ.ID
				goto END
			}

			if sdk.isFatalError(err) {
				log.Errorf("create linode ecs got fatal error, stop retry: %v", err)
				return nil, nil, err
			}

			log.Warnf("create linode ecs failed: %v, will retry another region or type", err)
		}
	}

END:
	if err != nil {
		return nil, nil, err
	}

	log.Printf("created linode ecs %s at %s and type is %s", created.ID, useRegion, useType)
	return created, &cloudsvr.PreferAttrs{RegionOrZone: useRegion, InstanceType: useType}, nil
}

// createNode actually create a new ecs with given region & instance-type
func (sdk *SDK) createNode(req *CreateInstancesInput) (*cloudsvr.CloudNode, error) {
	var err error

	err = req.Validate()
	if err != nil {
		return nil, err
	}

	// create ecs
	ecsID, err := sdk.NewEcs(req)
	if err != nil {
		return nil, err // note: maybe *LnApiError, do NOT wrap any more
	}
	log.Printf("linode ecs %d created", ecsID)

	// if failed , clean up the newly created ecs instance to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("linode cloud node creation failed, clean up the newly created ecs instance %d. [%v]", ecsID, err)
			sdk.RemoveEcs(ecsID)
		}
	}()

	// wait ecs to be running
	err = sdk.WaitEcs(ecsID, "running", time.Second*120)
	if err != nil {
		return nil, fmt.Errorf("linode ecs %d waitting to be running failed: %v", ecsID, err)
	}
	log.Printf("linode ecs %d is running now", ecsID)

	// inspect ecs instance
	info, err := sdk.InspectEcs(ecsID)
	if err != nil {
		return nil, err
	}

	var ipAddr string
	if len(info.IPv4) > 0 {
		ipAddr = info.IPv4[0]
		log.Printf("linode ecs %d got the public ip %s", ecsID, ipAddr)
	}

	// return the final
	return &cloudsvr.CloudNode{
		ID:             strconv.Itoa(ecsID),
		RegionOrZoneID: info.Region,
		InstanceType:   info.Type,
		CloudSvrType:   sdk.Type(),
		IPAddr:         ipAddr,
		Port:           "22",
		User:           "root",
		Password:       req.RootPass,
	}, nil
}
