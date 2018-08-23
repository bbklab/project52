package tencent

import (
	"errors"
	"fmt"
	"strings"

	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"

	"github.com/bbklab/inf/pkg/orderparam"
	"github.com/bbklab/inf/pkg/ptype"
)

func validateRunInstanceReq(req *cvm.RunInstancesRequest) error {
	if req.Placement == nil {
		return errors.New("Placement options required")
	}
	if req.InternetAccessible == nil {
		return errors.New("InternetAccessible options required")
	}
	return nil
}

func mergeRunInstanceReq(params *orderparam.Params, req *cvm.RunInstancesRequest) {
	params.Set("InstanceChargeType", ptype.StringV(req.InstanceChargeType))
	params.Set("Placement.Zone", ptype.StringV(req.Placement.Zone))
	params.Set("InstanceType", ptype.StringV(req.InstanceType))
	params.Set("ImageId", ptype.StringV(req.ImageId))
	params.Set("InternetAccessible.InternetChargeType", ptype.StringV(req.InternetAccessible.InternetChargeType))
	params.Set("InternetAccessible.InternetMaxBandwidthOut", fmt.Sprintf("%d", ptype.Int64V(req.InternetAccessible.InternetMaxBandwidthOut)))
	params.Set("InternetAccessible.PublicIpAssigned", fmt.Sprintf("%t", ptype.BoolV(req.InternetAccessible.PublicIpAssigned)))

	params.SetIgnoreNull("InstanceCount", fmt.Sprintf("%d", ptype.Int64V(req.InstanceCount)))
	params.SetIgnoreNull("InstanceName", ptype.StringV(req.InstanceName))
	params.SetIgnoreNull("LoginSettings.Password", ptype.StringV(req.LoginSettings.Password))
	params.SetIgnoreNull("EnhancedService.SecurityService.Enabled", fmt.Sprintf("%t", ptype.BoolV(req.EnhancedService.SecurityService.Enabled)))
	params.SetIgnoreNull("EnhancedService.MonitorService.Enabled", fmt.Sprintf("%t", ptype.BoolV(req.EnhancedService.MonitorService.Enabled)))
	params.SetIgnoreNull("ClientToken", ptype.StringV(req.ClientToken))

	for idx, spec := range req.TagSpecification {
		params.Set(fmt.Sprintf("TagSpecification.%d.ResourceType", idx), ptype.StringV(spec.ResourceType))
		for tidx, tag := range spec.Tags {
			params.Set(fmt.Sprintf("TagSpecification.%d.Tags.%d.Key", idx, tidx), ptype.StringV(tag.Key))
			params.Set(fmt.Sprintf("TagSpecification.%d.Tags.%d.Value", idx, tidx), ptype.StringV(tag.Value))
		}
	}
}

// tencent zone is always be the form as "region-idx"
// such as:  ap-guangzhou-19   ap-hongkong-2
// trim the zone suffix -[0-9] then we got the region
func regionOfZone(zone string) string {
	zone = strings.TrimRightFunc(zone, func(r rune) bool {
		return r >= '0' && r <= '9'
	})
	return strings.TrimSuffix(zone, "-")
}
