package metrics

import (
	"context"
	"log"
	"math/rand"

	"github.com/jellydator/ttlcache/v3"
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
		prometheus.BuildFQName("my_namespace", "aws_stats", "ebs_volume_count"),
		"EBS Volume Count",
		[]string{"account_id"}, nil,
	)
)

func (MetricEbsVolumeCount) Scrape(ctx context.Context, ch chan<- prometheus.Metric, logger log.Logger, cache *ttlcache.Cache[string, float64]) error {
	ch <- prometheus.MustNewConstMetric(
		EbsVolumeCountDesc,
		prometheus.GaugeValue,
		cache.Get("ebsVolumeCount").Value(),
		"",
	)

	return nil
}

func (MetricEbsVolumeCount) ExternalDataPull() float64 {
	return rand.Float64()
}

var _ Metric = MetricEbsVolumeCount{}
