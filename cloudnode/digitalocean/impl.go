// Package digitalocean ...
// this file rewrap existing methods to implement the cloudsvr.Handler interface
//
package digitalocean

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
	CloudType = "digitalocean"
)

var (
	// OsImage is exported
	OsImage = 33948356 // fedora-28-x64 (centos 7.2 x64 is already unavaliable)
	// NodeNamePrefix is exported
	NodeNamePrefix = "inf-agent-digitalocean-node"
	// SSHKeyNamePrefix is exported
	SSHKeyNamePrefix = "inf-agent-digitalocean-sshkey"
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
		if reg.Available {
			ret = append(ret, &cloudsvr.CloudRegion{
				ID:       reg.Slug,
				Location: reg.Name,
			})
		}
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
			ID:       typ.Slug,
			Name:     "-",
			RegionID: "-", // TODO fixme later
			CPU:      fmt.Sprintf("%d", typ.VCpus),
			Memory:   fmt.Sprintf("%0.1fGB", float64(typ.Memory)/float64(1024)),
			Disk:     fmt.Sprintf("%dGB", typ.Disk),
		})
	}

	return ret, nil
}

// ListNodes list all digitalocean ecs instances with label prefix: NodeNamePrefix
// the nodes listed does NOT have any auth fields `User` `Password`
func (sdk *SDK) ListNodes() ([]*cloudsvr.CloudNode, error) {

	ecses, err := sdk.ListEcses()
	if err != nil {
		log.Errorf("sdk.ListNodes.ListEcses() on all regions error: %v", err)
		return nil, err
	}

	var ret []*cloudsvr.CloudNode

	for _, ecs := range ecses {
		if strings.HasPrefix(ecs.Name, NodeNamePrefix) {
			var ipaddr string
			if ecs.Networks != nil && len(ecs.Networks.V4) > 0 {
				ipaddr = ecs.Networks.V4[0].IPAddress
			}
			var region string
			if ecs.Region != nil {
				region = ecs.Region.Slug
			}
			ret = append(ret, &cloudsvr.CloudNode{
				ID:             strconv.Itoa(ecs.ID),
				RegionOrZoneID: region,
				InstanceType:   ecs.SizeSlug,
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

// RemoveNode remove digitalocean ecs instance
func (sdk *SDK) RemoveNode(node *cloudsvr.CloudNode) error {
	var (
		id, _ = strconv.Atoi(node.ID)
	)
	return sdk.RemoveEcs(id)
}

// NewNode create digitalocean ecs instance, try to use prefered attributes firstly
func (sdk *SDK) NewNode(prefer *cloudsvr.PreferAttrs) (*cloudsvr.CloudNode, *cloudsvr.PreferAttrs, error) {

	var (
		suffix = utils.RandomString(6)
		req    = &CreateInstancesInput{
			Name:  fmt.Sprintf("%s-%s", NodeNamePrefix, suffix), // with label prefix
			Image: OsImage,
		}
	)

	// if prefered attributes set, use prefer region & instance-type
	if prefer != nil && prefer.Valid() == nil {

		log.Printf("create digitalocean ecs by using prefered region %s, instance type %s ...", prefer.RegionOrZone, prefer.InstanceType)

		req.Region = prefer.RegionOrZone
		req.Size = prefer.InstanceType

		created, err := sdk.createNode(req, suffix)
		if err != nil {
			return nil, nil, err
		}

		log.Printf("created prefered digitalocean ecs succeed: %s", created.ID)
		return created, prefer, nil
	}

	log.Infoln("creating digitalocean ecs by trying all regions & types ...")

	// if prefered created failed, or without prefer region & instance-type
	// try best on all region & instance-types to create the new digitalocean ecs
	var (
		created *cloudsvr.CloudNode
		err     error
	)

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
	for _, typ := range types {
		for _, region := range typ.Regions {

			req.Region = region
			req.Size = typ.Slug

			// if created succeed, directly return
			created, err = sdk.createNode(req, suffix)
			if err == nil {
				useRegion, useType = region, typ.Slug
				goto END
			}

			if sdk.isFatalError(err) {
				log.Errorf("create digitalocean ecs got fatal error, stop retry: %v", err)
				return nil, nil, err
			}

			log.Warnf("create digitalocean ecs failed: %v, will retry another region or type", err)
		}
	}

END:
	if err != nil {
		return nil, nil, err
	}

	log.Printf("created digitalocean ecs %s at %s and type is %s", created.ID, useRegion, useType)
	return created, &cloudsvr.PreferAttrs{RegionOrZone: useRegion, InstanceType: useType}, nil
}

// createNode actually create a new ecs with given region & instance-type
func (sdk *SDK) createNode(req *CreateInstancesInput, suffix string) (*cloudsvr.CloudNode, error) {
	var err error

	// create ssh key
	sshkeyID, privKey, err := sdk.CreateSSHKey(suffix)
	if err != nil {
		return nil, err
	}

	// if failed , clean up the newly created ssh key to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("digitalocean cloud node creation failed, clean up the newly created ssh key %d. [%v]", sshkeyID, err)
			sdk.RemoveSSHKey(sshkeyID)
		}
	}()

	// rewrite request to attach ssh key id
	req.SSHKeys = []int{sshkeyID}

	err = req.Validate()
	if err != nil {
		return nil, err
	}

	// create ecs
	ecsID, err := sdk.NewEcs(req)
	if err != nil {
		return nil, err // note: maybe *LnApiError, do NOT wrap any more
	}
	log.Printf("digitalocean ecs %d created", ecsID)

	// if failed , clean up the newly created ecs instance to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("digitalocean cloud node creation failed, clean up the newly created ecs instance %d. [%v]", ecsID, err)
			sdk.RemoveEcs(ecsID)
		}
	}()

	// wait ecs to be running
	err = sdk.WaitEcs(ecsID, "active", time.Second*120)
	if err != nil {
		return nil, fmt.Errorf("digitalocean ecs %d waitting to be running failed: %v", ecsID, err)
	}
	log.Printf("digitalocean ecs %d is running now", ecsID)

	// inspect ecs instance
	info, err := sdk.InspectEcs(ecsID)
	if err != nil {
		return nil, err
	}

	var ipaddr string
	if info.Networks != nil && len(info.Networks.V4) > 0 {
		ipaddr = info.Networks.V4[0].IPAddress
		log.Printf("digitalocean ecs %d got the public ip %s", ecsID, ipaddr)
	}

	var region string
	if info.Region != nil {
		region = info.Region.Slug
	}

	// return the final
	return &cloudsvr.CloudNode{
		ID:             strconv.Itoa(ecsID),
		RegionOrZoneID: region,
		InstanceType:   info.SizeSlug,
		CloudSvrType:   sdk.Type(),
		IPAddr:         ipaddr,
		Port:           "22",
		User:           "root",
		PrivKey:        privKey,
	}, nil
}
