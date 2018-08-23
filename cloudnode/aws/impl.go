// Package aws ...
// this file rewrap existing methods to implement the cloudsvr.Handler interface
package aws

import (
	"errors"
	"fmt"
	"math"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/bbklab/inf/pkg/cloudsvr"
	"github.com/bbklab/inf/pkg/ptype"
	"github.com/bbklab/inf/pkg/utils"
)

var (
	// CloudType is exported
	CloudType = "aws"
)

var (
	// OsImage is exported to make pass golint
	// OsImage = "ami-c60b90d1"  // no use, aws doesn't have a fixed ami for every regions

	// NodeName is exported to make pass golint
	NodeName = "inf-agent-aws-node"
	// NodeLabels is exported to make pass golint
	NodeLabels = map[string]string{cloudsvr.CLOUDFLAGKEY: CloudType}
	// SSHKeyNamePrefix is exported
	SSHKeyNamePrefix = "inf-agent-aws-sshkey"
	// SecurityGrouopNamePrefix is exported
	SecurityGrouopNamePrefix = "inf-agent-aws-servicegroup"
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
		name := ptype.StringV(reg.RegionName)
		location := name
		if regext, ok := regmap[name]; ok {
			location = regext.Location + " (" + regext.City + ")"
		}
		ret = append(ret, &cloudsvr.CloudRegion{
			ID:       name,
			Location: location,
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
	for name, typ := range types {
		ret = append(ret, &cloudsvr.CloudNodeType{
			ID:       name,
			Name:     "-",
			RegionID: "-",
			CPU:      fmt.Sprintf("%d", typ.CPU),
			Memory:   fmt.Sprintf("%0.1fGB", typ.Memory),
			Disk:     "-",
		})
	}

	return ret, nil
}

// ListNodes list all aws ecs instances with labels key: CLOUDFLAGKEY
// the nodes listed does NOT have any auth fields `User` `Password`
func (sdk *SDK) ListNodes() ([]*cloudsvr.CloudNode, error) {
	ecses, err := sdk.ListEcses("", NodeLabels)
	if err != nil {
		return nil, err
	}

	var ret []*cloudsvr.CloudNode
	for _, ecs := range ecses {
		if ptype.StringV(ecs.State.Name) == "terminated" {
			continue
		}
		ret = append(ret, &cloudsvr.CloudNode{
			ID:             ptype.StringV(ecs.InstanceId),
			RegionOrZoneID: regionOfZone(ptype.StringV(ecs.Placement.AvailabilityZone)),
			InstanceType:   ptype.StringV(ecs.InstanceType),
			CloudSvrType:   sdk.Type(),
			IPAddr:         ptype.StringV(ecs.PublicIpAddress),
			User:           "ec2-user", // Amazone Linux 2 AMI default login user
			CreatTime:      ecs.LaunchTime.Format(time.RFC3339),
			Status:         ptype.StringV(ecs.State.Name),
		})
	}

	return ret, nil
}

// InspectNode show details of one given ecs instance
func (sdk *SDK) InspectNode(id, regionOrZone string) (interface{}, error) {
	return sdk.InspectEcs(regionOrZone, id)
}

// RemoveNode remove aws ecs instance
func (sdk *SDK) RemoveNode(node *cloudsvr.CloudNode) error {
	var (
		region = node.RegionOrZoneID
		ecsID  = node.ID
	)

	if region == "" {
		return errors.New("aws node removal require region ID")
	}

	err := sdk.RemoveEcs(region, ecsID)
	if err != nil {
		log.Errorf("remove aws ecs %s error: %v", ecsID, err)
		return err
	}

	log.Printf("aws ecs %s removed", ecsID)
	return nil
}

// NewNode create aws ecs instance, try to use prefered attributes firstly
// Note: count of (each type of) instances are limited per-region / per-user
// See: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-resource-limits.html
func (sdk *SDK) NewNode(prefer *cloudsvr.PreferAttrs) (*cloudsvr.CloudNode, *cloudsvr.PreferAttrs, error) {
	var (
		req = &ec2.RunInstancesInput{
			MinCount:     ptype.Int64(1),
			MaxCount:     ptype.Int64(1),
			Monitoring:   &ec2.RunInstancesMonitoringEnabled{Enabled: ptype.Bool(false)},
			EbsOptimized: ptype.Bool(false),
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: ptype.String("instance"),
					Tags: []*ec2.Tag{
						{
							Key:   ptype.String(cloudsvr.CLOUDFLAGKEY),
							Value: ptype.String(CloudType),
						},
						{
							Key:   ptype.String("Name"), // aws reserved tag `Name` will be displayed as instance name
							Value: ptype.String(NodeName),
						},
					},
				},
			},
			// ImageId:          ptype.String(OsImage),                                // setup later, AMI image
			// Placement:        &ec2.Placement{AvailabilityZone: ptype.String(zone)}, // setup later, region & zone
			// KeyName:          ptype.String(keyname),                                // setup later, sshkey
			// InstanceType:     ptype.String(typ),                                    // setup later, instance type
			// SecurityGroupIds: ptype.StringSlice([]string{sgid}),                    // setup later, security group
		}
	)

	// if prefered attributes set, use prefer region & instance-type
	if prefer != nil && prefer.Valid() == nil {
		var (
			reg = prefer.RegionOrZone
			typ = prefer.InstanceType
		)
		log.Printf("create aws ecs by using prefered region %s, instance type %s ...", reg, typ)

		zone := sdk.PickupZone(reg)
		// img, _ := sdk.PickupCentos7AMI(reg) // nouse: use this region's centos7 x86_64
		img := regamimap[reg] // use Amazon Linux 2 AMI
		req.ImageId = ptype.String(img)
		req.Placement = &ec2.Placement{
			AvailabilityZone: ptype.String(zone),
		}
		req.InstanceType = ptype.String(typ)

		created, err := sdk.createNode(reg, req)
		if err != nil {
			return nil, nil, err
		}

		log.Printf("created prefered aws ecs succeed: %s", created.ID)
		return created, prefer, nil
	}

	log.Infoln("creating aws ecs by trying all regions & types ...")

	// if prefered created failed, or without prefer region & instance-type
	// try best on all region & instance-types to create the new aws ecs
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
	types, err := sdk.ListInstanceTypes(1, 4, 1, 4) // TODO range of given cpus/mems ranges
	if err != nil {
		log.Errorf("sdk.NewNode.ListInstanceTypes() error: %v", err)
		return nil, nil, err
	}

	var (
		useRegionID, useInsType string
	)
	// range all regions & types to try to create ecs instance
	for _, reg := range regions {
		regid := ptype.StringV(reg.RegionName)

		for typid := range types {

			zone := sdk.PickupZone(regid)
			// img, _ := sdk.PickupCentos7AMI(regid) // nouse: use this region's centos7 x86_64
			img := regamimap[regid] // use Amazon Linux 2 AMI
			req.ImageId = ptype.String(img)
			req.Placement = &ec2.Placement{
				AvailabilityZone: ptype.String(zone),
			}
			req.InstanceType = ptype.String(typid)

			// if created succeed, directly return
			created, err = sdk.createNode(regid, req)
			if err == nil {
				useRegionID, useInsType = regid, typid
				goto END
			}

			if sdk.isFatalError(err) {
				log.Errorf("create aws ecs got fatal error, stop retry: %v", err)
				return nil, nil, err
			}

			log.Warnf("create aws ecs failed: %v, will retry another region or type", err)
			time.Sleep(time.Millisecond * 200)
		}
	}

END:
	if err != nil {
		return nil, nil, err
	}

	log.Printf("created aws ecs %s at %s and type is %s", created.ID, useRegionID, useInsType)
	return created, &cloudsvr.PreferAttrs{RegionOrZone: useRegionID, InstanceType: useInsType}, nil
}

// createNode actually create a new ecs with given region & instance-type
func (sdk *SDK) createNode(region string, req *ec2.RunInstancesInput) (*cloudsvr.CloudNode, error) {
	var err error

	var suffix = utils.RandomString(10)

	// create sshkey pair firstly
	keyname, priv, err := sdk.CreateSSHKey(region, suffix)
	if err != nil {
		return nil, err
	}

	// if failed , clean up the newly created ssh key to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("aws cloud node creation failed: [%v], clean up the newly created ssh key %s", err, keyname)
			sdk.RemoveSSHKey(region, keyname)
		}
	}()

	// create security group firstly
	sgid, err := sdk.CreateSecurityGroup(region, suffix)
	if err != nil {
		return nil, err
	}

	// if failed , clean up the newly created security group to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("aws cloud node creation failed: [%v], clean up the newly created security group %s", err, sgid)
			sdk.RemoveSecurityGroup(region, sgid)
		}
	}()

	// attach keyname onto the request
	req.KeyName = ptype.String(keyname)
	// attach security group id onto the request
	req.SecurityGroupIds = ptype.StringSlice([]string{sgid})

	// create ecs instance
	ecsID, err := sdk.NewEcs(region, req)
	if err != nil {
		return nil, fmt.Errorf("aws ecs create failed: %v", err)
	}
	log.Printf("aws ecs %s created at %s", ecsID, region)

	// if create succeed, but other operations failed, clean up the newly created ecs instance to prevent garbage left
	defer func() {
		if err != nil {
			log.Warnf("aws cloud node creation failed, clean up the newly created ecs instance %s. [%v]", ecsID, err)
			sdk.RemoveNode(&cloudsvr.CloudNode{ID: ecsID, RegionOrZoneID: region})
		}
	}()

	// wait ecs to be running
	err = sdk.WaitEcs(region, ecsID, "running", time.Second*300)
	if err != nil {
		return nil, fmt.Errorf("aws ecs %s waitting to be running failed: %v", ecsID, err)
	}
	log.Printf("aws ecs %s is Running now", ecsID)

	// get ecs public ip address
	info, err := sdk.InspectEcs(region, ecsID)
	if err != nil {
		return nil, fmt.Errorf("pick up the newly created aws ecs %s failed: %v", ecsID, err)
	}
	ip := ptype.StringV(info.PublicIpAddress)

	return &cloudsvr.CloudNode{
		ID:             ecsID,
		RegionOrZoneID: region,
		CloudSvrType:   sdk.Type(),
		IPAddr:         ip,
		Port:           "22",
		User:           "ec2-user", // this depends on the AMI, maybe `ubuntu` or `ec2-user`
		PrivKey:        priv,
	}, nil
}
