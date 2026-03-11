package measurement

import (
	"context"
	"sync"
	"time"

	"github.com/yanjiulab/lopa/internal/logger"
	"github.com/yanjiulab/lopa/internal/node"
	"github.com/yanjiulab/lopa/internal/protocol"
)

// Engine manages measurement tasks and results in memory.
type Engine struct {
	mu      sync.RWMutex
	tasks   map[TaskID]*Task
	results map[TaskID]*Result
	cancel  map[TaskID]context.CancelFunc
}

var (
	defaultEngine *Engine
	once          sync.Once
)

// DefaultEngine returns the singleton engine instance.
func DefaultEngine() *Engine {
	once.Do(func() {
		defaultEngine = &Engine{
			tasks:   make(map[TaskID]*Task),
			results: make(map[TaskID]*Result),
			cancel:  make(map[TaskID]context.CancelFunc),
		}
	})
	return defaultEngine
}

// CreatePingTask creates and starts a ping task with given parameters.
func (e *Engine) CreatePingTask(params TaskParams) (TaskID, error) {
	id := TaskID(node.NextTaskID())
	now := time.Now()

	t := &Task{
		ID:        id,
		Params:    params,
		NodeID:    node.ID(),
		CreatedAt: now,
	}

	r := &Result{
		TaskID:   id,
		NodeID:   t.NodeID,
		Target:   params.Target,
		Type:     params.Type,
		Mode:     params.Mode,
		Started:  now,
		Status:   "running",
		Rounds:   make([]RoundResult, 0),
		Total:    Stats{},
		Window:   nil,
		Error:    "",
	}

	e.mu.Lock()
	e.tasks[id] = t
	e.results[id] = r
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	e.mu.Lock()
	e.cancel[id] = cancel
	e.mu.Unlock()

	go e.runProbe(ctx, t, r)

	return id, nil
}

// CreateTcpTask creates and starts a TCP connect (TCPING) task.
func (e *Engine) CreateTcpTask(params TaskParams) (TaskID, error) {
	id := TaskID(node.NextTaskID())
	now := time.Now()

	t := &Task{
		ID:        id,
		Params:    params,
		NodeID:    node.ID(),
		CreatedAt: now,
	}

	r := &Result{
		TaskID:   id,
		NodeID:   t.NodeID,
		Target:   params.Target,
		Type:     params.Type,
		Mode:     params.Mode,
		Started:  now,
		Status:   "running",
		Rounds:   make([]RoundResult, 0),
		Total:    Stats{},
		Window:   nil,
		Error:    "",
	}

	e.mu.Lock()
	e.tasks[id] = t
	e.results[id] = r
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	e.mu.Lock()
	e.cancel[id] = cancel
	e.mu.Unlock()

	go e.runTcp(ctx, t, r)

	return id, nil
}

// CreateUdpTask creates and starts a UDP probe task (target must be a reflector host:port).
func (e *Engine) CreateUdpTask(params TaskParams) (TaskID, error) {
	id := TaskID(node.NextTaskID())
	now := time.Now()

	t := &Task{
		ID:        id,
		Params:    params,
		NodeID:    node.ID(),
		CreatedAt: now,
	}

	r := &Result{
		TaskID:   id,
		NodeID:   t.NodeID,
		Target:   params.Target,
		Type:     params.Type,
		Mode:     params.Mode,
		Started:  now,
		Status:   "running",
		Rounds:   make([]RoundResult, 0),
		Total:    Stats{},
		Window:   nil,
		Error:    "",
	}

	e.mu.Lock()
	e.tasks[id] = t
	e.results[id] = r
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	e.mu.Lock()
	e.cancel[id] = cancel
	e.mu.Unlock()

	go e.runTwamp(ctx, t, r)

	return id, nil
}

// CreateTwampTask creates and starts a TWAMP-light task (target must be a standard Session-Reflector, typically host:862).
func (e *Engine) CreateTwampTask(params TaskParams) (TaskID, error) {
	id := TaskID(node.NextTaskID())
	now := time.Now()

	t := &Task{
		ID:        id,
		Params:    params,
		NodeID:    node.ID(),
		CreatedAt: now,
	}

	r := &Result{
		TaskID:   id,
		NodeID:   t.NodeID,
		Target:   params.Target,
		Type:     params.Type,
		Mode:     params.Mode,
		Started:  now,
		Status:   "running",
		Rounds:   make([]RoundResult, 0),
		Total:    Stats{},
		Window:   nil,
		Error:    "",
	}

	e.mu.Lock()
	e.tasks[id] = t
	e.results[id] = r
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	e.mu.Lock()
	e.cancel[id] = cancel
	e.mu.Unlock()

	go e.runTwamp(ctx, t, r)

	return id, nil
}

// StopTask stops a running task.
func (e *Engine) StopTask(id TaskID) {
	e.mu.Lock()
	cancel, ok := e.cancel[id]
	e.mu.Unlock()
	if ok {
		cancel()
	}
}

// GetResult returns the latest result for a task.
func (e *Engine) GetResult(id TaskID) (*Result, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	r, ok := e.results[id]
	return r, ok
}

// ListResults returns a snapshot of all task results.
func (e *Engine) ListResults() []*Result {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*Result, 0, len(e.results))
	for _, r := range e.results {
		out = append(out, r)
	}
	return out
}

// DeleteTask removes a task and its result from the engine.
// If the task is still running, it will be stopped first.
func (e *Engine) DeleteTask(id TaskID) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if cancel, ok := e.cancel[id]; ok {
		cancel()
		delete(e.cancel, id)
	}

	_, taskExists := e.tasks[id]
	_, resExists := e.results[id]
	if !taskExists && !resExists {
		return false
	}

	delete(e.tasks, id)
	delete(e.results, id)
	return true
}

func (e *Engine) runProbe(ctx context.Context, task *Task, result *Result) {
	log := logger.S()
	params := task.Params

	pinger := &protocol.ICMPPinger{
		Addr:      params.Target,
		IPVersion: params.IPVersion,
		Timeout:   params.Timeout,
		Size:      params.PacketSize,
	}

	switch params.Mode {
	case ModeCount:
		e.runPingCount(ctx, pinger, task, result)
	case ModeDuration:
		e.runPingDuration(ctx, pinger, task, result)
	case ModeContinuous:
		e.runPingContinuous(ctx, pinger, task, result)
	default:
		log.Warnf("unknown mode %v for task %s", params.Mode, task.ID)
	}
}

func (e *Engine) runTcp(ctx context.Context, task *Task, result *Result) {
	log := logger.S()
	params := task.Params

	network := "tcp"
	if params.IPVersion == "ipv4" {
		network = "tcp4"
	} else if params.IPVersion == "ipv6" {
		network = "tcp6"
	}
	pinger := &protocol.TCPPinger{
		Target:    params.Target,
		Timeout:   params.Timeout,
		Network:   network,
		SourceIP:  params.SourceIP,
		Interface: params.Interface,
	}

	switch params.Mode {
	case ModeCount:
		e.runPingCount(ctx, pinger, task, result)
	case ModeDuration:
		e.runPingDuration(ctx, pinger, task, result)
	case ModeContinuous:
		e.runPingContinuous(ctx, pinger, task, result)
	default:
		log.Warnf("unknown mode %v for task %s", params.Mode, task.ID)
	}
}

func (e *Engine) runUdp(ctx context.Context, task *Task, result *Result) {
	log := logger.S()
	params := task.Params

	network := "udp"
	if params.IPVersion == "ipv4" {
		network = "udp4"
	} else if params.IPVersion == "ipv6" {
		network = "udp6"
	}
	size := params.PacketSize
	if size < 8 {
		size = 8
	}
	pinger := &protocol.UDPProber{
		Target:     params.Target,
		Timeout:    params.Timeout,
		PacketSize: size,
		Network:    network,
		SourceIP:   params.SourceIP,
		Interface:  params.Interface,
	}

	switch params.Mode {
	case ModeCount:
		e.runPingCount(ctx, pinger, task, result)
	case ModeDuration:
		e.runPingDuration(ctx, pinger, task, result)
	case ModeContinuous:
		e.runPingContinuous(ctx, pinger, task, result)
	default:
		log.Warnf("unknown mode %v for task %s", params.Mode, task.ID)
	}
}

func (e *Engine) runTwamp(ctx context.Context, task *Task, result *Result) {
	log := logger.S()
	params := task.Params

	network := "udp"
	if params.IPVersion == "ipv4" {
		network = "udp4"
	} else if params.IPVersion == "ipv6" {
		network = "udp6"
	}
	size := params.PacketSize
	if size < 16 {
		size = 16
	}
	pinger := &protocol.TWAMPPinger{
		Target:     params.Target,
		Timeout:    params.Timeout,
		PacketSize: size,
		Network:    network,
		SourceIP:   params.SourceIP,
		Interface:  params.Interface,
	}

	switch params.Mode {
	case ModeCount:
		e.runPingCount(ctx, pinger, task, result)
	case ModeDuration:
		e.runPingDuration(ctx, pinger, task, result)
	case ModeContinuous:
		e.runPingContinuous(ctx, pinger, task, result)
	default:
		log.Warnf("unknown mode %v for task %s", params.Mode, task.ID)
	}
}

func (e *Engine) updateResult(id TaskID, fn func(*Result)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if r, ok := e.results[id]; ok {
		fn(r)
		r.LastUpdated = time.Now()
	}
}

// computeStats updates statistics given a new probe RTT.
// Jitter is the running average of |rtt - previous_rtt| (consecutive delay variation).
func computeStats(s *Stats, rtt time.Duration, ok bool) {
	s.Sent++
	if ok {
		s.Received++
		if s.MinRTT == 0 || rtt < s.MinRTT {
			s.MinRTT = rtt
		}
		if rtt > s.MaxRTT {
			s.MaxRTT = rtt
		}
		n := s.Received
		if n == 1 {
			s.AvgRTT = rtt
			s.lastRTT = rtt
		} else {
			s.AvgRTT = ((s.AvgRTT * time.Duration(n-1)) + rtt) / time.Duration(n)
			diff := rtt - s.lastRTT
			if diff < 0 {
				diff = -diff
			}
			s.Jitter = (s.Jitter*time.Duration(n-2) + diff) / time.Duration(n-1)
			s.lastRTT = rtt
		}
	}
	if s.Sent > 0 {
		s.LossRate = float64(s.Sent-s.Received) / float64(s.Sent)
	}
}


