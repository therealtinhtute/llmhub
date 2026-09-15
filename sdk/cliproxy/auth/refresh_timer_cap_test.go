package auth

import (
	"testing"
	"time"
)

// The following tests cover the refresh-loop timer cap ported from upstream
// CLIProxyAPI commit 48e5e9e03d21 ("fix(auth): cap refresh loop timer wait
// duration"). Local symbols under test: authAutoRefreshLoop.nextWait,
// maxRefreshTimerWait, authAutoRefreshLoop.resetTimer.

func TestNextWait_CapsLongExpiryAtThirtySeconds(t *testing.T) {
	now := time.Now()
	loop := newAuthAutoRefreshLoop(NewManager(nil, nil, nil), time.Minute, 1)
	loop.upsert("a1", now.Add(10*time.Minute))

	wait, ok := loop.nextWait(now)
	if !ok {
		t.Fatal("nextWait() ok = false, want true")
	}
	if wait != maxRefreshTimerWait {
		t.Fatalf("nextWait() = %v, want capped at %v", wait, maxRefreshTimerWait)
	}
}

func TestNextWait_ShortWaitUnchanged(t *testing.T) {
	now := time.Now()
	loop := newAuthAutoRefreshLoop(NewManager(nil, nil, nil), time.Minute, 1)
	loop.upsert("a1", now.Add(5*time.Second))

	wait, ok := loop.nextWait(now)
	if !ok {
		t.Fatal("nextWait() ok = false, want true")
	}
	if wait != 5*time.Second {
		t.Fatalf("nextWait() = %v, want 5s", wait)
	}
}

func TestNextWait_PastDueClampsToZero(t *testing.T) {
	now := time.Now()
	loop := newAuthAutoRefreshLoop(NewManager(nil, nil, nil), time.Minute, 1)
	loop.upsert("a1", now.Add(-time.Minute))

	wait, ok := loop.nextWait(now)
	if !ok {
		t.Fatal("nextWait() ok = false, want true")
	}
	if wait != 0 {
		t.Fatalf("nextWait() = %v, want 0 for past-due entry", wait)
	}
}

func TestNextWait_EmptyQueue(t *testing.T) {
	loop := newAuthAutoRefreshLoop(NewManager(nil, nil, nil), time.Minute, 1)
	if wait, ok := loop.nextWait(time.Now()); ok || wait != 0 {
		t.Fatalf("nextWait() = (%v, %v), want (0, false) for empty queue", wait, ok)
	}
}

// Simulated suspend/resume: an entry scheduled far in the future must still
// wake the loop within maxRefreshTimerWait so credentials expiring during a
// system sleep are detected promptly after resume.
func TestRefreshLoop_TimerWakesWithinCapForFarFutureEntry(t *testing.T) {
	manager := NewManager(nil, nil, nil)
	loop := newAuthAutoRefreshLoop(manager, time.Minute, 1)
	loop.upsert("far-future", time.Now().Add(24*time.Hour))

	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	defer timer.Stop()
	var timerCh <-chan time.Time
	loop.resetTimer(timer, &timerCh, time.Now())
	if timerCh == nil {
		t.Fatal("expected armed timer channel")
	}
	select {
	case <-timerCh:
		// woke within the cap — correct
	case <-time.After(maxRefreshTimerWait + 15*time.Second):
		t.Fatal("timer did not fire within maxRefreshTimerWait; suspend-resume wake guarantee broken")
	}
}
