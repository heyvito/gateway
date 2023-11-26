package gateway

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
)

type NetRouteKind uint8

const (
	NetRouteKindV4 NetRouteKind = iota + 1
	NetRouteKindV6
)

type NetRoute struct {
	Kind        NetRouteKind
	Destination string
	Flags       string
	Netif       string
	Gateway     string
}

func (n NetRoute) HasFlags(flags ...string) bool {
	for _, v := range flags {
		if !strings.Contains(n.Flags, v) {
			return false
		}
	}

	return true
}

type NetRouteList []NetRoute

func (n NetRouteList) FindDefaults(kind NetRouteKind) []NetRoute {
	var filter func(r *NetRoute) bool
	if kind == NetRouteKindV4 {
		filter = func(r *NetRoute) bool {
			return (r.Destination == "default" || r.Destination == "0.0.0.0") &&
				r.HasFlags("U", "G") &&
				!r.HasFlags("H")
		}
	} else if kind == NetRouteKindV6 {
		filter = func(r *NetRoute) bool {
			return (r.Destination == "::/0" || r.Destination == "default") &&
				r.HasFlags("U", "G") &&
				!r.HasFlags("H") &&
				r.Gateway != "fe80::" &&
				!strings.HasPrefix(r.Gateway, "fe80::%")
		}
	} else {
		panic(fmt.Sprintf("Invalid NetRouteKind %d", kind))
	}

	var result []NetRoute

	for _, v := range n {
		if v.Kind == kind && filter(&v) {
			result = append(result, v)
		}
	}

	return result
}

// FindDefaultGateways returns a list of addresses of all gateways used by
// default routes, both IPv4 and IPv6.
func FindDefaultGateways() ([]netip.Addr, error) {
	routes, err := getRoutes()
	if err != nil {
		return nil, err
	}
	var ips []netip.Addr
	if rs := routes.FindDefaults(NetRouteKindV4); len(rs) > 0 {
		for _, r := range rs {
			v, err := netip.ParseAddr(r.Gateway)
			if err != nil {
				return nil, err
			}
			ips = append(ips, v)
		}
	}
	if rs := routes.FindDefaults(NetRouteKindV6); len(rs) > 0 {
		for _, r := range rs {
			v, err := netip.ParseAddr(r.Gateway)
			if err != nil {
				return nil, err
			}
			ips = append(ips, v)
		}
	}

	return ips, nil
}

// FindDefaultInterfaces returns a slice of strings containing the name of
// interfaces using a default gateway.
func FindDefaultInterfaces() ([]string, error) {
	routes, err := getRoutes()
	if err != nil {
		return nil, err
	}
	var ifsMap []string
	if rs := routes.FindDefaults(NetRouteKindV4); rs != nil {
		for _, r := range rs {
			_, err := netip.ParseAddr(r.Gateway)
			if err != nil {
				return nil, err
			}
			ifsMap = append(ifsMap, r.Netif)
		}
	}
	if rs := routes.FindDefaults(NetRouteKindV6); rs != nil {
		for _, r := range rs {
			_, err := netip.ParseAddr(r.Gateway)
			if err != nil {
				return nil, err
			}
			ifsMap = append(ifsMap, r.Netif)
		}
	}
	return unique(ifsMap), nil
}

// PickDefaultInterface picks the interface with most IPs based on the result of
// FindDefaultInterfaces.
func PickDefaultInterface() (string, error) {
	ifaces, err := FindDefaultInterfaces()
	if err != nil {
		return "", err
	}

	ipCount := map[string]int{}

	for _, name := range ifaces {
		iface, err := net.InterfaceByName(name)
		if err != nil {
			return "", err
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		ipCount[name] = len(addrs)
	}

	maxLen := 0
	maxName := ""
	for k, v := range ipCount {
		if v > maxLen {
			maxLen = v
			maxName = k
		}
	}

	return maxName, nil
}

// FindDefaultIPs returns a list of IPs associated to all interfaces using a
// default gateway.
func FindDefaultIPs() ([]netip.Addr, error) {
	interfaces, err := FindDefaultInterfaces()
	if err != nil {
		return nil, err
	}
	var out []netip.Addr

	for _, ifaceName := range interfaces {
		iface, err := net.InterfaceByName(ifaceName)
		if err != nil {
			return nil, err
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, v := range addrs {
			ip := v.(*net.IPNet).IP
			if ip4 := ip.To4(); ip4 != nil {
				ip = ip4
			}
			if add, ok := netip.AddrFromSlice(ip); ok {
				out = append(out, add)
			}
		}
	}

	return out, nil
}

var getRoutes func() (NetRouteList, error) = nil
