package digitalocean

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/bbklab/inf/pkg/ssh"
)

//
// Ecs Instance
//

// NewEcs create a new ecs instance with given settings
func (sdk *SDK) NewEcs(req *CreateInstancesInput) (int, error) {
	var resp DropletOne
	err := sdk.apiCall("POST", "/droplets", req, &resp)
	if err != nil {
		return -1, err
	}
	return resp.Data.ID, err
}

// ListEcses show all of ecs instances
func (sdk *SDK) ListEcses() ([]*Droplet, error) {
	var resp Droplets
	err := sdk.apiCall("GET", "/droplets?per_page=10000", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Datas, nil
}

// InspectEcs show details of a given ecs instance
func (sdk *SDK) InspectEcs(id int) (*Droplet, error) {
	var resp *DropletOne
	err := sdk.apiCall("GET", fmt.Sprintf("/droplets/%d", id), nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// RemoveEcs remove the specified ecs instance
func (sdk *SDK) RemoveEcs(id int) error {
	// inspect ecs firstly to get the name suffix
	info, err := sdk.InspectEcs(id)
	if err != nil {
		return err
	}
	var suffix = strings.TrimPrefix(info.Name, NodeNamePrefix+"-")

	// remove ecs
	err = sdk.apiCall("DELETE", fmt.Sprintf("/droplets/%d", id), nil, nil)
	if err != nil {
		return err
	}

	// remove corresponding sshkey
	sshkey, err := sdk.SearchSSHKey(suffix)
	if err != nil {
		return nil
	}
	sdk.RemoveSSHKey(sshkey.ID)
	return nil
}

// WaitEcs wait ecs instance status reached to expected status until maxWait timeout
func (sdk *SDK) WaitEcs(id int, expectStatus string, maxWait time.Duration) error {
	timeout := time.After(maxWait)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("wait ecs instance %d -> %s timeout in %s", id, expectStatus, maxWait)

		case <-ticker.C:
			info, err := sdk.InspectEcs(id)
			if err != nil {
				return fmt.Errorf("WaitEcs.InspectEcs %d error: %v", id, err)
			}
			logrus.Printf("digitalocean ecs instance %d is %s ...", id, info.Status)
			if info.Status == expectStatus {
				return nil
			}
		}
	}
}

// ListRegions show all of regions digitalocean supported
func (sdk *SDK) ListRegions() ([]*Region, error) {
	var resp Regions
	err := sdk.apiCall("GET", "/regions", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Datas, nil
}

// ListInstanceTypes show all of instance types under given zone
// support cpu / memory minimal/maximize filter parameters
func (sdk *SDK) ListInstanceTypes(minCPU, maxCPU, minMem, maxMem int) ([]*Size, error) {
	var resp Sizes
	err := sdk.apiCall("GET", "/sizes", nil, &resp)
	if err != nil {
		return nil, err
	}

	var ret []*Size
	for _, size := range resp.Datas {
		var cpus, mems = size.VCpus, size.Memory
		if cpus <= maxCPU && cpus >= minCPU && mems <= maxMem && mems >= minMem {
			ret = append(ret, size)
		}
	}

	return ret, nil
}

// ListImages show all of images digitalocean supported
func (sdk *SDK) ListImages() ([]*Image, error) {
	var resp Images
	err := sdk.apiCall("GET", "/images", nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Datas, nil
}

// CreateSSHKey create a new ssh key
func (sdk *SDK) CreateSSHKey(suffix string) (int, string, error) {
	// generate ssh key pairs
	priv, pub, err := ssh.GenSSHKeypair()
	if err != nil {
		return -1, "", err
	}

	// fmt.Println("RPIV --->", string(priv))
	// fmt.Println("PUB  --->", string(pub))

	var (
		created *SSHKeyOne
		req     = &CreateSSHKeyInput{
			Name:      fmt.Sprintf("%s-%s", SSHKeyNamePrefix, suffix),
			PublicKey: string(pub),
		}
	)
	err = sdk.apiCall("POST", "/account/keys", req, &created)
	if err != nil {
		return -1, "", err
	}
	return created.Data.ID, string(priv), nil
}

// SearchSSHKey search exists ssh key
func (sdk *SDK) SearchSSHKey(suffix string) (*SSHKey, error) {
	var resp SSHKeys
	err := sdk.apiCall("GET", "/account/keys", nil, &resp)
	if err != nil {
		return nil, err
	}
	for _, sshkey := range resp.Datas {
		if sshkey.Name == fmt.Sprintf("%s-%s", SSHKeyNamePrefix, suffix) {
			return sshkey, nil
		}
	}
	return nil, errors.New("no sshkeys has the suffix: " + suffix)
}

// RemoveSSHKey remove ssh key
func (sdk *SDK) RemoveSSHKey(id int) error {
	return sdk.apiCall("DELETE", fmt.Sprintf("/account/keys/%d", id), nil, nil)
}
