package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/tomwerneruk/caching-prom-exporter/internal/datasource"
)

var (
	ebsVolumes = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "ebs_volumes_count",
		Help:      "The total number of ebs volumes in account",
		Namespace: "account_trends",
	})
	ebsSnapshots = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "ebs_snapshots_count",
		Help:      "The total number of ebs snapshots in account",
		Namespace: "account_trends",
	})
)

func main() {

	// define a loader function, this is used for populating cache items when it is empty / expired
	loader := ttlcache.LoaderFunc[string, float64](
		func(c *ttlcache.Cache[string, float64], key string) *ttlcache.Item[string, float64] {
			log.Println("Refreshing data from remote source")
			log.Println(fmt.Sprintf("Populating key %s", key))
			switch key {
			case "ebs_vol_count":
				return c.Set("ebs_vol_count", datasource.GetEbsCount(), time.Second*45)
			case "ebs_snap_count":
				return c.Set("ebs_snap_count", datasource.GetSnapCount(), time.Second*45)
			}

			return nil
		},
	)

	// setup an in-memory cache that has a default 30m expiry and will evict expired items
	cache := ttlcache.New[string, float64](
		ttlcache.WithLoader[string, float64](loader),
		ttlcache.WithTTL[string, float64](30*time.Minute),
		ttlcache.WithDisableTouchOnHit[string, float64](),
	)

	cache.OnEviction(func(ctx context.Context, reason ttlcache.EvictionReason, item *ttlcache.Item[string, float64]) {
		log.Println(fmt.Sprintf("Evicting %s", item.Key()))
	})

	// Start the async cache worker. This manages item expiry in the cache.
	go cache.Start()

	// setup a regular tick to populate our prom metrics with the cache. This may end up doing a lazy
	// refresh of the cache item via the loader, if required
	s := gocron.NewScheduler(time.UTC)

	createMetricRefresh(ebsVolumes, "ebs_vol_count", 30, s, cache)
	createMetricRefresh(ebsSnapshots, "ebs_snap_count", 30, s, cache)

	s.StartAsync()

	// serve our prom client
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

}

func createMetricRefresh(metric prometheus.Gauge, cacheKey string, ttl int, cron_scheduler *gocron.Scheduler, cache_ref *ttlcache.Cache[string, float64]) {

	cron_scheduler.Every(30).Seconds().Do(func() {
		log.Println(fmt.Sprintf("Refreshing gauge for %s", cacheKey))
		item := cache_ref.Get(cacheKey)
		metric.Set(item.Value())
		log.Printf("Item %s Expiry: %s", item.Key(), item.ExpiresAt())
	})

}
