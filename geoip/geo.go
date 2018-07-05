package geoip

import (
	"fmt"
	"net"
	"strings"

	geoip2 "github.com/oschwald/geoip2-golang"
	maxminddb "github.com/oschwald/maxminddb-golang"
)

// default geo files
var (
	fgeocity = "/usr/share/GeoIP/GeoLite2-City.mmdb"
	fgeoasn  = "/usr/share/GeoIP/GeoLite2-ASN.mmdb"
)

// default update url
// see more: https://dev.maxmind.com/geoip/geoip2/geolite2/
var (
	cityupdate = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.tar.gz"
	asnupdate  = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-ASN.tar.gz"
)

// Handler represents geo data interface
type Handler interface {
	GetGeoInfo(addr, lang string) *GeoInfo // query geo info for given address & lang["de","en","es","fr","ja","pt-BR","ru","zh-CN"]

	Metadata() map[string]maxminddb.Metadata // current geo data info (version,epoch,size,lang,etc)

	Update() error // update geo data
}

// GeoInfo is exported
type GeoInfo struct {
	IP          string `json:"ip" bson:"ip"`
	Continent   string `json:"continent" bson:"continent"`
	Country     string `json:"country" bson:"country"`
	CountryISO  string `json:"country_iso" bson:"country_iso"`
	City        string `json:"city" bson:"city"`
	TimeZone    string `json:"timezone" bson:"timezone"`
	Orgnization string `json:"orgnization" bson:"orgnization"`
}

// Geo is an implemention of Handler interface
type Geo struct {
	fcity string
	fasn  string
	rcity *geoip2.Reader // city reader
	rasn  *geoip2.Reader // asn reader
}

// NewGeo is exported
func NewGeo(fcity, fasn string) (*Geo, error) {
	if fcity == "" {
		fcity = fgeocity
	}
	if fasn == "" {
		fasn = fgeoasn
	}

	g := &Geo{
		fcity: fcity,
		fasn:  fasn,
	}

	var err error
	g.rcity, err = geoip2.Open(g.fcity)
	if err != nil {
		return nil, err
	}

	g.rasn, err = geoip2.Open(g.fasn)
	if err != nil {
		return nil, err
	}

	return g, nil
}

// GetGeoInfo implement Handler interface
// note: always return a non-nil *GeoInfo
func (g *Geo) GetGeoInfo(addr, lang string) (info *GeoInfo) {
	if lang == "" {
		lang = "en-US"
	}

	var (
		ip  = normalize(addr)
		ipv = net.ParseIP(ip)
	)

	info = &GeoInfo{
		IP: ip,
	}

	record, err := g.rcity.City(ipv)
	if err != nil {
		return
	}

	var subdivision string
	if len(record.Subdivisions) > 0 { // prevent panic
		subdivision = record.Subdivisions[0].Names[lang]
	}

	info.Continent = record.Continent.Names[lang]
	info.Country = record.Country.Names[lang]
	info.CountryISO = record.Country.IsoCode
	info.City = record.City.Names[lang]
	if info.City == "" {
		info.City = subdivision
	} else {
		if subdivision != "" {
			info.City += fmt.Sprintf(",%s", subdivision)
		}
	}
	info.TimeZone = record.Location.TimeZone

	// asn db
	asn, err := g.rasn.ASN(ipv)
	if err != nil {
		return
	}
	info.Orgnization = asn.AutonomousSystemOrganization

	return
}

// Metadata implement Handler interface
func (g *Geo) Metadata() map[string]maxminddb.Metadata {
	return map[string]maxminddb.Metadata{
		"city": g.rcity.Metadata(),
		"asn":  g.rasn.Metadata(),
	}
}

func normalize(addr string) string {
	fields := strings.SplitN(addr, ":", 2)
	if len(fields) != 2 {
		return addr
	}
	return fields[0]
}
