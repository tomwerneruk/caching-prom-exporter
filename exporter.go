package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tomwerneruk/caching-prom-exporter/internal/metrics"
	"golang.org/x/exp/maps"
)

var timeoutOffset float64 = 0.25

func main() {

	var availableMetrics = map[string]metrics.Metric{
		"ebsVolumeCount": metrics.MetricEbsVolumeCount{},
	}

	// define a loader function, this is used for populating cache items when it is empty / expired
	// This invokes the ExternalDataPull method on a metric under metrics package. This allows custom
	// funcationality to get the value
	loader := ttlcache.LoaderFunc[string, float64](
		func(c *ttlcache.Cache[string, float64], key string) *ttlcache.Item[string, float64] {
			log.Println("Refreshing data from remote source")
			log.Println(fmt.Sprintf("Populating key %s", key))
			item := c.Set(key, availableMetrics[key].ExternalDataPull(), time.Second*3)
			return item
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
	s.StartAsync()

	availableMetricsList := maps.Values(availableMetrics)

	// serve our prom client
	http.Handle("/metrics", promhttp.InstrumentMetricHandler(prometheus.DefaultRegisterer, promHandler(availableMetricsList, log.Logger{}, cache)))
	log.Println("Starting server")
	http.ListenAndServe(":2112", nil)

}

func promHandler(scrapers []metrics.Metric, logger log.Logger, cache_ref *ttlcache.Cache[string, float64]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Use request context for cancellation when connection gets closed.
		ctx := r.Context()
		// If a timeout is configured via the Prometheus header, add it to the context.
		if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
			timeoutSeconds, err := strconv.ParseFloat(v, 64)
			if err != nil {
				log.Fatalln("msg", "Failed to parse timeout from Prometheus header", "err", err)
			} else {
				if timeoutOffset >= timeoutSeconds {
					// Ignore timeout offset if it doesn't leave time to scrape.
					log.Fatalln("msg", "Timeout offset should be lower than prometheus scrape timeout", "offset", timeoutOffset, "prometheus_scrape_timeout", timeoutSeconds)
				} else {
					// Subtract timeout offset from timeout.
					timeoutSeconds -= timeoutOffset
				}
				// Create new timeout context with request context as parent.
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds*float64(time.Second)))
				defer cancel()
				// Overwrite request with timeout context.
				r = r.WithContext(ctx)
			}
		}

		registry := prometheus.NewRegistry()

		ctxWithCache := context.WithValue(ctx, "cacheRef", cache_ref)
		registry.MustRegister(metrics.New(ctxWithCache, scrapers, log.Logger{}))

		gatherers := prometheus.Gatherers{
			prometheus.DefaultGatherer,
			registry,
		}
		// Delegate http serving to Prometheus client library, which will call metrics.Collect.
		h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}
