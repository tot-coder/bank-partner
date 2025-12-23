package services

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMetrics struct {
	transactionProcessed        *prometheus.CounterVec
	transactionDuration         prometheus.Histogram
	queueDepth                  *prometheus.GaugeVec
	retryAttempts               *prometheus.CounterVec
	circuitBreakerState         *prometheus.GaugeVec
	transfersTotal              *prometheus.CounterVec
	transferDuration            prometheus.Histogram
	transferAmount              prometheus.Histogram
	customerSearchRequests      *prometheus.CounterVec
	customerSearchDuration      prometheus.Histogram
	customerCreatedTotal        prometheus.Counter
	customerUpdatedTotal        *prometheus.CounterVec
	customerDeletedTotal        prometheus.Counter
	accountOwnershipTransferred prometheus.Counter
	activeCustomersTotal        prometheus.Gauge
	authenticationEventsTotal   *prometheus.CounterVec
}

func NewPrometheusMetrics() MetricsRecorderInterface {
	return &PrometheusMetrics{
		transactionProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "transaction_processing_total",
				Help: "Total number of transactions processed",
			},
			[]string{"operation", "status"},
		),
		transactionDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "transaction_processing_duration_milliseconds",
				Help:    "Transaction processing duration in milliseconds",
				Buckets: prometheus.ExponentialBuckets(1, 2, 12),
			},
		),
		queueDepth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "transaction_queue_depth",
				Help: "Current depth of transaction processing queue",
			},
			[]string{"status"},
		),
		retryAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "transaction_retry_attempts_total",
				Help: "Total number of transaction retry attempts",
			},
			[]string{"operation"},
		),
		circuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"service"},
		),
		transfersTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "transfers_total",
				Help: "Total number of transfers processed",
			},
			[]string{"status"},
		),
		transferDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "transfer_duration_milliseconds",
				Help:    "Transfer processing duration in milliseconds",
				Buckets: prometheus.ExponentialBuckets(1, 2, 12),
			},
		),
		transferAmount: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "transfer_amount",
				Help:    "Transfer amount in base currency units",
				Buckets: prometheus.ExponentialBuckets(1, 10, 8),
			},
		),
		customerSearchRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "customer_search_requests_total",
				Help: "Total number of customer search requests",
			},
			[]string{"status"},
		),
		customerSearchDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "customer_search_duration_seconds",
				Help:    "Customer search duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
		customerCreatedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "customer_created_total",
				Help: "Total number of customers created",
			},
		),
		customerUpdatedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "customer_updated_total",
				Help: "Total number of customer updates by field",
			},
			[]string{"field"},
		),
		customerDeletedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "customer_deleted_total",
				Help: "Total number of customers deleted",
			},
		),
		accountOwnershipTransferred: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "account_ownership_transferred_total",
				Help: "Total number of account ownership transfers",
			},
		),
		activeCustomersTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_customers_total",
				Help: "Current number of active customers",
			},
		),
		authenticationEventsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "authentication_events_total",
				Help: "Total number of authentication events",
			},
			[]string{"event_type"},
		),
	}
}

func (m *PrometheusMetrics) IncrementCounter(name string, tags map[string]string) {
	operation := tags["operation"]
	reason := tags["reason"]
	status := tags["status"]

	switch name {
	case "transaction.processed.success":
		m.transactionProcessed.WithLabelValues(operation, "success").Inc()
	case "transaction.processed.failed":
		m.transactionProcessed.WithLabelValues(operation, "failed_"+reason).Inc()
	case "transaction.processing.retry":
		m.retryAttempts.WithLabelValues(operation).Inc()
	case "transaction.duplicate.rejected":
		m.transactionProcessed.WithLabelValues("", "duplicate").Inc()
	case "queue.enqueued":
		m.queueDepth.WithLabelValues("pending").Inc()
	case "circuit_breaker.open":
		m.circuitBreakerState.WithLabelValues(tags["service"]).Set(1)
	case "transfers_total":
		if status != "" {
			m.transfersTotal.WithLabelValues(status).Inc()
		}
	case "customer_search_request":
		if status != "" {
			m.customerSearchRequests.WithLabelValues(status).Inc()
		}
	case "customer_created":
		m.customerCreatedTotal.Inc()
	case "customer_updated":
		if field := tags["field"]; field != "" {
			m.customerUpdatedTotal.WithLabelValues(field).Inc()
		}
	case "customer_deleted":
		m.customerDeletedTotal.Inc()
	case "account_ownership_transferred":
		m.accountOwnershipTransferred.Inc()
	case "authentication_event":
		if eventType := tags["event_type"]; eventType != "" {
			m.authenticationEventsTotal.WithLabelValues(eventType).Inc()
		}
	}
}

func (m *PrometheusMetrics) RecordProcessingTime(name string, duration time.Duration) {
	switch name {
	case "transaction.processing":
		m.transactionDuration.Observe(float64(duration.Milliseconds()))
	case "transfer_duration_success", "transfer_duration_failed":
		m.transferDuration.Observe(float64(duration.Milliseconds()))
	case "customer_search":
		m.customerSearchDuration.Observe(duration.Seconds())
	}
}

func (m *PrometheusMetrics) RecordGauge(name string, value float64, tags map[string]string) {
	status := tags["status"]
	switch name {
	case "transfer_amount":
		m.transferAmount.Observe(value)
	case "active_customers":
		m.activeCustomersTotal.Set(value)
	default:
		if status != "" {
			m.queueDepth.WithLabelValues(status).Set(value)
		}
	}
}

func (m *PrometheusMetrics) UpdateQueueMetrics(pending, processing, failed int64) {
	m.queueDepth.WithLabelValues("pending").Set(float64(pending))
	m.queueDepth.WithLabelValues("processing").Set(float64(processing))
	m.queueDepth.WithLabelValues("failed").Set(float64(failed))
}
