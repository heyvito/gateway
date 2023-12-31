package gateway

import (
	"strings"
)

const (
	nsDestination = "Destination"
	nsFlags       = "Flags"
	nsNetif       = "Netif"
	nsGateway     = "Gateway"
	nsInterface   = "Interface"
)

type netstatParserState int

const (
	netstatParserStateHeader netstatParserState = iota
	netstatParserStateInternetHeader
	netstatParserStateInternet4Header
	netstatParserStateInternet4Data
	netstatParserStateInternet6Header
	netstatParserStateInternet6Data
)

type netstatParser struct {
	state      netstatParserState
	netData    NetRouteList
	net4Fields map[string]int
	net6Fields map[string]int
}

func (n *netstatParser) feed(line string) error {
	line = strings.TrimSpace(line)

	switch n.state {
	case netstatParserStateHeader:
		return n.parseHeader(line)
	case netstatParserStateInternetHeader:
		n.parseInternetHeader(line)

	case netstatParserStateInternet4Header:
		n.parseInternetHeader4(line)
	case netstatParserStateInternet4Data:
		n.parseInternet4Data(line)

	case netstatParserStateInternet6Header:
		n.parseInternetHeader6(line)
	case netstatParserStateInternet6Data:
		n.parseInternet6Data(line)
	}

	return nil
}

func (n *netstatParser) reset() {
	n.state = netstatParserStateHeader
	clear(n.netData)
	n.netData = n.netData[:0]
	clear(n.net4Fields)
	clear(n.net6Fields)
}

func (n *netstatParser) parseHeader(line string) error {
	if strings.ToLower(line) == "routing tables" {
		n.state = netstatParserStateInternetHeader
		return nil
	}

	return &ErrCantParse{}
}

func (n *netstatParser) parseInternetHeader(line string) {
	if len(line) == 0 {
		return
	}

	line = strings.ToLower(line)
	switch line {
	case "internet:":
		n.state = netstatParserStateInternet4Header
	case "internet6:":
		n.state = netstatParserStateInternet6Header
	default:
		n.reset()
	}
}

func (n *netstatParser) parseInternetHeader4(line string) {
	fields := fieldSet(strings.Fields(line))
	if len(fields) < 4 {
		n.reset()
		return
	}

	wantedFields := []string{nsDestination, nsGateway, nsFlags}
	for _, v := range wantedFields {
		idx := fields.fieldIdx(v)
		if idx == -1 {
			n.reset()
			return
		}
		n.net4Fields[v] = idx
	}

	iface, netif := fields.fieldIdx(nsInterface), fields.fieldIdx(nsNetif)
	if iface == -1 && netif == -1 {
		n.reset()
		return
	}

	if iface > 0 {
		// NetBSD
		n.net4Fields[nsNetif] = iface
	} else {
		// Other BSD (Solaris, Darwin...)
		n.net4Fields[nsNetif] = netif
	}

	n.state = netstatParserStateInternet4Data
}

func (n *netstatParser) parseInternet4Data(line string) {
	if len(line) == 0 {
		n.state = netstatParserStateInternetHeader
		return
	}

	fields := strings.Fields(line)
	n.netData = append(n.netData, NetRoute{
		Kind:        NetRouteKindV4,
		Destination: fields[n.net4Fields[nsDestination]],
		Flags:       fields[n.net4Fields[nsFlags]],
		Netif:       fields[n.net4Fields[nsNetif]],
		Gateway:     fields[n.net4Fields[nsGateway]],
	})
}

func (n *netstatParser) parseInternetHeader6(line string) {
	fields := fieldSet(strings.Fields(line))
	if len(fields) < 4 {
		n.reset()
		return
	}

	wantedFields := []string{nsDestination, nsGateway, nsFlags}
	for _, v := range wantedFields {
		idx := fields.fieldIdx(v)
		if idx == -1 {
			n.reset()
			return
		}
		n.net6Fields[v] = idx
	}

	iface, netif := fields.fieldIdx(nsInterface), fields.fieldIdx(nsNetif)
	if iface == -1 && netif == -1 {
		n.reset()
		return
	}

	if iface > 0 {
		// NetBSD
		n.net6Fields[nsNetif] = iface
	} else {
		// Other BSD (Solaris, Darwin...)
		n.net6Fields[nsNetif] = netif
	}

	n.state = netstatParserStateInternet6Data
}

func (n *netstatParser) parseInternet6Data(line string) {
	if len(line) == 0 {
		n.state = netstatParserStateInternetHeader
		return
	}

	fields := strings.Fields(line)
	n.netData = append(n.netData, NetRoute{
		Kind:        NetRouteKindV6,
		Destination: fields[n.net6Fields[nsDestination]],
		Flags:       fields[n.net6Fields[nsFlags]],
		Netif:       fields[n.net6Fields[nsNetif]],
		Gateway:     fields[n.net6Fields[nsGateway]],
	})
}

func (n *netstatParser) result() NetRouteList {
	newList := make(NetRouteList, len(n.netData))
	for i, v := range n.netData {
		newList[i] = v
	}
	return newList
}

func newNetstatParser() *netstatParser {
	return &netstatParser{
		state:      netstatParserStateHeader,
		netData:    nil,
		net4Fields: map[string]int{},
		net6Fields: map[string]int{},
	}
}
