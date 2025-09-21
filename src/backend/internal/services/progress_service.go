package services

import (
    "sync"
    "time"
    "github.com/google/uuid"
)

// ProgressEvent describes a progress update for a long-running operation.
type ProgressEvent struct {
    OpID     string      `json:"opId"`
    Type     string      `json:"type"`      // start, step-start, step-log, step-done, error, complete
    Step     string      `json:"step"`
    Message  string      `json:"message"`
    Host     string      `json:"host,omitempty"`
    Progress float64     `json:"progress,omitempty"` // 0..1
    Data     interface{} `json:"data,omitempty"`
    TS       int64       `json:"ts"` // unix millis
}

type OperationStatus string

const (
    StatusRunning  OperationStatus = "running"
    StatusFailed   OperationStatus = "failed"
    StatusComplete OperationStatus = "complete"
)

// Operation holds in-memory state for a running progress operation.
type Operation struct {
    ID          string
    Name        string
    StartedAt   time.Time
    CompletedAt time.Time
    Status      OperationStatus
    Events      []ProgressEvent

    subs map[chan ProgressEvent]struct{}
    mu   sync.Mutex
}

func newOperation(name string) *Operation {
    return &Operation{
        ID:        uuid.NewString(),
        Name:      name,
        StartedAt: time.Now(),
        Status:    StatusRunning,
        Events:    make([]ProgressEvent, 0, 32),
        subs:      make(map[chan ProgressEvent]struct{}),
    }
}

func (o *Operation) addSubscriber() chan ProgressEvent {
    o.mu.Lock()
    defer o.mu.Unlock()
    ch := make(chan ProgressEvent, 32)
    o.subs[ch] = struct{}{}
    return ch
}

func (o *Operation) removeSubscriber(ch chan ProgressEvent) {
    o.mu.Lock()
    defer o.mu.Unlock()
    if _, ok := o.subs[ch]; ok {
        delete(o.subs, ch)
        close(ch)
    }
}

func (o *Operation) publish(ev ProgressEvent) {
    o.mu.Lock()
    o.Events = append(o.Events, ev)
    // broadcast non-blocking
    for ch := range o.subs {
        select {
        case ch <- ev:
        default:
            // drop if subscriber is slow
        }
    }
    o.mu.Unlock()
}

func (o *Operation) complete(failed bool) {
    o.mu.Lock()
    if failed {
        o.Status = StatusFailed
    } else {
        o.Status = StatusComplete
    }
    o.CompletedAt = time.Now()
    // close subscribers
    for ch := range o.subs {
        close(ch)
        delete(o.subs, ch)
    }
    o.mu.Unlock()
}

// ProgressManager is a singleton manager for progress operations.
type ProgressManager struct {
    mu   sync.RWMutex
    ops  map[string]*Operation
}

var (
    defaultPM   *ProgressManager
    defaultPMOnce sync.Once
)

// GetProgressManager returns a global singleton progress manager.
func GetProgressManager() *ProgressManager {
    defaultPMOnce.Do(func() {
        defaultPM = &ProgressManager{ops: make(map[string]*Operation)}
    })
    return defaultPM
}

// Start creates a new operation, stores it, and emits an initial start event.
func (pm *ProgressManager) Start(name string, startMsg string) *Operation {
    op := newOperation(name)
    pm.mu.Lock()
    pm.ops[op.ID] = op
    pm.mu.Unlock()
    op.publish(ProgressEvent{OpID: op.ID, Type: "start", Step: "start", Message: startMsg, TS: time.Now().UnixMilli()})
    return op
}

// Get retrieves an operation by id.
func (pm *ProgressManager) Get(id string) (*Operation, bool) {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    op, ok := pm.ops[id]
    return op, ok
}

// Snapshot returns a copy of current events and status for the operation.
type ProgressSnapshot struct {
    ID        string           `json:"id"`
    Name      string           `json:"name"`
    Status    OperationStatus  `json:"status"`
    StartedAt int64            `json:"startedAt"`
    CompletedAt int64          `json:"completedAt,omitempty"`
    Events    []ProgressEvent  `json:"events"`
}

func (pm *ProgressManager) Snapshot(id string) (ProgressSnapshot, bool) {
    op, ok := pm.Get(id)
    if !ok {
        return ProgressSnapshot{}, false
    }
    op.mu.Lock()
    defer op.mu.Unlock()
    snap := ProgressSnapshot{
        ID: op.ID,
        Name: op.Name,
        Status: op.Status,
        StartedAt: op.StartedAt.UnixMilli(),
        Events: append([]ProgressEvent(nil), op.Events...),
    }
    if !op.CompletedAt.IsZero() {
        snap.CompletedAt = op.CompletedAt.UnixMilli()
    }
    return snap, true
}

// Emit appends and broadcasts an event for the operation.
func (pm *ProgressManager) Emit(id string, ev ProgressEvent) {
    if op, ok := pm.Get(id); ok {
        if ev.TS == 0 {
            ev.TS = time.Now().UnixMilli()
        }
        ev.OpID = id
        op.publish(ev)
    }
}

// Subscribe returns a channel of future events for the operation.
func (pm *ProgressManager) Subscribe(id string) (chan ProgressEvent, bool) {
    op, ok := pm.Get(id)
    if !ok {
        return nil, false
    }
    ch := op.addSubscriber()
    return ch, true
}

// Complete marks operation done and closes subscribers.
func (pm *ProgressManager) Complete(id string, failed bool, message string) {
    op, ok := pm.Get(id)
    if !ok {
        return
    }
    op.publish(ProgressEvent{OpID: id, Type: "complete", Step: "complete", Message: message, TS: time.Now().UnixMilli()})
    op.complete(failed)
}
