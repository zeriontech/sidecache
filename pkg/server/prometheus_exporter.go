package server

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	buildInfoGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sidecache_admission_build_info",
			Help: "Build info for sidecache admission webhook",
		}, []string{"version"})

	cacheHitCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache_" + ProjectName,
			Name:      "cache_hit_counter",
			Help:      "Cache hit count",
		})

	lockAcquiringAttemptsHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "sidecache_" + ProjectName,
			Name:      "lock_acquiring_attempts_histogram",
			Help:      "Lock acquiring attempts histogram",
			Buckets:   []float64{0.999, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 15, 20, 100, 200, 300, 500, 1000},
		})

	totalRequestCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache_" + ProjectName,
			Name:      "all_request_hit_counter",
			Help:      "All request hit counter",
		})
)

type Prometheus struct {
	CacheHitCounter                prometheus.Counter
	LockAcquiringAttemptsHistogram prometheus.Histogram
	TotalRequestCounter            prometheus.Counter
}

func NewPrometheusClient() *Prometheus {
	prometheus.MustRegister(buildInfoGaugeVec, cacheHitCounter, lockAcquiringAttemptsHistogram, totalRequestCounter)

	return &Prometheus{
		CacheHitCounter:                cacheHitCounter,
		LockAcquiringAttemptsHistogram: lockAcquiringAttemptsHistogram,
		TotalRequestCounter:            totalRequestCounter,
	}
}

func BuildInfo(admission string) {
	isNotEmptyAdmissionVersion := len(strings.TrimSpace(admission)) > 0

	if isNotEmptyAdmissionVersion {
		buildInfoGaugeVec.WithLabelValues(admission)
	}
}
