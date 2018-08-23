// Package vultr ...
// this file rewrap existing methods to implement the cloudsvr.Handler interface
//
package vultr

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
	CloudType = "vultr"
)

var (
	// OsImage is exported
	OsImage = 167 // CentOS 7 x64
	// NodeLabelPrefix is exported
	NodeLabelPrefix = "inf-agent-vultr-node"
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
			ID:       strconv.Itoa(reg.ID),
			Location: fmt.Sprintf("%s %s %s", reg.Continent, reg.Country, reg.State),
		})
	}

	return ret, nil
}

// ListCloudTypes implement cloudsvr.Handler
func (sdk *SDK) ListCloudTypes(region string) ([]*cloudsvr.CloudNodeType, error) {
	types, err := sdk.ListPlans(-1, math.MaxInt32, -1, math.MaxInt32)
	if err != nil {
		return nil, err
	}

	var ret []*cloudsvr.CloudNodeType
	for _, typ := range types {
		ret = append(ret, &cloudsvr.CloudNodeType{
			ID:       fmt.Sprintf("%d", typ.ID),
			Name:     typ.Name,
			RegionID: "-", // TODO fixme later
			CPU:      fmt.Sprintf("%d", typ.VCpus),
			Memory:   fmt.Sprintf("%0.1fGB", float64(typ.RAM)/float64(1000)),
			Disk:     fmt.Sprintf("%dGB", typ.Disk),
		})
	}

	return ret, nil
}

// ListNodes list all vultr ecs instances with label prefix: NodeLabelPrefix
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
			ret = append(ret, &cloudsvr.CloudNode{
				ID:             ecs.ID,
				RegionOrZoneID: strconv.Itoa(ecs.RegionID),
				InstanceType:   strconv.Itoa(ecs.PlanID),
				CloudSvrType:   sdk.Type(),
				IPAddr:         ecs.MainIP,
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

// RemoveNode remove vultr ecs instance
func (sdk *SDK) RemoveNode(node *cloudsvr.CloudNode) error {
	var (
		id, _ = strconv.Atoi(node.ID)
	)
	return sdk.RemoveEcs(id)
}

// NewNode create vultr ecs instance, try to use prefered attributes firstly
func (sdk *SDK) NewNode(prefer *cloudsvr.PreferAttrs) (*cloudsvr.CloudNode, *cloudsvr.PreferAttrs, error) {
	var (
		req = &CreateServerInput{
			Label:      fmt.Sprintf("%s-%s", NodeLabelPrefix, utils.RandomString(6)), // with label prefix
			OSID:       167,                                                          // CentOS 7 x64
			EnableIPv6: false,
			// DCID:       region,
			// VPSPLANID:  plan,
		}
	)

	// if prefered attributes set, use prefer region & instance-type
	if prefer != nil && prefer.Valid() == nil {

		log.Printf("create vultr ecs by using prefered region %s, instance plan %s ...", prefer.RegionOrZone, prefer.InstanceType)

		req.DCID, _ = strconv.Atoi(prefer.RegionOrZone)
		req.VPSPLANID, _ = strconv.Atoi(prefer.InstanceType)

		created, err := sdk.createNode(req)
		if err != nil {
			return nil, nil, err
		}

		log.Printf("created prefered vultr ecs succeed: %s", created.ID)
		return created, prefer, nil
	}

	log.Infoln("creating vultr ecs by trying all regions & plans ...")

	// if prefered created failed, or without prefer region & instance-type
	// try best on all region & instance-types to create the new vultr ecs
	var (
		created *cloudsvr.CloudNode
		err     error
	)

	// list all instance plans
	plans, err := sdk.ListPlans(1, 2, 1024, 4096) // TODO range of given cpus/mems ranges
	if err != nil {
		log.Errorf("sdk.NewNode.ListPlans() on all plans error: %v", err)
		return nil, nil, err
	}

	var (
		useRegion, useType string
	)

	// range all plans to try to create ecs instance
	for _, plan := range plans {
		for _, region := range plan.Regions {

			req.DCID = region
			req.VPSPLANID = plan.ID

			// if created succeed, directly return
			created, err = sdk.createNode(req)
			if err == nil {
				useRegion, useType = strconv.Itoa(region), strconv.Itoa(plan.ID)
				goto END
			}

			if sdk.isFatalError(err) {
				log.Errorf("create vultr ecs got fatal error, stop retry: %v", err)
				return nil, nil, err
			}

			log.Warnf("create vultr ecs failed: %v, will retry another region or type", err)
		}
	}

END:
	if err != nil {
		return nil, nil, err
	}

	log.Printf("created vultr ecs %s at %s and type is %s", created.ID, useRegion, useType)
	return created, &cloudsvr.PreferAttrs{RegionOrZone: useRegion, InstanceType: useType}, nil
}

// createNode actually create a new ecs with given region & instance-type
func (sdk *SDK) createNode(req *CreateServerInput) (*cloudsvr.CloudNode, error) {
	var err error

	err = req.Validate()
	if err != nil {
		return nil, err
	}

	// create ecs
	ecsID, err := sdk.NewEcs(req)
	if err != nil {
		return nil, err
	}
	log.Printf("vultr ecs %d created", ecsID)

	// if failed , clean up the newly created ecs instance to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("vultr cloud node creation failed, clean up the newly created ecs instance %d. [%v]", ecsID, err)
			sdk.RemoveEcs(ecsID)
		}
	}()

	// wait ecs to be running
	err = sdk.WaitEcs(ecsID, "active", time.Second*120)
	if err != nil {
		return nil, fmt.Errorf("vultr ecs %d waitting to be running failed: %v", ecsID, err)
	}
	log.Printf("vultr ecs %d is running now", ecsID)

	// inspect ecs instance
	info, err := sdk.InspectEcs(ecsID)
	if err != nil {
		return nil, err
	}

	var ipAddr = info.MainIP
	log.Printf("vultr ecs %d got the public ip %s", ecsID, ipAddr)

	// return the final
	return &cloudsvr.CloudNode{
		ID:             strconv.Itoa(ecsID),
		RegionOrZoneID: strconv.Itoa(info.RegionID),
		InstanceType:   strconv.Itoa(info.PlanID),
		CloudSvrType:   sdk.Type(),
		IPAddr:         ipAddr,
		Port:           "22",
		User:           "root",
		Password:       info.DefaultPassword,
	}, nil
}
