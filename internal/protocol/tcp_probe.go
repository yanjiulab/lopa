package protocol

import (
	"context"
	"net"
	"time"
)

// TCPPinger implements Pinger by measuring TCP connect RTT to host:port.
type TCPPinger struct {
	Target    string
	Timeout   time.Duration
	Network   string
	SourceIP  string
	Interface string
}

func (p *TCPPinger) Ping(ctx context.Context) (time.Duration, bool, error) {
	if p.Timeout <= 0 {
		p.Timeout = 3 * time.Second
	}
	addr := p.Target
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, "80")
	}
	network := p.Network
	if network == "" {
		network = "tcp"
	}
	start := time.Now()
	dialer := net.Dialer{Timeout: p.Timeout}
	if laddr, err := LocalAddr(network, p.SourceIP, p.Interface); err == nil && laddr != nil {
		dialer.LocalAddr = laddr
	}
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return 0, false, nil
	}
	conn.Close()
	return time.Since(start), true, nil
}
