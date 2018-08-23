package aws

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/bbklab/inf/pkg/ptype"
	"github.com/bbklab/inf/pkg/ssh"
)

//
// Ecs Instance
//

// NewEcs create a new ecs instance with given settings
func (sdk *SDK) NewEcs(region string, req *ec2.RunInstancesInput) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	ins, err := sdk.SwitchRegion(region).RunInstancesWithContext(ctx, req)
	if err != nil {
		return "", err
	}

	if len(ins.Instances) == 0 {
		return "", errors.New("NewEcs() got abnormal response, at least one instance should exists")
	}

	return ptype.StringV(ins.Instances[0].InstanceId), nil
}

// RemoveEcs remove the specified ecs instance
func (sdk *SDK) RemoveEcs(region, ecsID string) error {
	// remove ecs related sshkey pair
	info, err := sdk.InspectEcs(region, ecsID)
	if err != nil {
		return err
	}

	err = sdk.RemoveSSHKey(region, ptype.StringV(info.KeyName))
	if err != nil {
		return err
	}
	logrus.Infof("remove aws ecs %s related sshkey %s", ecsID, ptype.StringV(info.KeyName))

	// remove ecs instance
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, err = sdk.SwitchRegion(region).TerminateInstancesWithContext(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: ptype.StringSlice([]string{ecsID}),
	})
	if err != nil {
		return err
	}
	err = sdk.WaitEcs(region, ecsID, "terminated", time.Second*120)
	if err != nil {
		return err
	}
	logrus.Infof("terminated aws ecs %s", ecsID)

	// remove ecs related security group
	if sgs := info.SecurityGroups; len(sgs) > 0 {
		err = sdk.RemoveSecurityGroup(region, ptype.StringV(sgs[0].GroupId))
		if err != nil {
			return err
		}
		logrus.Infof("remove aws ecs %s related security group %s", ecsID, ptype.StringV(sgs[0].GroupId))
	}

	return nil
}

// InspectEcs show details of a given ecs instance
func (sdk *SDK) InspectEcs(region, ecsID string) (*ec2.Instance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	resp, err := sdk.SwitchRegion(region).DescribeInstancesWithContext(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			ptype.String(ecsID),
		},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Reservations) == 0 {
		return nil, fmt.Errorf("no such Ecs %s found", ecsID)
	}

	if len(resp.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("no such Ecs %s found", ecsID)
	}

	return resp.Reservations[0].Instances[0], nil
}

// WaitEcs wait ecs instance status reached to expected status until maxWait timeout
// expectStatus could be:
//  0 : pending
// 16 : running
// 32 : shutting-down
// 48 : terminated
// 64 : stopping
// 80 : stopped
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
			if info.State == nil {
				continue
			}
			status := ptype.StringV(info.State.Name)
			logrus.Printf("aws ecs instance %s is %s ...", ecsID, status)
			if status == expectStatus {
				return nil
			}
		}
	}
}

// ListEcses show all of ecs instances for
// a specified region (regionID parameter set) or
// all of regions (regionID parameter empty)
func (sdk *SDK) ListEcses(region string, lbs map[string]string) ([]*ec2.Instance, error) {

	// tag key-val filter
	var filters = make([]*ec2.Filter, 0, 0)
	for key, val := range lbs {
		filters = append(filters, &ec2.Filter{
			Name:   ptype.String("tag:" + key),
			Values: ptype.StringSlice([]string{val}),
		})
	}

	var input = &ec2.DescribeInstancesInput{
		MaxResults: ptype.Int64(1000),
	}
	if len(filters) > 0 {
		input.Filters = filters
	}

	// query on one region
	if region != "" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		resp, err := sdk.SwitchRegion(region).DescribeInstancesWithContext(ctx, input)
		if err != nil {
			return nil, err
		}
		if len(resp.Reservations) == 0 {
			return nil, nil
		}
		var ret []*ec2.Instance
		for _, res := range resp.Reservations {
			ret = append(ret, res.Instances...)
		}
		return ret, nil
	}

	// query on all regions
	regions, err := sdk.ListRegions()
	if err != nil {
		return nil, err
	}

	// query all by concurrency
	var (
		ret = make([]*ec2.Instance, 0, 0)
		l   sync.Mutex
		wg  sync.WaitGroup
	)

	wg.Add(len(regions))
	for _, reg := range regions {
		region := ptype.StringV(reg.RegionName)
		go func(region string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()
			resp, err := sdk.SwitchRegion(region).DescribeInstancesWithContext(ctx, input)
			if err != nil {
				return
			}
			if len(resp.Reservations) == 0 {
				return
			}
			l.Lock()
			for _, res := range resp.Reservations {
				ret = append(ret, res.Instances...)
			}
			l.Unlock()
		}(region)
	}
	wg.Wait()
	return ret, nil
}

// ListRegions show all of regions aws supported
func (sdk *SDK) ListRegions() ([]*ec2.Region, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	resp, err := sdk.SwitchRegion("").DescribeRegionsWithContext(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return nil, err
	}
	return resp.Regions, nil
}

// ListInstanceTypes show all of instance types aws supported
// support cpu / memory minimal/maximize filter parameters
func (sdk *SDK) ListInstanceTypes(minCPU, maxCPU, minMem, maxMem int) (map[string]*InstanceTypeSpec, error) {
	var ret = map[string]*InstanceTypeSpec{}
	for name, spec := range itmap {
		var cpus, mems = int64(spec.CPU), int64(spec.Memory)
		if cpus <= int64(maxCPU) && cpus >= int64(minCPU) && mems <= int64(maxMem) && mems >= int64(minMem) {
			ret[name] = spec
		}
	}
	return ret, nil
}

//
// Region Zones
//

// ListZones list available zones under one given region
func (sdk *SDK) ListZones(region string) ([]*ec2.AvailabilityZone, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	resp, err := sdk.SwitchRegion(region).DescribeAvailabilityZonesWithContext(ctx, &ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return nil, err
	}

	return resp.AvailabilityZones, nil
}

// PickupZone pick up the first avaliable zone under a specified region
func (sdk *SDK) PickupZone(region string) string {
	defaultZone := fmt.Sprintf("%sa", region) // pick up the first one

	zones, err := sdk.ListZones(region)
	if err != nil {
		return defaultZone
	}

	for _, zone := range zones {
		if strings.ToLower(ptype.StringV(zone.State)) == "available" {
			return ptype.StringV(zone.ZoneName)
		}
	}

	return defaultZone
}

//
// Region AMI
//

// actually we're using centos product code to search aws ami market currently
// owner: aws-marketplace
// product_code: aw0evgkw8e5c1q413zgy5pjce
// eg:
// aws ec2 describe-images \
//    --owners 'aws-marketplace' \
//    --filters 'Name=product-code,Values=aw0evgkw8e5c1q413zgy5pjce' \
//    --query 'sort_by(Images, &CreationDate)[-1].[ImageId]' \
//    --output 'text'

// Deprecated
// Offical CentOS is being distributed as a marketplace AMI rather than
// a community AMI, so you can't use it through API, it will lead to errors like:
// In order to use this AWS Marketplace product you need to accept terms and subscribe.
// blabla ...
//
// use centos7 offical aws product code to filter the offical centos7 x86_64 AMIs
// See: https://wiki.centos.org/Cloud/AWS

// PickupCentos7AMI pick up the first avaliable centos7 AMI under a specified region
func (sdk *SDK) PickupCentos7AMI(region string) (string, error) {

	var (
		centos7OfficalProductCode = "aw0evgkw8e5c1q413zgy5pjce" // Deprecated
	)

	amifilters := []*ec2.Filter{
		{
			Name:   ptype.String("product-code"),
			Values: ptype.StringSlice([]string{centos7OfficalProductCode}),
		},
		{
			Name:   ptype.String("state"),
			Values: ptype.StringSlice([]string{"available"}),
		},
		{
			Name:   ptype.String("image-type"),
			Values: ptype.StringSlice([]string{"machine"}),
		},
		{
			Name:   ptype.String("architecture"),
			Values: ptype.StringSlice([]string{"x86_64"}),
		},
		{
			Name:   ptype.String("virtualization-type"),
			Values: ptype.StringSlice([]string{"hvm"}),
		},
		{
			Name:   ptype.String("is-public"),
			Values: ptype.StringSlice([]string{"true"}),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	res, err := sdk.SwitchRegion(region).DescribeImagesWithContext(ctx, &ec2.DescribeImagesInput{
		ExecutableUsers: ptype.StringSlice([]string{"all"}),
		Filters:         amifilters,
	})
	if err != nil {
		return "", err
	}

	if len(res.Images) == 0 {
		return "", fmt.Errorf("region %s: no CentOS 7 AMI (product-code=%s) available", region, centos7OfficalProductCode)
	}

	// sort by CreationDate, use the newest one
	sort.Sort(ImageSorter(res.Images))

	return ptype.StringV(res.Images[0].ImageId), nil

	// no need to filter the AMI description text cause the product code
	/*
		var (
			imgs7 = []*ec2.Image{}
		)
		for _, img := range res.Images {
			desc := strings.ToLower(ptype.StringV(img.Description))
			if ok, _ := regexp.MatchString(`centos.*7`, desc); ok {
				imgs7 = append(imgs7, img)
			}
		}

		if len(imgs7) == 0 {
			return "", fmt.Errorf("region %s: no CentOS 7 AMI available", region)
		}
	*/
}

//
// SecurityGroup
//

// CreateSecurityGroup is exported
func (sdk *SDK) CreateSecurityGroup(region, suffix string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	var name = fmt.Sprintf("%s-%s", SecurityGrouopNamePrefix, suffix)
	groupResp, err := sdk.SwitchRegion(region).CreateSecurityGroupWithContext(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   ptype.String(name),
		Description: ptype.String("inf agent aws security group"),
	})
	if err != nil {
		return "", err
	}

	var sgid = groupResp.GroupId

	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel2()

	_, err = sdk.SwitchRegion(region).AuthorizeSecurityGroupIngressWithContext(ctx2, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: sgid,
		IpPermissions: []*ec2.IpPermission{ // allow all
			{
				IpProtocol: ptype.String("-1"),                                  // all
				FromPort:   ptype.Int64(-1),                                     // all
				ToPort:     ptype.Int64(-1),                                     // all
				IpRanges:   []*ec2.IpRange{{CidrIp: ptype.String("0.0.0.0/0")}}, // all
				Ipv6Ranges: []*ec2.Ipv6Range{{CidrIpv6: ptype.String("::/0")}},  // all
			},
		},
	})
	return ptype.StringV(sgid), err
}

// RemoveSecurityGroup is exported
func (sdk *SDK) RemoveSecurityGroup(region, sgid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, err := sdk.SwitchRegion(region).DeleteSecurityGroupWithContext(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId: ptype.String(sgid),
	})
	return err
}

//
// SSH Key Pair
//

// CreateSSHKey import a new ssh key on given aws region
func (sdk *SDK) CreateSSHKey(region, suffix string) (string, string, error) {
	var name = fmt.Sprintf("%s-%s", SSHKeyNamePrefix, suffix)

	// generate ssh key pairs
	priv, pub, err := ssh.GenSSHKeypair()
	if err != nil {
		return "", "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, err = sdk.SwitchRegion(region).ImportKeyPairWithContext(ctx, &ec2.ImportKeyPairInput{
		KeyName:           ptype.String(name),
		PublicKeyMaterial: pub,
	})
	if err != nil {
		return "", "", err
	}

	return name, string(priv), nil
}

// RemoveSSHKey remove a given ssh key on given aws region
func (sdk *SDK) RemoveSSHKey(region, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, err := sdk.SwitchRegion(region).DeleteKeyPairWithContext(ctx, &ec2.DeleteKeyPairInput{
		KeyName: ptype.String(name),
	})
	return err
}
