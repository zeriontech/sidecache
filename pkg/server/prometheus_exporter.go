package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

var (
	cacheHitCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache_" + ProjectName,
			Name:      "cache_hit_counter",
			Help:      "Cache hit count",
		})

	totalRequestCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache_" + ProjectName,
			Name:      "all_request_hit_counter",
			Help:      "All request hit counter",
		})

	buildInfoGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sidecache_admission_build_info",
			Help: "Build info for sidecache admission webhook",
		}, []string{"version"})
)

type Prometheus struct {
	CacheHitCounter     prometheus.Counter
	TotalRequestCounter prometheus.Counter
}

func NewPrometheusClient() *Prometheus {
	prometheus.MustRegister(cacheHitCounter, totalRequestCounter, buildInfoGaugeVec)

	return &Prometheus{TotalRequestCounter: totalRequestCounter, CacheHitCounter: cacheHitCounter}
}

func BuildInfo(admission string) {
	isNotEmptyAdmissionVersion := len(strings.TrimSpace(admission)) > 0

	if isNotEmptyAdmissionVersion {
		buildInfoGaugeVec.WithLabelValues(admission)
	}
}
