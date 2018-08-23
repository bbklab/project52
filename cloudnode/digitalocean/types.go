package digitalocean

import "errors"

// ErrorResponse is exported
type ErrorResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// Account is exported
type Account struct {
	Data struct {
		DropletLimit    int    `json:"droplet_limit"`
		FloatingIPLimit int    `json:"floating_ip_limit"`
		Email           string `json:"email"`
		UUID            string `json:"uuid"`
		EmailVerified   bool   `json:"email_verified"`
		Status          string `json:"status"`
		StatusMessage   string `json:"status_message"`
	} `json:"account"`
}

//
// Regions
//

// Regions is exported
type Regions struct {
	Datas []*Region `json:"regions"`
}

// Region is exported
type Region struct {
	Slug      string   `json:"slug"` // uniq
	Name      string   `json:"name"`
	Sizes     []string `json:"sizes"`
	Features  []string `json:"features"`
	Available bool     `json:"available"`
}

//
// Image
//

// Images is exported
type Images struct {
	Datas []*Image `json:"images"`
}

// Image is exported
type Image struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Distribution string   `json:"distribution"`
	Slug         string   `json:"slug"`
	Public       bool     `json:"public"`
	Regions      []string `json:"regions"`
	MinDiskSize  int      `json:"min_disk_size"`
	Created      string   `json:"created_at"`
}

//
// SSHKey
//

// SSHKeys is exported
type SSHKeys struct {
	Datas []*SSHKey `json:"ssh_keys"`
}

// SSHKeyOne is exported
type SSHKeyOne struct {
	Data *SSHKey `json:"ssh_key"`
}

// SSHKey is exported
type SSHKey struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	FingerPrint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
}

// CreateSSHKeyInput is exported
type CreateSSHKeyInput struct {
	Name      string `json:"name"`       // must
	PublicKey string `json:"public_key"` // must
}

//
// Size
//

// Sizes is exported
type Sizes struct {
	Datas []*Size `json:"sizes"`
}

// Size is exported
type Size struct {
	Slug         string   `json:"slug"`   // uniq
	Memory       int      `json:"memory"` // by MiB
	VCpus        int      `json:"vcpus"`
	Disk         int      `json:"disk"`          // by GiB
	Transfer     float64  `json:"transfer"`      // by TiB
	PriceMonthly float64  `json:"price_monthly"` // by $
	PriceHourly  float64  `json:"price_hourly"`  // by $
	Regions      []string `json:"regions"`
}

//
// Droplet
//

// Droplets is exported
type Droplets struct {
	Datas []*Droplet `json:"droplets"`
}

// DropletOne is exported
type DropletOne struct {
	Data *Droplet `json:"droplet"`
}

// Droplet is exported
type Droplet struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Memory      int       `json:"memory"`
	Vcpus       int       `json:"vcpus"`
	Disk        int       `json:"disk"`
	Region      *Region   `json:"region"`
	Image       *Image    `json:"image"`
	Size        *Size     `json:"size"`
	SizeSlug    string    `json:"size_slug"`
	BackupIDs   []int     `json:"backup_ids"`
	SnapshotIDs []int     `json:"snapshot_ids"`
	Features    []string  `json:"features"`
	Locked      bool      `json:"locked"`
	Status      string    `json:"status"` // new active off archive
	Networks    *Networks `json:"networks"`
	Created     string    `json:"created_at"`
	Kernel      *Kernel   `json:"kernel"`
	Tags        []string  `json:"tags"`
	VolumeIDs   []string  `json:"volume_ids"`
	// NextBackupWindow *BackupWindow `json:"next_backup_window"`
}

// Networks is exported
type Networks struct {
	V4 []NetworkV4 `json:"v4"`
	V6 []NetworkV6 `json:"v6"`
}

// NetworkV4 is exported
type NetworkV4 struct {
	IPAddress string `json:"ip_address"`
	Netmask   string `json:"netmask"`
	Gateway   string `json:"gateway"`
	Type      string `json:"type"`
}

// NetworkV6 is exported
type NetworkV6 struct {
	IPAddress string `json:"ip_address"`
	Netmask   int    `json:"netmask"`
	Gateway   string `json:"gateway"`
	Type      string `json:"type"`
}

// Kernel is exported
type Kernel struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// CreateInstancesInput is exported
type CreateInstancesInput struct {
	Name              string   `json:"name"`     // must
	Region            string   `json:"region"`   // must
	Size              string   `json:"size"`     // must
	Image             int      `json:"image"`    // must
	SSHKeys           []int    `json:"ssh_keys"` // required
	Backups           bool     `json:"backups"`
	IPv6              bool     `json:"ipv6"`
	PrivateNetworking bool     `json:"private_networking"`
	Monitoring        bool     `json:"monitoring"`
	UserData          string   `json:"user_data"`
	Volumes           []string `json:"volumes"`
	Tags              []string `json:"tags"`
}

// Validate is exported
func (req *CreateInstancesInput) Validate() error {
	if req.Name == "" {
		return errors.New("name required")
	}
	if req.Region == "" {
		return errors.New("region required")
	}
	if req.Size == "" {
		return errors.New("size required")
	}
	if req.Image == 0 {
		return errors.New("image required")
	}
	if len(req.SSHKeys) == 0 {
		return errors.New("sshkeys required")
	}
	return nil
}
