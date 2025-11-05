package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Root struct {
	CI *CI `yaml:"ci"`
}

type CI struct {
	AuthorCompany      *string              `yaml:"author-company" json:"authorCompany"`
	AuthorDepartment   *string              `yaml:"author-department" json:"authorDepartment"`
	Classification     *string              `yaml:"classification" json:"classification"`
	Versions           []*Version           `yaml:"versions" json:"versions"`
	AuditVersions      []*AuditVersion      `yaml:"audit-versions" json:"auditVersions"`
	ReleaseVersions    []*ReleaseVersion    `yaml:"release-versions" json:"releaseVersions"`
	Requirements       []*Requirement       `yaml:"requirements" json:"requirements"`
	SurroundingSystems []*SurroundingSystem `yaml:"surrounding-systems" json:"surroundingSystems"`
	Description        *Description         `yaml:"description" json:"description"`
	Configuration      *Configuration       `yaml:"configuration" json:"configuration"`
	Interfaces         []*Interface         `yaml:"interfaces" json:"interfaces"`
	Accounts           []*Account           `yaml:"accounts" json:"accounts"`
}

type Account struct {
	Type  *string `yaml:"type" json:"type"`
	Name  *string `yaml:"name" json:"name"`
	Usage *string `yaml:"usage" json:"usage"`
}

type Version struct {
	Number      *string `yaml:"number" json:"name"`
	Date        *string `yaml:"date" json:"date"`
	User        *string `yaml:"user" json:"user"`
	Description *string `yaml:"description" json:"description"`
}

type AuditVersion struct {
	Number    *string `yaml:"number"`
	Date      *string `yaml:"date"`
	Authority *string `yaml:"authority"`
	Remarks   *string `yaml:"remarks"`
}

type ReleaseVersion struct {
	Number    *string `yaml:"number" json:"number"`
	Date      *string `yaml:"date" json:"date"`
	Authority *string `yaml:"authority" json:"authority"`
	Remarks   *string `yaml:"remarks" json:"remarks"`
}

type Requirement struct {
	Type *string `yaml:"type" json:"type"`
	Name *string `yaml:"name" json:"name"`
}

type SurroundingSystem struct {
	Type        *string `yaml:"type" json:"type"`
	Name        *string `yaml:"name" json:"name"`
	Address     *string `yaml:"address" json:"address"`
	Description *string `yaml:"description" json:"description"`
}

type Description struct {
	ServiceCode *string `yaml:"service-code" json:"serviceCode"`
	Customer    *string `yaml:"customer" json:"customer"`
	Descr       *string `yaml:"description" json:"description"`
	Supplier    *string `yaml:"supplier" json:"supplier"`
	DisasterLvl *int    `yaml:"disaster-lvl" json:"disasterLvl"`
}

type Configuration struct {
	Name   *string   `yaml:"name" json:"name"`
	FQDN   *string   `yaml:"fqdn" json:"fqdn"`
	OS     *string   `yaml:"os" json:"os"`
	RAM    *int      `yaml:"ram" json:"ram"`
	CPU    *int      `yaml:"cpu" json:"cpu"`
	Domain *string   `yaml:"domain" json:"domain"`
	NTP    []*string `yaml:"ntp" json:"ntp"`
	SNMP   *string   `yaml:"snmp" json:"snmp"`
}

type Interface struct {
	Name   *string   `yaml:"name" json:"name"`
	Zone   *string   `yaml:"zone" json:"zone"`
	VLAN   *int      `yaml:"vlan" json:"vlan"`
	DHCP   *bool     `yaml:"dhcp" json:"dhcp"`
	IP     *string   `yaml:"ip" json:"ip"`
	Subnet *string   `yaml:"subnet" json:"subnet"`
	DNS    []*string `yaml:"dns" json:"dns"`
}

func (r *Root) Validate() error {
	if r == nil || r.CI == nil {
		return nil
	}
	return r.CI.Validate()
}

func (c *CI) Validate() error {
	var me MultiError

	for i, v := range c.Versions {
		if v == nil {
			continue
		}
		if err := v.Validate(fmt.Sprintf("ci.versions[%d]", i)); err != nil {
			me.add("", err)
		}
	}
	for i, v := range c.AuditVersions {
		if v == nil {
			continue
		}
		if err := v.Validate(fmt.Sprintf("ci.audit_versions[%d]", i)); err != nil {
			me.add("", err)
		}
	}
	for i, v := range c.ReleaseVersions {
		if v == nil {
			continue
		}
		if err := v.Validate(fmt.Sprintf("ci.release_versions[%d]", i)); err != nil {
			me.add("", err)
		}
	}
	for i, req := range c.Requirements {
		if req == nil {
			continue
		}
		if err := req.Validate(fmt.Sprintf("ci.requirements[%d]", i)); err != nil {
			me.add("", err)
		}
	}
	for i, s := range c.SurroundingSystems {
		if s == nil {
			continue
		}
		if err := s.Validate(fmt.Sprintf("ci.surrounding_systems[%d]", i)); err != nil {
			me.add("", err)
		}
	}
	if c.Description != nil {
		if err := c.Description.Validate("ci.description"); err != nil {
			me.add("", err)
		}
	}
	if c.Configuration != nil {
		if err := c.Configuration.Validate("ci.configuration"); err != nil {
			me.add("", err)
		}
	}
	for i, in := range c.Interfaces {
		if in == nil {
			continue
		}
		if err := in.Validate(fmt.Sprintf("ci.interfaces[%d]", i)); err != nil {
			me.add("", err)
		}
	}
	return me.ToError()
}

func (v *Version) Validate(path string) error {
	var me MultiError
	me.add(path+".number", validateVersionNumber(path+".number", v.Number))
	me.add(path+".date", validateDate(path+".date", v.Date))
	return me.ToError()
}

func (v *AuditVersion) Validate(path string) error {
	var me MultiError
	me.add(path+".number", validateVersionNumber(path+".number", v.Number))
	me.add(path+".date", validateDate(path+".date", v.Date))
	return me.ToError()
}

func (v *ReleaseVersion) Validate(path string) error {
	var me MultiError
	me.add(path+".number", validateVersionNumber(path+".number", v.Number))
	me.add(path+".date", validateDate(path+".date", v.Date))
	return me.ToError()
}

func (r *Requirement) Validate(_ string) error {
	return nil
}

func (s *SurroundingSystem) Validate(path string) error {
	var me MultiError
	me.add(path+".address", validateHostnameOrIP(path+".address", s.Address))
	return me.ToError()
}

func (d *Description) Validate(path string) error {
	var me MultiError
	me.add(path+".disaster_lvl", validateNonNegativeInt(path+".disaster_lvl", d.DisasterLvl))
	return me.ToError()
}

func (c *Configuration) Validate(path string) error {
	var me MultiError
	me.add(path+".fqdn", validateFQDN(path+".fqdn", c.FQDN))
	me.add(path+".ram", validateNonNegativeInt(path+".ram", c.RAM))
	me.add(path+".cpu", validateNonNegativeInt(path+".cpu", c.CPU))
	me.add(path+".ntp", validateIPList(path+".ntp", c.NTP))
	return me.ToError()
}

func (i *Interface) Validate(path string) error {
	var me MultiError
	me.add(path+".vlan", validateVLAN(path+".vlan", i.VLAN))
	me.add(path+".ip", validateIP(path+".ip", i.IP))
	me.add(path+".subnet", validateSubnetMask(path+".subnet", i.Subnet))
	me.add(path+".dns", validateIPList(path+".dns", i.DNS))

	if i.DHCP != nil && !boolOrFalse(i.DHCP) {
		if isEmpty(i.IP) && isEmpty(i.Subnet) {
			_ = errors.New("dhcp=false but ip/subnet not provided")
		}
	}

	if s := strOrEmpty(i.Subnet); strings.HasPrefix(s, "/") && len(s) > 1 {
		if n, err := strconv.Atoi(strings.TrimPrefix(s, "/")); err == nil {
			if n < 0 || n > 32 {
				me.add(path+".subnet", fmt.Errorf("subnet prefix out of range /%d", n))
			}
		}
	}

	return me.ToError()
}

func DecodeYaml(r io.Reader) (*Root, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var root Root
	if err := yaml.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	return &root, nil
}

func DecodeJson(r io.Reader) (*Root, error) {

	var root Root
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}

	return &root, nil
}
