package executor

import (
	"sync/atomic"
	"testing"

	"github.com/gorilla/websocket"
	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

type countingWebsocketLifecycle struct {
	binds   atomic.Int32
	retains atomic.Int32
}

func (l *countingWebsocketLifecycle) Bind(func() error) error {
	l.binds.Add(1)
	return nil
}

func (*countingWebsocketLifecycle) End(string) {}

func (l *countingWebsocketLifecycle) Retain() {
	l.retains.Add(1)
}

func TestCodexWebsocketSessionBindsSameLifecycleAndConnectionOnce(t *testing.T) {
	conn := &websocket.Conn{}
	session := &codexWebsocketSession{conn: conn}
	lifecycle := &countingWebsocketLifecycle{}
	opts := cliproxyexecutor.Options{ExecutionLifecycle: lifecycle}

	if errBind := session.bindExecutionLifecycle(opts, conn); errBind != nil {
		t.Fatalf("first bindExecutionLifecycle() error = %v", errBind)
	}
	if errBind := session.bindExecutionLifecycle(opts, conn); errBind != nil {
		t.Fatalf("second bindExecutionLifecycle() error = %v", errBind)
	}
	if got := lifecycle.binds.Load(); got != 1 {
		t.Fatalf("lifecycle Bind calls = %d, want 1", got)
	}
	if got := lifecycle.retains.Load(); got != 1 {
		t.Fatalf("lifecycle Retain calls = %d, want 1", got)
	}
}
