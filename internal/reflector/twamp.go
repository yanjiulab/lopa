// TWAMP-light Session-Reflector (RFC 5357 §4.2.1).
// Receives Session-Sender packet (min 16 bytes), responds with reflector format (40 bytes min):
// first 16 bytes (Seq, T1, Error, MBZ) + T2 block (8+2+2) + T3 block (8+2+2).

package reflector

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"time"

	"github.com/yanjiulab/lopa/internal/logger"
)

const (
	twampSenderMin   = 16
	twampReflectorLen = 40
)

func encodeTWAMPTimestamp(b []byte, t time.Time) {
	sec := uint32(t.Unix())
	frac := uint32(uint64(t.Nanosecond()) * (1 << 32) / 1e9)
	binary.BigEndian.PutUint32(b[0:4], sec)
	binary.BigEndian.PutUint32(b[4:8], frac)
}

// RunTWAMP runs a TWAMP-light Session-Reflector on addr (e.g. ":862").
// It runs until ctx is cancelled. Compatible with standard TWAMP-light clients.
func RunTWAMP(ctx context.Context, addr string) error {
	if addr == "" {
		return nil
	}
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}

	logger.S().Infow("twamp reflector listening", "addr", addr)

	var wg sync.WaitGroup
	defer wg.Wait()

	go func() {
		<-ctx.Done()
		_ = pc.Close()
	}()

	buf := make([]byte, 65535)
	for {
		n, remote, err := pc.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			logger.S().Warnw("twamp reflector read error", "err", err)
			continue
		}
		if n < twampSenderMin {
			continue
		}
		wg.Add(1)
		go func(data []byte, to net.Addr) {
			defer wg.Done()
			reply := make([]byte, twampReflectorLen)
			copy(reply[:16], data[:16])
			now := time.Now()
			encodeTWAMPTimestamp(reply[16:24], now)
			binary.BigEndian.PutUint16(reply[24:26], 0)
			binary.BigEndian.PutUint16(reply[26:28], 0)
			encodeTWAMPTimestamp(reply[28:36], time.Now())
			binary.BigEndian.PutUint16(reply[36:38], 0)
			binary.BigEndian.PutUint16(reply[38:40], 0)
			_, _ = pc.WriteTo(reply, to)
		}(append([]byte(nil), buf[:n]...), remote)
	}
}
