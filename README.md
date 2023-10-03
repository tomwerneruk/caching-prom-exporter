# caching-prom-exporter

Proof of concept prometheus exporter strongly modelled on https://github.com/prometheus/mysqld_exporter/tree/main.

This is a boilerplate for a prom exporter that has a built in cache for metric data. This could be useful when the metric source is pulled from somewhere on a long poll. This means that the prom metrics thread won't block and hit a timeout.

Caching works on a TTL basis, with items loaded asynchrousnly as they expire.

The prom collector implementation pulls the data from the cache that is responsible for refreshing if it is stale. 

## Adding a new metric

* add a new metric to `var availableMetrics` in `exporter.go`
* copy `internal/metrics/ebs_volumes.go` to a new file, change the struct name i.e `MetricEbsVolumeCount` and the cache key i.e `ebsVolumeCount`
* change the function for `ExternalDataPull()` to pull your data from a remote, slow source i.e AWS API Call