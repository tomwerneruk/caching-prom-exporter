package metrics

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// create an interface as a template for each metric we want
type Metric interface {
	// Name of the Metric. Should be unique.
	Name() string

	// Help describes the role of the Metric.
	Help() string

	// Scrape collects data and sends it over channel as prometheus metric
	Scrape(ctx context.Context, ch chan<- prometheus.Metric, logger log.Logger) error
}

// MetricExporter collects MySQL metrics. It implements prometheus.Collector.
type MetricExporter struct {
	ctx     context.Context
	logger  log.Logger
	metrics []Metric
}

var _ prometheus.Collector = (*MetricExporter)(nil)

var (
	metricScrapeDurationSeconds = prometheus.NewDesc(
		prometheus.BuildFQName("namespace", "exporter", "collector_duration_seconds"),
		"CWhether a collector succeeded",
		[]string{"collector"}, nil,
	)
	metricScrapeCollectorSuccess = prometheus.NewDesc(
		prometheus.BuildFQName("namespace", "exporter", "collector_success"),
		"Collector time duration.",
		[]string{"collector"}, nil,
	)
)

// New returns a new MySQL MetricExporter for the provided DSN.
func New(ctx context.Context, metrics []Metric, logger log.Logger) *MetricExporter {
	return &MetricExporter{
		ctx:     ctx,
		logger:  logger,
		metrics: metrics,
	}
}

// Describe implements prometheus.Collector.
func (e *MetricExporter) Describe(ch chan<- *prometheus.Desc) {
	// ch <- mysqlUp
}

// Collect implements prometheus.Collector.
func (e *MetricExporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(e.ctx, ch)
}

// scrape collects metrics from the target, returns an up metric value.
func (e *MetricExporter) scrape(ctx context.Context, ch chan<- prometheus.Metric) float64 {
	scrapeTime := time.Now()

	ch <- prometheus.MustNewConstMetric(metricScrapeDurationSeconds, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "connection")

	var wg sync.WaitGroup
	defer wg.Wait()
	for _, metric := range e.metrics {
		wg.Add(1)
		go func(metric Metric) {
			defer wg.Done()
			label := "collect." + metric.Name()
			scrapeTime := time.Now()
			collectorSuccess := 1.0
			if err := metric.Scrape(ctx, ch, log.Logger{}); err != nil {
				log.Println("msg", "Error from metric", "metric", metric.Name(), "target", "err", err)
				collectorSuccess = 0.0
			}
			ch <- prometheus.MustNewConstMetric(metricScrapeCollectorSuccess, prometheus.GaugeValue, collectorSuccess, label)
			ch <- prometheus.MustNewConstMetric(metricScrapeDurationSeconds, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), label)
		}(metric)
	}
	return 1.0
}
