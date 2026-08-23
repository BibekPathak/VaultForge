package core

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector holds application-level metrics in memory.
type MetricsCollector struct {
	requestCount    atomic.Int64
	requestErrors   atomic.Int64
	policyEvals     atomic.Int64
	policyDenied    atomic.Int64
	zkProofs        atomic.Int64
	mpcSigns        atomic.Int64
	txSubmitted     atomic.Int64
	txConfirmed     atomic.Int64
	txFailed        atomic.Int64
	webhooksSent    atomic.Int64
	webhookErrors   atomic.Int64
	auditEvents     atomic.Int64
	reconciliations atomic.Int64

	latencyBuckets sync.Map // bucket -> count

	startTime time.Time
}

// MetricsSnapshot is a point-in-time snapshot of all metrics.
type MetricsSnapshot struct {
	Uptime          string         `json:"uptime"`
	RequestCount    int64          `json:"request_count"`
	RequestErrors   int64          `json:"request_errors"`
	PolicyEvals     int64          `json:"policy_evaluations"`
	PolicyDenied    int64          `json:"policy_denied"`
	ZKProofs        int64          `json:"zk_proofs"`
	MPCSigns        int64          `json:"mpc_signs"`
	TXSubmitted     int64          `json:"transactions_submitted"`
	TXConfirmed     int64          `json:"transactions_confirmed"`
	TXFailed        int64          `json:"transactions_failed"`
	WebhooksSent    int64          `json:"webhooks_sent"`
	WebhookErrors   int64          `json:"webhook_errors"`
	AuditEvents     int64          `json:"audit_events"`
	Reconciliations int64          `json:"reconciliations"`
	LatencyBuckets  map[string]int `json:"latency_buckets"`
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now().UTC(),
	}
}

func (m *MetricsCollector) IncrRequestCount()      { m.requestCount.Add(1) }
func (m *MetricsCollector) IncrRequestErrors()     { m.requestErrors.Add(1) }
func (m *MetricsCollector) IncrPolicyEvals()       { m.policyEvals.Add(1) }
func (m *MetricsCollector) IncrPolicyDenied()      { m.policyDenied.Add(1) }
func (m *MetricsCollector) IncrZKProofs()          { m.zkProofs.Add(1) }
func (m *MetricsCollector) IncrMPCSigns()          { m.mpcSigns.Add(1) }
func (m *MetricsCollector) IncrTXSubmitted()       { m.txSubmitted.Add(1) }
func (m *MetricsCollector) IncrTXConfirmed()       { m.txConfirmed.Add(1) }
func (m *MetricsCollector) IncrTXFailed()          { m.txFailed.Add(1) }
func (m *MetricsCollector) IncrWebhooksSent()      { m.webhooksSent.Add(1) }
func (m *MetricsCollector) IncrWebhookErrors()     { m.webhookErrors.Add(1) }
func (m *MetricsCollector) IncrAuditEvents()       { m.auditEvents.Add(1) }
func (m *MetricsCollector) IncrReconciliations()   { m.reconciliations.Add(1) }

// RecordLatency records a request latency into predefined buckets.
func (m *MetricsCollector) RecordLatency(d time.Duration) {
	var bucket string
	ms := d.Milliseconds()
	switch {
	case ms < 10:
		bucket = "<10ms"
	case ms < 50:
		bucket = "10-50ms"
	case ms < 100:
		bucket = "50-100ms"
	case ms < 500:
		bucket = "100-500ms"
	case ms < 1000:
		bucket = "500ms-1s"
	case ms < 5000:
		bucket = "1s-5s"
	default:
		bucket = ">5s"
	}
	val, _ := m.latencyBuckets.LoadOrStore(bucket, &atomic.Int64{})
	val.(*atomic.Int64).Add(1)
}

// Snapshot returns a point-in-time copy of all metrics.
func (m *MetricsCollector) Snapshot() MetricsSnapshot {
	buckets := make(map[string]int)
	m.latencyBuckets.Range(func(key, value any) bool {
		buckets[key.(string)] = int(value.(*atomic.Int64).Load())
		return true
	})

	return MetricsSnapshot{
		Uptime:          time.Since(m.startTime).Round(time.Second).String(),
		RequestCount:    m.requestCount.Load(),
		RequestErrors:   m.requestErrors.Load(),
		PolicyEvals:     m.policyEvals.Load(),
		PolicyDenied:    m.policyDenied.Load(),
		ZKProofs:        m.zkProofs.Load(),
		MPCSigns:        m.mpcSigns.Load(),
		TXSubmitted:     m.txSubmitted.Load(),
		TXConfirmed:     m.txConfirmed.Load(),
		TXFailed:        m.txFailed.Load(),
		WebhooksSent:    m.webhooksSent.Load(),
		WebhookErrors:   m.webhookErrors.Load(),
		AuditEvents:     m.auditEvents.Load(),
		Reconciliations: m.reconciliations.Load(),
		LatencyBuckets:  buckets,
	}
}

// MetricsMiddleware returns an HTTP middleware that records request metrics.
func (m *MetricsCollector) MetricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			elapsed := time.Since(start)
			m.IncrRequestCount()
			m.RecordLatency(elapsed)
		})
	}
}
