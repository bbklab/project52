package vultr

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

//
// Ecs Instance
//

// NewEcs create a new ecs instance with given settings
func (sdk *SDK) NewEcs(req *CreateServerInput) (int, error) {
	if err := req.Validate(); err != nil {
		return -1, err
	}

	var (
		params = req.ToURLValues()
		resp   map[string]string
	)
	err := sdk.apiCall("POST", "/server/create", params, &resp)
	if err != nil {
		return -1, err
	}

	id := resp["SUBID"]
	idN, err := strconv.Atoi(id)
	if err != nil {
		return -1, fmt.Errorf("abnormal response ecsid [%s]", id)
	}
	return idN, nil
}

// ListEcses show all of ecs instances
func (sdk *SDK) ListEcses() (map[string]*Server, error) {
	var resp map[string]*Server
	err := sdk.apiCall("GET", "/server/list", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// InspectEcs show details of a given ecs instance
func (sdk *SDK) InspectEcs(id int) (*Server, error) {
	var resp *Server
	err := sdk.apiCall("GET", fmt.Sprintf("/server/list?SUBID=%d", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// RemoveEcs remove the specified ecs instance
func (sdk *SDK) RemoveEcs(id int) error {
	vals := url.Values{"SUBID": {strconv.Itoa(id)}}
	err := sdk.apiCall("POST", "/server/destroy", vals, nil)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "cannot be destroyed within 5 minutes of being created") {
		log.Warnf("can't destroy ecs currently, delay removal later ...")
		time.AfterFunc(time.Second*10, func() {
			sdk.RemoveEcs(id)
		})
		return nil
	}
	return err
}

// WaitEcs wait ecs instance status reached to expected status until maxWait timeout
// note: vultr server status should be wait for at least 3 consecutive status
func (sdk *SDK) WaitEcs(id int, expectStatus string, maxWait time.Duration) error {
	timeout := time.After(maxWait)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	var cntConsecutive int

	for {
		select {
		case <-timeout:
			return fmt.Errorf("wait ecs instance %d -> %s timeout in %s", id, expectStatus, maxWait)

		case <-ticker.C:
			info, err := sdk.InspectEcs(id)
			if err != nil {
				return fmt.Errorf("WaitEcs.InspectEcs %d error: %v", id, err)
			}
			log.Printf("vultr ecs instance %d current status=%s, power=%s, state=%s ...", id, info.Status, info.PowerStatus, info.ServerState)
			if info.Status == expectStatus && info.PowerStatus == "running" { // power == running & status == active
				cntConsecutive++
			} else {
				cntConsecutive = 0
			}

			if cntConsecutive >= 3 {
				return nil
			}
		}
	}
}

// ListRegions show all of regions vultr supported
// note: public api, do NOT require auth
func (sdk *SDK) ListRegions() (map[string]*Region, error) {
	var resp map[string]*Region
	err := sdk.apiCall("GET", "/regions/list", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp, err
}

// ListPlans show all of instance plans
// support cpu / memory minimal/maximize filter parameters
// note: public api, do NOT require auth
func (sdk *SDK) ListPlans(minCPU, maxCPU, minMem, maxMem int) (map[string]*Plan, error) {
	var resp map[string]*Plan
	err := sdk.apiCall("GET", "/plans/list", nil, &resp)
	if err != nil {
		return nil, err
	}

	var ret = make(map[string]*Plan)
	for id, plan := range resp {
		var cpus, mems = plan.VCpus, plan.RAM
		if cpus <= maxCPU && cpus >= minCPU && mems <= maxMem && mems >= minMem {
			ret[id] = plan
		}
	}

	return ret, nil
}
