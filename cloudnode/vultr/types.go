// Package vultr ...
// Most borrowed from:
//   https://github.com/JamesClonk/vultr/blob/master/lib/servers.go
//
package vultr

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

//
// Account
//

// Account is exported
type Account struct {
	Balance           float64 `json:"balance,string"`
	PendingCharges    float64 `json:"pending_charges,string"`
	LastPaymentDate   string  `json:"last_payment_date"`
	LastPaymentAmount float64 `json:"last_payment_amount,string"`
}

//
// Region
//

// Region is exported
type Region struct {
	ID           int    `json:"DCID,string"`
	Name         string `json:"name"`
	Country      string `json:"country"`
	Continent    string `json:"continent"`
	State        string `json:"state"`
	Ddos         bool   `json:"ddos_protection"`
	BlockStorage bool   `json:"block_storage"`
	Code         string `json:"regioncode"`
}

//
// Instance Types
//

// Plan is exported
type Plan struct {
	ID        int     `json:"VPSPLANID,string"`
	Name      string  `json:"name"` // Starter, Basic ...
	VCpus     int     `json:"vcpu_count,string"`
	RAM       int     `json:"ram,string"`             // by MiB
	Disk      int     `json:"disk,string"`            // by GiB
	Bandwidth float64 `json:"bandwidth,string"`       // by MiB
	Price     float64 `json:"price_per_month,string"` // by $
	Type      string  `json:"plan_type"`              // SSD SATA
	Regions   []int   `json:"available_locations"`
}

//
// Server
//

// CreateServerInput is exported
// See: https://www.vultr.com/api/#server_create
type CreateServerInput struct {
	DCID                 int // must
	VPSPLANID            int // must
	OSID                 int // must
	ISOID                string
	SCRIPTID             string
	SNAPSHOTID           string
	EnableIPv6           bool
	EnablePrivateNetwork bool
	NETWORKID            string
	Label                string // label text
	SSHKEYID             string
	AutoBackups          bool
	APPID                string
	Userdata             string
	NotifyActivate       bool
	DdosProtection       bool
	ReservedIPv4         string
	Hostname             string
	Tag                  string
	FIREWALLGROUPID      string
}

// Validate is exported
func (req *CreateServerInput) Validate() error {
	if req.DCID == 0 {
		return errors.New("region (DCID) required")
	}
	if req.VPSPLANID == 0 {
		return errors.New("vps plan (VPSPLANID) required")
	}
	if req.OSID == 0 {
		return errors.New("os (OSID) required")
	}
	if n := len(req.Label); n < 3 || n > 32 {
		return errors.New("label length must between [3-32]")
	}
	return nil
}

// ToURLValues is exported
// TODO more ...
func (req *CreateServerInput) ToURLValues() url.Values {
	val := url.Values{}
	val.Set("DCID", strconv.Itoa(req.DCID))
	val.Set("VPSPLANID", strconv.Itoa(req.VPSPLANID))
	val.Set("OSID", strconv.Itoa(req.OSID))
	val.Set("enable_ipv6", formatBool(req.EnableIPv6))
	val.Set("enable_private_network", formatBool(req.EnablePrivateNetwork))
	val.Set("auto_backups", formatBool(req.AutoBackups))
	val.Set("notify_activate", formatBool(req.NotifyActivate))
	val.Set("ddos_protection", formatBool(req.DdosProtection))
	if req.Label != "" {
		val.Set("label", req.Label)
	}
	return val
}

func formatBool(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

// Server is exported
type Server struct {
	ID               string      `json:"SUBID"`
	Label            string      `json:"label"`
	OS               string      `json:"os"`
	RAM              string      `json:"ram"`
	Disk             string      `json:"disk"`
	MainIP           string      `json:"main_ip"`
	VCpus            int         `json:"vcpu_count,string"`
	Location         string      `json:"location"`
	RegionID         int         `json:"DCID,string"`
	DefaultPassword  string      `json:"default_password"`
	Created          string      `json:"date_created"`
	PendingCharges   float64     `json:"pending_charges"`
	Status           string      `json:"status"` // pending | active | suspended | closed
	Cost             string      `json:"cost_per_month"`
	CurrentBandwidth float64     `json:"current_bandwidth_gb"`
	AllowedBandwidth float64     `json:"allowed_bandwidth_gb,string"`
	NetmaskV4        string      `json:"netmask_v4"`
	GatewayV4        string      `json:"gateway_v4"`
	PowerStatus      string      `json:"power_status"` // running, stopped ...
	ServerState      string      `json:"server_state"` // none | locked | installingbooting | isomounting | ok
	PlanID           int         `json:"VPSPLANID,string"`
	V6Networks       []V6Network `json:"v6_networks"`
	InternalIP       string      `json:"internal_ip"`
	KVMUrl           string      `json:"kvm_url"`
	AutoBackups      string      `json:"auto_backups"`
	Tag              string      `json:"tag"`
	OSID             string      `json:"OSID"`
	AppID            string      `json:"APPID"`
	FirewallGroupID  string      `json:"FIREWALLGROUPID"`
}

// V6Network represents a IPv6 network of a Vultr server
type V6Network struct {
	Network     string `json:"v6_network"`
	MainIP      string `json:"v6_main_ip"`
	NetworkSize string `json:"v6_network_size"`
}

// UnmarshalJSON implements json.Unmarshaller on Server.
// This is needed because the Vultr API is inconsistent in it's JSON responses for servers.
// Some fields can change type, from JSON number to JSON string and vice-versa.
func (s *Server) UnmarshalJSON(data []byte) (err error) {
	var (
		valueNil = "<nil>"
	)

	if s == nil {
		*s = Server{}
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	value := fmt.Sprintf("%v", fields["vcpu_count"])
	if len(value) == 0 || value == valueNil {
		value = "0"
	}
	vcpu, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	s.VCpus = int(vcpu)

	value = fmt.Sprintf("%v", fields["DCID"])
	if len(value) == 0 || value == valueNil {
		value = "0"
	}
	region, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	s.RegionID = int(region)

	value = fmt.Sprintf("%v", fields["VPSPLANID"])
	if len(value) == 0 || value == valueNil {
		value = "0"
	}
	plan, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	s.PlanID = int(plan)

	value = fmt.Sprintf("%v", fields["pending_charges"])
	if len(value) == 0 || value == valueNil {
		value = "0"
	}
	pc, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	s.PendingCharges = pc

	value = fmt.Sprintf("%v", fields["current_bandwidth_gb"])
	if len(value) == 0 || value == valueNil {
		value = "0"
	}
	cb, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	s.CurrentBandwidth = cb

	value = fmt.Sprintf("%v", fields["allowed_bandwidth_gb"])
	if len(value) == 0 || value == valueNil {
		value = "0"
	}
	ab, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	s.AllowedBandwidth = ab

	value = fmt.Sprintf("%v", fields["OSID"])
	if value == valueNil {
		value = ""
	}
	s.OSID = value

	value = fmt.Sprintf("%v", fields["APPID"])
	if value == valueNil || value == "0" {
		value = ""
	}
	s.AppID = value

	value = fmt.Sprintf("%v", fields["FIREWALLGROUPID"])
	if value == valueNil || value == "0" {
		value = ""
	}
	s.FirewallGroupID = value

	s.ID = fmt.Sprintf("%v", fields["SUBID"])
	s.Label = fmt.Sprintf("%v", fields["label"])
	s.OS = fmt.Sprintf("%v", fields["os"])
	s.RAM = fmt.Sprintf("%v", fields["ram"])
	s.Disk = fmt.Sprintf("%v", fields["disk"])
	s.MainIP = fmt.Sprintf("%v", fields["main_ip"])
	s.Location = fmt.Sprintf("%v", fields["location"])
	s.DefaultPassword = fmt.Sprintf("%v", fields["default_password"])
	s.Created = fmt.Sprintf("%v", fields["date_created"])
	s.Status = fmt.Sprintf("%v", fields["status"])
	s.Cost = fmt.Sprintf("%v", fields["cost_per_month"])
	s.NetmaskV4 = fmt.Sprintf("%v", fields["netmask_v4"])
	s.GatewayV4 = fmt.Sprintf("%v", fields["gateway_v4"])
	s.PowerStatus = fmt.Sprintf("%v", fields["power_status"])
	s.ServerState = fmt.Sprintf("%v", fields["server_state"])

	v6networks := make([]V6Network, 0)
	if networks, ok := fields["v6_networks"].([]interface{}); ok {
		for _, network := range networks {
			if network, ok := network.(map[string]interface{}); ok {
				v6network := V6Network{
					Network:     fmt.Sprintf("%v", network["v6_network"]),
					MainIP:      fmt.Sprintf("%v", network["v6_main_ip"]),
					NetworkSize: fmt.Sprintf("%v", network["v6_network_size"]),
				}
				v6networks = append(v6networks, v6network)
			}
		}
		s.V6Networks = v6networks
	}

	s.InternalIP = fmt.Sprintf("%v", fields["internal_ip"])
	s.KVMUrl = fmt.Sprintf("%v", fields["kvm_url"])
	s.AutoBackups = fmt.Sprintf("%v", fields["auto_backups"])
	s.Tag = fmt.Sprintf("%v", fields["tag"])

	return
}
