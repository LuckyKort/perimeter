package config

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"
)

// ParseIPOrNet accepts "1.2.3.4" or "1.2.3.0/24".
func ParseIPOrNet(s string) (netip.Prefix, error) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "/") {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return netip.Prefix{}, fmt.Errorf("%q - не IP и не подсеть (пример: 1.2.3.4 или 1.2.3.0/24)", s)
		}
		return p.Masked(), nil
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("%q - не IP и не подсеть (пример: 1.2.3.4 или 1.2.3.0/24)", s)
	}
	return netip.PrefixFrom(a, a.BitLen()), nil
}

// IsEmpty reports whether nothing is allowed yet (first run).
func (a Allow) IsEmpty() bool { return len(a.IPs) == 0 && len(a.Countries) == 0 }

// Matches reports whether ip (with its country code, possibly unknown "") is allowed.
func (a Allow) Matches(ip, country string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, s := range a.IPs {
		if p, err := ParseIPOrNet(s); err == nil && p.Contains(addr) {
			return true
		}
	}
	return country != "" && slices.Contains(a.Countries, strings.ToUpper(country))
}

func (a *Allow) AddIP(ip string) bool {
	if slices.Contains(a.IPs, ip) {
		return false
	}
	a.IPs = append(a.IPs, ip)
	return true
}

func (a *Allow) AddCountry(cc string) bool {
	cc = strings.ToUpper(cc)
	if cc == "" || slices.Contains(a.Countries, cc) {
		return false
	}
	a.Countries = append(a.Countries, cc)
	return true
}

func (a *Allow) RemoveIP(ip string) {
	a.IPs = slices.DeleteFunc(a.IPs, func(s string) bool { return s == ip })
}
func (a *Allow) RemoveCountry(cc string) {
	a.Countries = slices.DeleteFunc(a.Countries, func(s string) bool { return s == cc })
}
