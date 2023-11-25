package gateway

import "strings"

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

type NetRoute struct {
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

type netstatParser struct {
	state      netstatParserState
	net4Data   []NetRoute
	net6Data   []NetRoute
	net4Fields map[string]int
	net6Fields map[string]int
}

func (n *netstatParser) feed(line string) {
	line = strings.TrimSpace(line)

	switch n.state {
	case netstatParserStateHeader:
		n.parseHeader(line)
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
}

func (n *netstatParser) reset() {
	n.state = netstatParserStateHeader
	clear(n.net4Data)
	clear(n.net6Data)
	n.net4Data = n.net6Data[:0]
	n.net6Data = n.net6Data[:0]
	clear(n.net4Fields)
	clear(n.net6Fields)
}

func (n *netstatParser) parseHeader(line string) {
	if strings.ToLower(line) == "routing tables" {
		n.state = netstatParserStateInternetHeader
	}
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
	n.net4Data = append(n.net4Data, NetRoute{
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
	n.net6Data = append(n.net6Data, NetRoute{
		Destination: fields[n.net6Fields[nsDestination]],
		Flags:       fields[n.net6Fields[nsFlags]],
		Netif:       fields[n.net6Fields[nsNetif]],
		Gateway:     fields[n.net6Fields[nsGateway]],
	})
}
