package protocol

import (
	"net"
)

// LocalAddr returns a local net.Addr for binding, from sourceIP and/or interface name.
// If sourceIP is set it is used as the IP; if only iface is set the first IP of that interface is used.
// Returns nil if both are empty. For "tcp"/"tcp4"/"tcp6" returns *net.TCPAddr; for "udp"/"udp4"/"udp6" returns *net.UDPAddr.
func LocalAddr(network, sourceIP, iface string) (net.Addr, error) {
	var ip net.IP
	if sourceIP != "" {
		ip = net.ParseIP(sourceIP)
		if ip == nil {
			return nil, nil
		}
	} else if iface != "" {
		ifc, err := net.InterfaceByName(iface)
		if err != nil {
			return nil, err
		}
		addrs, err := ifc.Addrs()
		if err != nil || len(addrs) == 0 {
			return nil, nil
		}
		want4 := network == "tcp4" || network == "udp4"
		want6 := network == "tcp6" || network == "udp6"
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				cand := v.IP
				if cand.IsLoopback() {
					continue
				}
				if want4 && cand.To4() != nil {
					ip = cand.To4()
					break
				}
				if want6 && cand.To4() == nil {
					ip = cand
					break
				}
				if !want4 && !want6 && ip == nil {
					ip = cand
				}
			}
			if ip != nil && (want4 || want6) {
				break
			}
		}
		if ip == nil && !want4 && !want6 && len(addrs) > 0 {
			if v, ok := addrs[0].(*net.IPNet); ok {
				ip = v.IP
			}
		}
		if ip == nil {
			return nil, nil
		}
	} else {
		return nil, nil
	}
	switch network {
	case "tcp", "tcp4", "tcp6":
		return &net.TCPAddr{IP: ip, Port: 0}, nil
	case "udp", "udp4", "udp6":
		return &net.UDPAddr{IP: ip, Port: 0}, nil
	default:
		return nil, nil
	}
}
