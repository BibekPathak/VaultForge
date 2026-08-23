package core

import (
	"testing"
	"time"
)

func TestMetricsCollector_IncrementCounters(t *testing.T) {
	m := NewMetricsCollector()

	m.IncrRequestCount()
	m.IncrRequestCount()
	m.IncrRequestCount()
	m.IncrRequestErrors()
	m.IncrPolicyEvals()
	m.IncrPolicyEvals()
	m.IncrPolicyDenied()
	m.IncrZKProofs()
	m.IncrMPCSigns()
	m.IncrTXSubmitted()
	m.IncrTXConfirmed()
	m.IncrTXFailed()
	m.IncrWebhooksSent()
	m.IncrWebhookErrors()
	m.IncrAuditEvents()
	m.IncrReconciliations()

	snap := m.Snapshot()
	if snap.RequestCount != 3 {
		t.Errorf("expected RequestCount=3, got %d", snap.RequestCount)
	}
	if snap.RequestErrors != 1 {
		t.Errorf("expected RequestErrors=1, got %d", snap.RequestErrors)
	}
	if snap.PolicyEvals != 2 {
		t.Errorf("expected PolicyEvals=2, got %d", snap.PolicyEvals)
	}
	if snap.PolicyDenied != 1 {
		t.Errorf("expected PolicyDenied=1, got %d", snap.PolicyDenied)
	}
	if snap.ZKProofs != 1 {
		t.Errorf("expected ZKProofs=1, got %d", snap.ZKProofs)
	}
	if snap.MPCSigns != 1 {
		t.Errorf("expected MPCSigns=1, got %d", snap.MPCSigns)
	}
	if snap.TXSubmitted != 1 {
		t.Errorf("expected TXSubmitted=1, got %d", snap.TXSubmitted)
	}
	if snap.TXConfirmed != 1 {
		t.Errorf("expected TXConfirmed=1, got %d", snap.TXConfirmed)
	}
	if snap.TXFailed != 1 {
		t.Errorf("expected TXFailed=1, got %d", snap.TXFailed)
	}
	if snap.WebhooksSent != 1 {
		t.Errorf("expected WebhooksSent=1, got %d", snap.WebhooksSent)
	}
	if snap.WebhookErrors != 1 {
		t.Errorf("expected WebhookErrors=1, got %d", snap.WebhookErrors)
	}
	if snap.AuditEvents != 1 {
		t.Errorf("expected AuditEvents=1, got %d", snap.AuditEvents)
	}
	if snap.Reconciliations != 1 {
		t.Errorf("expected Reconciliations=1, got %d", snap.Reconciliations)
	}
}

func TestMetricsCollector_RecordLatency(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordLatency(5 * time.Millisecond)
	m.RecordLatency(25 * time.Millisecond)
	m.RecordLatency(75 * time.Millisecond)
	m.RecordLatency(300 * time.Millisecond)
	m.RecordLatency(700 * time.Millisecond)
	m.RecordLatency(3 * time.Second)
	m.RecordLatency(10 * time.Second)

	snap := m.Snapshot()
	if snap.LatencyBuckets["<10ms"] != 1 {
		t.Errorf("expected 1 request in <10ms, got %d", snap.LatencyBuckets["<10ms"])
	}
	if snap.LatencyBuckets["10-50ms"] != 1 {
		t.Errorf("expected 1 request in 10-50ms, got %d", snap.LatencyBuckets["10-50ms"])
	}
	if snap.LatencyBuckets["50-100ms"] != 1 {
		t.Errorf("expected 1 request in 50-100ms, got %d", snap.LatencyBuckets["50-100ms"])
	}
	if snap.LatencyBuckets["100-500ms"] != 1 {
		t.Errorf("expected 1 request in 100-500ms, got %d", snap.LatencyBuckets["100-500ms"])
	}
	if snap.LatencyBuckets["500ms-1s"] != 1 {
		t.Errorf("expected 1 request in 500ms-1s, got %d", snap.LatencyBuckets["500ms-1s"])
	}
	if snap.LatencyBuckets["1s-5s"] != 1 {
		t.Errorf("expected 1 request in 1s-5s, got %d", snap.LatencyBuckets["1s-5s"])
	}
	if snap.LatencyBuckets[">5s"] != 1 {
		t.Errorf("expected 1 request in >5s, got %d", snap.LatencyBuckets[">5s"])
	}
}

func TestMetricsCollector_Uptime(t *testing.T) {
	m := NewMetricsCollector()
	time.Sleep(10 * time.Millisecond)
	snap := m.Snapshot()
	if snap.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}
