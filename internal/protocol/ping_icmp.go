package protocol

import (
	"context"
	"errors"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// Pinger sends a single ICMP echo request and waits for reply, returning RTT.
type Pinger interface {
	Ping(ctx context.Context) (time.Duration, bool, error)
}

// ICMPPinger implements Pinger over ICMP for IPv4/IPv6.
type ICMPPinger struct {
	Addr      string
	IPVersion string // "ipv4" or "ipv6"
	Timeout   time.Duration
	Size      int
}

func (p *ICMPPinger) Ping(ctx context.Context) (time.Duration, bool, error) {
	if p.Timeout <= 0 {
		p.Timeout = 3 * time.Second
	}
	size := p.Size
	if size <= 0 {
		size = 56
	}

	ipAddr, err := net.ResolveIPAddr("ip", p.Addr)
	if err != nil {
		return 0, false, err
	}

	var network string
	var icmpType icmp.Type

	if ipAddr.IP.To4() != nil || p.IPVersion == "ipv4" {
		network = "ip4:icmp"
		icmpType = ipv4.ICMPTypeEcho
	} else {
		network = "ip6:ipv6-icmp"
		icmpType = ipv6.ICMPTypeEchoRequest
	}

	c, err := icmp.ListenPacket(network, "")
	if err != nil {
		return 0, false, err
	}
	defer c.Close()

	echo := &icmp.Message{
		Type: icmpType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: make([]byte, size),
		},
	}

	wb, err := echo.Marshal(nil)
	if err != nil {
		return 0, false, err
	}

	if dl, ok := ctx.Deadline(); ok {
		_ = c.SetDeadline(dl)
	} else {
		_ = c.SetDeadline(time.Now().Add(p.Timeout))
	}

	start := time.Now()
	if _, err = c.WriteTo(wb, ipAddr); err != nil {
		return 0, false, err
	}

	rb := make([]byte, 1500)
	for {
		n, peer, err := c.ReadFrom(rb)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return 0, false, nil
			}
			return 0, false, err
		}
		_ = peer

		duration := time.Since(start)

		rm, err := icmp.ParseMessage(getProto(ipAddr), rb[:n])
		if err != nil {
			continue
		}

		switch body := rm.Body.(type) {
		case *icmp.Echo:
			if body.ID == echo.Body.(*icmp.Echo).ID {
				return duration, true, nil
			}
		default:
			continue
		}
	}
}

func getProto(ip *net.IPAddr) int {
	if ip.IP.To4() != nil {
		return 1 // ICMP for IPv4
	}
	return 58 // ICMPv6
}

