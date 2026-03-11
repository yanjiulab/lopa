package protocol

import (
	"context"
	"encoding/binary"
	"net"
	"sync/atomic"
	"time"
)

// UDPProber implements Pinger by sending UDP packets to a reflector and measuring RTT from echo.
// Packet format: 8-byte sequence number (big-endian) + optional padding to PacketSize.
// The reflector echoes the packet back; we match by sequence and measure round-trip time.
type UDPProber struct {
	Target     string
	Timeout    time.Duration
	PacketSize int
	Network    string
	SourceIP   string
	Interface  string
	seq        uint64
}

func (p *UDPProber) Ping(ctx context.Context) (time.Duration, bool, error) {
	if p.Timeout <= 0 {
		p.Timeout = 3 * time.Second
	}
	size := p.PacketSize
	if size < 8 {
		size = 8
	}
	network := p.Network
	if network == "" {
		network = "udp"
	}

	raddr, err := net.ResolveUDPAddr(network, p.Target)
	if err != nil {
		return 0, false, err
	}
	var laddr *net.UDPAddr
	if a, err := LocalAddr(network, p.SourceIP, p.Interface); err == nil && a != nil {
		laddr = a.(*net.UDPAddr)
	}
	conn, err := net.DialUDP(network, laddr, raddr)
	if err != nil {
		return 0, false, err
	}
	defer conn.Close()

	seq := atomic.AddUint64(&p.seq, 1)
	buf := make([]byte, size)
	binary.BigEndian.PutUint64(buf[:8], seq)

	dl, ok := ctx.Deadline()
	if !ok {
		dl = time.Now().Add(p.Timeout)
	}
	if err := conn.SetDeadline(dl); err != nil {
		return 0, false, err
	}

	start := time.Now()
	if _, err := conn.Write(buf); err != nil {
		return 0, false, err
	}

	rb := make([]byte, size+8)
	n, err := conn.Read(rb)
	if err != nil {
		return 0, false, nil
	}
	if n < 8 {
		return 0, false, nil
	}
	if binary.BigEndian.Uint64(rb[:8]) != seq {
		return 0, false, nil
	}
	return time.Since(start), true, nil
}
