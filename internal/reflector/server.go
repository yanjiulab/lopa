package reflector

import (
	"context"
	"net"
	"sync"

	"github.com/yanjiulab/lopa/internal/logger"
)

// Run starts the reflector: listens on addr (UDP) and echoes every packet back to the sender.
// It runs until ctx is cancelled. Typically addr is ":8081".
func Run(ctx context.Context, addr string) error {
	if addr == "" {
		addr = ":8081"
	}
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}

	logger.S().Infow("reflector listening", "addr", addr)

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
			logger.S().Warnw("reflector read error", "err", err)
			continue
		}
		wg.Add(1)
		go func(data []byte, to net.Addr) {
			defer wg.Done()
			_, _ = pc.WriteTo(data, to)
		}(append([]byte(nil), buf[:n]...), remote)
	}
}
