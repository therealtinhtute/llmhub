package registry

import (
	"testing"
)

func TestRegisterClientIfEpochCurrent(t *testing.T) {
	r := newTestModelRegistry()
	clientID := "test-fence-client"
	provider := "gemini"
	models := []*ModelInfo{{ID: "gemini-2.5-flash"}}

	// Initial epoch is 0
	epoch0 := r.ClientRegistrationEpoch(clientID)
	if epoch0 != 0 {
		t.Fatalf("expected initial epoch 0, got %d", epoch0)
	}

	// 1. Successful registration with current epoch (0)
	publishCalled := false
	published := r.RegisterClientIfEpochCurrent(clientID, provider, epoch0, models, func() bool {
		publishCalled = true
		return true
	})
	if !published || !publishCalled {
		t.Fatalf("expected successful registration at epoch 0, published=%v, publishCalled=%v", published, publishCalled)
	}

	// After registration, epoch should be 1
	epoch1 := r.ClientRegistrationEpoch(clientID)
	if epoch1 != 1 {
		t.Fatalf("expected epoch 1 after registration, got %d", epoch1)
	}

	// 2. Stale registration attempting to use expectedEpoch=0 must be rejected
	publishCalled = false
	stalePublished := r.RegisterClientIfEpochCurrent(clientID, provider, 0, models, func() bool {
		publishCalled = true
		return true
	})
	if stalePublished || publishCalled {
		t.Fatalf("expected stale registration to be rejected, published=%v, publishCalled=%v", stalePublished, publishCalled)
	}

	// 3. Current epoch registration but publish() callback returns false -> registration aborted
	publishCalled = false
	rejectedByPublish := r.RegisterClientIfEpochCurrent(clientID, provider, epoch1, models, func() bool {
		publishCalled = true
		return false
	})
	if rejectedByPublish || !publishCalled {
		t.Fatalf("expected rejection when publish returns false, rejectedByPublish=%v, publishCalled=%v", rejectedByPublish, publishCalled)
	}
	// Epoch remains unchanged because registration was aborted
	if gotEpoch := r.ClientRegistrationEpoch(clientID); gotEpoch != epoch1 {
		t.Fatalf("expected epoch to remain %d, got %d", epoch1, gotEpoch)
	}

	// 4. Successful UnregisterClientIfEpochCurrent
	publishCalled = false
	unregistered := r.UnregisterClientIfEpochCurrent(clientID, epoch1, func() bool {
		publishCalled = true
		return true
	})
	if !unregistered || !publishCalled {
		t.Fatalf("expected unregister to succeed, unregistered=%v, publishCalled=%v", unregistered, publishCalled)
	}

	// 5. Stale unregister must be rejected
	publishCalled = false
	staleUnregistered := r.UnregisterClientIfEpochCurrent(clientID, 999, func() bool {
		publishCalled = true
		return true
	})
	if staleUnregistered || publishCalled {
		t.Fatalf("expected stale unregister to be rejected")
	}
}
