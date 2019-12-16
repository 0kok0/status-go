package whisper

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/tsenart/tb"
)

type runLoop func(p *Peer, rw p2p.MsgReadWriter) error

type RateLimiterHandler interface {
	ExceedPeerLimit() error
	ExceedIPLimit() error
}

type MetricsRateLimiterHandler struct{}

func (MetricsRateLimiterHandler) ExceedPeerLimit() error { rateLimiterPeerExceeded.Inc(1); return nil }
func (MetricsRateLimiterHandler) ExceedIPLimit() error   { rateLimiterIPExceeded.Inc(1); return nil }

var ErrRateLimitExceeded = errors.New("rate limit has been exceeded")

type DropPeerRateLimiterHandler struct {
	// Tolerance is a number of how many a limit must be exceeded
	// in order to drop a peer.
	Tolerance int64

	peerLimitExceeds int64
	ipLimitExceeds   int64
}

func (h *DropPeerRateLimiterHandler) ExceedPeerLimit() error {
	h.peerLimitExceeds++
	if h.Tolerance > 0 && h.peerLimitExceeds >= h.Tolerance {
		return ErrRateLimitExceeded
	}
	return nil
}

func (h *DropPeerRateLimiterHandler) ExceedIPLimit() error {
	h.ipLimitExceeds++
	if h.Tolerance > 0 && h.ipLimitExceeds >= h.Tolerance {
		return ErrRateLimitExceeded
	}
	return nil
}

type PeerRateLimiterConfig struct {
	LimitPerSecIP      int64
	LimitPerSecPeerID  int64
	WhitelistedIPs     []string
	WhitelistedPeerIDs []enode.ID
}

var defaultPeerRateLimiterConfig = PeerRateLimiterConfig{
	LimitPerSecIP:      10,
	LimitPerSecPeerID:  5,
	WhitelistedIPs:     nil,
	WhitelistedPeerIDs: nil,
}

type PeerRateLimiter struct {
	peerIDThrottler *tb.Throttler
	ipThrottler     *tb.Throttler

	limitPerSecIP     int64
	limitPerSecPeerID int64

	whitelistedPeerIDs []enode.ID
	whitelistedIPs     []string

	handlers []RateLimiterHandler
}

func NewPeerRateLimiter(cfg *PeerRateLimiterConfig, handlers ...RateLimiterHandler) *PeerRateLimiter {
	if cfg == nil {
		copy := defaultPeerRateLimiterConfig
		cfg = &copy
	}

	return &PeerRateLimiter{
		peerIDThrottler:    tb.NewThrottler(time.Millisecond * 100),
		ipThrottler:        tb.NewThrottler(time.Millisecond * 100),
		limitPerSecIP:      cfg.LimitPerSecIP,
		limitPerSecPeerID:  cfg.LimitPerSecPeerID,
		whitelistedPeerIDs: cfg.WhitelistedPeerIDs,
		whitelistedIPs:     cfg.WhitelistedIPs,
		handlers:           handlers,
	}
}

func (r *PeerRateLimiter) decorate(p *Peer, rw p2p.MsgReadWriter, runLoop runLoop) error {
	in, out := p2p.MsgPipe()
	defer in.Close()
	defer out.Close()
	errC := make(chan error, 1)

	// Read from the original reader and write to the message pipe.
	go func() {
		for {
			packet, err := rw.ReadMsg()
			if err != nil {
				errC <- fmt.Errorf("failed to read packet: %v", err)
				return
			}

			rateLimiterProcessed.Inc(1)

			var ip string
			if p != nil && p.peer != nil {
				ip = p.peer.Node().IP().String()
			}
			if halted := r.throttleIP(ip); halted {
				for _, h := range r.handlers {
					if err := h.ExceedIPLimit(); err != nil {
						errC <- fmt.Errorf("exceed rate limit by IP: %w", err)
						return
					}
				}
			}

			var peerID []byte
			if p != nil {
				peerID = p.ID()
			}
			if halted := r.throttlePeer(peerID); halted {
				for _, h := range r.handlers {
					if err := h.ExceedPeerLimit(); err != nil {
						errC <- fmt.Errorf("exceeded rate limit by peer: %w", err)
						return
					}
				}
			}

			if err := in.WriteMsg(packet); err != nil {
				errC <- fmt.Errorf("failed to write packet to pipe: %v", err)
				return
			}
		}
	}()

	// Read from the message pipe and write to the original writer.
	go func() {
		for {
			packet, err := in.ReadMsg()
			if err != nil {
				errC <- fmt.Errorf("failed to read packet from pipe: %v", err)
				return
			}
			if err := rw.WriteMsg(packet); err != nil {
				errC <- fmt.Errorf("failed to write packet: %v", err)
				return
			}
		}
	}()

	go func() {
		errC <- runLoop(p, out)
	}()

	return <-errC
}

// throttleIP throttles a number of messages incoming from a given IP.
// It allows 10 packets per second.
func (r *PeerRateLimiter) throttleIP(ip string) bool {
	if r.limitPerSecIP == 0 {
		return false
	}
	if stringSliceContains(r.whitelistedIPs, ip) {
		return false
	}
	return r.ipThrottler.Halt(ip, 1, r.limitPerSecIP)
}

// throttlePeer throttles a number of messages incoming from a peer.
// It allows 3 packets per second.
func (r *PeerRateLimiter) throttlePeer(peerID []byte) bool {
	if r.limitPerSecIP == 0 {
		return false
	}
	var id enode.ID
	copy(id[:], peerID)
	if enodeIDSliceContains(r.whitelistedPeerIDs, id) {
		return false
	}
	return r.peerIDThrottler.Halt(id.String(), 1, r.limitPerSecPeerID)
}

func stringSliceContains(s []string, searched string) bool {
	for _, item := range s {
		if item == searched {
			return true
		}
	}
	return false
}

func enodeIDSliceContains(s []enode.ID, searched enode.ID) bool {
	for _, item := range s {
		if bytes.Equal(item.Bytes(), searched.Bytes()) {
			return true
		}
	}
	return false
}
