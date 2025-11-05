package pkg

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

type ValidationError struct {
	Path string
	Err  error
}

func (ve ValidationError) Error() string {
	if ve.Path == "" {
		return ve.Err.Error()
	}
	return fmt.Sprintf("%s: %v", ve.Path, ve.Err)
}

func (ve ValidationError) Unwrap() error { return ve.Err }

type MultiError []ValidationError

func (m MultiError) Error() string {
	if len(m) == 0 {
		return ""
	}
	var b strings.Builder
	for i, e := range m {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(e.Error())
	}
	return b.String()
}

func (m *MultiError) add(path string, err error) {
	if err == nil {
		return
	}
	*m = append(*m, ValidationError{Path: path, Err: err})
}

func (m MultiError) ToError() error {
	if len(m) == 0 {
		return nil
	}
	return m
}

const dateLayout = "02.01.2006"

var (
	reVersionNumber    = regexp.MustCompile(`^\d+(?:\.\d+)*$`)
	reHostnameLabel    = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	reSubnetPrefixOnly = regexp.MustCompile(`^/(?:[0-9]|[12][0-9]|3[0-2])$`)
)

func strOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func boolOrFalse(p *bool) bool {
	return p != nil && *p
}

func isEmpty(p *string) bool { return p == nil || strings.TrimSpace(*p) == "" }

func validateDate(_ string, p *string) error { // path not needed for inner check
	s := strOrEmpty(p)
	if s == "" {
		return nil
	}
	if _, err := time.Parse(dateLayout, s); err != nil {
		return fmt.Errorf("invalid date %q (expected %s)", s, dateLayout)
	}
	return nil
}

func validateVersionNumber(_ string, p *string) error {
	s := strOrEmpty(p)
	if s == "" {
		return nil
	}
	if !reVersionNumber.MatchString(s) {
		return fmt.Errorf("invalid version number %q (expected digits with optional dots, e.g. 1.2.3)", s)
	}
	return nil
}

func validateNonNegativeInt(_ string, p *int) error {
	if p == nil {
		return nil
	}
	if *p < 0 {
		return fmt.Errorf("must be ≥ 0, got %d", *p)
	}
	return nil
}

func validateVLAN(_ string, p *int) error {
	if p == nil {
		return nil
	}
	if *p < 1 || *p > 4094 {
		return fmt.Errorf("VLAN must be in [1..4094], got %d", *p)
	}
	return nil
}

func validateIP(_ string, p *string) error {
	s := strOrEmpty(p)
	if s == "" {
		return nil
	}
	if net.ParseIP(s) == nil {
		return fmt.Errorf("invalid IP address %q", s)
	}
	return nil
}

func validateIPList(path string, list []*string) error {
	var me MultiError
	seen := map[string]struct{}{}
	for i, sp := range list {
		if sp == nil || strings.TrimSpace(*sp) == "" {
			continue
		}
		v := strings.TrimSpace(*sp)
		if net.ParseIP(v) == nil {
			me.add(fmt.Sprintf("%s[%d]", path, i), fmt.Errorf("invalid IP address %q", v))
			continue
		}
		if _, dup := seen[v]; dup {
			me.add(fmt.Sprintf("%s[%d]", path, i), fmt.Errorf("duplicate IP %q", v))
		} else {
			seen[v] = struct{}{}
		}
	}
	return me.ToError()
}

func validateFQDN(_ string, p *string) error {
	s := strOrEmpty(p)
	if s == "" {
		return nil
	}
	labels := strings.Split(s, ".")
	if len(labels) < 2 {
		return fmt.Errorf("invalid FQDN %q (need at least one dot)", s)
	}
	for _, lbl := range labels {
		if len(lbl) == 0 || len(lbl) > 63 || !reHostnameLabel.MatchString(lbl) {
			return fmt.Errorf("invalid FQDN %q (bad label %q)", s, lbl)
		}
	}
	if len(s) > 253 {
		return fmt.Errorf("invalid FQDN %q (too long)", s)
	}
	return nil
}

// Accepts either "/N" prefix or dotted IPv4 mask like "255.255.255.0".
func validateSubnetMask(_ string, p *string) error {
	s := strOrEmpty(p)
	if s == "" {
		return nil
	}
	if reSubnetPrefixOnly.MatchString(s) {
		return nil
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return fmt.Errorf("invalid subnet %q (use /N or dotted mask)", s)
	}
	ip = ip.To4()
	if ip == nil {
		return fmt.Errorf("invalid IPv4 mask %q", s)
	}
	mask := net.IPMask(ip)
	if !isContiguousIPv4Mask(mask) {
		return fmt.Errorf("invalid dotted netmask %q (must be contiguous)", s)
	}
	return nil
}

// isContiguousIPv4Mask checks mask has 1*0* bit pattern.
func isContiguousIPv4Mask(m net.IPMask) bool {
	if len(m) != 4 {
		return false
	}
	// Convert mask to 32-bit value
	v := uint32(m[0])<<24 | uint32(m[1])<<16 | uint32(m[2])<<8 | uint32(m[3])
	if v == 0 { // /0 is valid
		return true
	}
	// Contiguous if v has form 111..1100..00, i.e., v & (v+1) == 0xFFFFFFFF << ones becomes 0
	// A simpler check: v & (v + 1) == 0 when inverted (for 111..11 pattern). Use standard trick:
	inv := ^v
	// inv should be 000..0011..11 (zeros then ones). Check it is of the form x & (x+1) == 0
	return inv&(inv+1) == 0
}

func validateHostnameOrIP(_ string, p *string) error {
	s := strOrEmpty(p)
	if s == "" {
		return nil
	}
	if net.ParseIP(s) != nil {
		return nil
	}
	labels := strings.Split(s, ".")
	for _, lbl := range labels {
		if len(lbl) == 0 || len(lbl) > 63 || !reHostnameLabel.MatchString(lbl) {
			return fmt.Errorf("invalid host or IP %q", s)
		}
	}
	if len(s) > 253 {
		return fmt.Errorf("invalid host %q (too long)", s)
	}
	return nil
}
