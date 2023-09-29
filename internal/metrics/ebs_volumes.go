package metrics

import (
	"context"
	"log"
	"math/rand"

	"github.com/prometheus/client_golang/prometheus"
)

type MetricEbsVolumeCount struct{}

func (MetricEbsVolumeCount) Name() string {
	return "ebs_volume_count"
}

func (MetricEbsVolumeCount) Help() string {
	return ""
}

var (
	EbsVolumeCountDesc = prometheus.NewDesc(
		prometheus.BuildFQName("namespace", "aws_stats", "ebs_volume_count"),
		"EBS Volume Count",
		[]string{"account_id"}, nil,
	)
)

func (MetricEbsVolumeCount) Scrape(ctx context.Context, ch chan<- prometheus.Metric, logger log.Logger) error {
	ch <- prometheus.MustNewConstMetric(
		EbsVolumeCountDesc,
		prometheus.GaugeValue,
		rand.Float64(),
		"123455673434",
	)

	return nil
}

var _ Metric = MetricEbsVolumeCount{}
