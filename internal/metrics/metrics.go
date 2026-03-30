package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	MessagesSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wpp_messages_sent_total",
			Help: "Total messages sent by group and instance",
		},
		[]string{"group_id", "instance_id", "status"},
	)

	MessagesDeliveryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wpp_messages_delivery_duration_seconds",
			Help:    "Time from sent to delivered",
			Buckets: prometheus.ExponentialBuckets(0.5, 2, 10),
		},
		[]string{"group_id", "instance_id"},
	)

	MessagesInQueue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wpp_messages_in_queue",
			Help: "Messages currently in queue per group",
		},
		[]string{"group_id"},
	)

	InstanceStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wpp_instance_status",
			Help: "Instance status (1=active, 0=inactive)",
		},
		[]string{"group_id", "instance_id", "status"},
	)

	InstanceBudgetRemaining = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wpp_instance_budget_remaining",
			Help: "Remaining daily budget per instance",
		},
		[]string{"group_id", "instance_id"},
	)

	InstanceDeliveryRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wpp_instance_delivery_rate",
			Help: "Current delivery rate per instance",
		},
		[]string{"group_id", "instance_id"},
	)

	InstanceWebSocketLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wpp_instance_websocket_latency_ms",
			Help:    "WebSocket ping latency in milliseconds",
			Buckets: prometheus.ExponentialBuckets(10, 2, 8),
		},
		[]string{"instance_id"},
	)

	GroupActiveInstances = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wpp_group_active_instances",
			Help: "Number of active instances per group",
		},
		[]string{"group_id"},
	)

	GroupFailoversTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wpp_group_failovers_total",
			Help: "Total failover events per group",
		},
		[]string{"group_id"},
	)

	BroadcastsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wpp_broadcasts_active",
			Help: "Number of currently active broadcasts",
		},
	)

	WebhookDeliveries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wpp_webhook_deliveries_total",
			Help: "Total webhook delivery attempts",
		},
		[]string{"status"},
	)

	AntibanSignals = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wpp_antiban_signals_total",
			Help: "Anti-ban signals detected",
		},
		[]string{"type", "action"},
	)
)

func Register() {
	prometheus.MustRegister(
		MessagesSentTotal,
		MessagesDeliveryDuration,
		MessagesInQueue,
		InstanceStatus,
		InstanceBudgetRemaining,
		InstanceDeliveryRate,
		InstanceWebSocketLatency,
		GroupActiveInstances,
		GroupFailoversTotal,
		BroadcastsActive,
		WebhookDeliveries,
		AntibanSignals,
	)
}
