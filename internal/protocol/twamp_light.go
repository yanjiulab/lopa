// Package protocol: TWAMP-light client (RFC 5357 Appendix I, RFC 4656 §4.1.2).
// Session-Sender packet: 4 byte Seq, 8 byte Timestamp (T1), 2 byte Error, 2 byte MBZ, [padding].
// Session-Reflector response: same 16 bytes + T2 block (8+2+2) + T3 block (8+2+2) = 40 bytes min.

package protocol

import (
	"context"
	"encoding/binary"
	"net"
	"sync/atomic"
	"time"
)

const (
	twampSenderPktMin   = 16
	twampReflectorPktMin = 40
	twampDefaultPort    = "862"
)

// encodeTWAMPTimestamp writes 8-byte RFC 4656 timestamp (32b seconds + 32b fraction) big-endian.
func encodeTWAMPTimestamp(b []byte, t time.Time) {
	sec := uint32(t.Unix())
	frac := uint32(uint64(t.Nanosecond()) * (1 << 32) / 1e9)
	binary.BigEndian.PutUint32(b[0:4], sec)
	binary.BigEndian.PutUint32(b[4:8], frac)
}

// TWAMPPinger implements Pinger using TWAMP-light (RFC 5357) toward a standard Session-Reflector.
// Target must be host:port (default port 862). Packet format is RFC-compliant for interoperability.
type TWAMPPinger struct {
	Target     string
	Timeout    time.Duration
	PacketSize int
	Network    string
	SourceIP   string
	Interface  string
	seq        uint32
}

func (p *TWAMPPinger) Ping(ctx context.Context) (time.Duration, bool, error) {
	if p.Timeout <= 0 {
		p.Timeout = 3 * time.Second
	}
	size := p.PacketSize
	if size < twampSenderPktMin {
		size = twampSenderPktMin
	}
	network := p.Network
	if network == "" {
		network = "udp"
	}

	addr := p.Target
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, twampDefaultPort)
	}
	raddr, err := net.ResolveUDPAddr(network, addr)
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

	seq := atomic.AddUint32(&p.seq, 1)
	// Session-Sender packet: Seq(4) + Timestamp T1(8) + Error(2) + MBZ(2) + padding
	buf := make([]byte, size)
	binary.BigEndian.PutUint32(buf[0:4], seq)
	encodeTWAMPTimestamp(buf[4:12], time.Now())
	// Error Estimate = 0, MBZ = 0
	buf[12] = 0
	buf[13] = 0
	buf[14] = 0
	buf[15] = 0

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

	rb := make([]byte, 256)
	n, err := conn.Read(rb)
	if err != nil {
		return 0, false, nil
	}
	if n < twampReflectorPktMin {
		return 0, false, nil
	}
	if binary.BigEndian.Uint32(rb[0:4]) != seq {
		return 0, false, nil
	}
	return time.Since(start), true, nil
}
