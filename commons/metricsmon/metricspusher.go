package metricsmon

import (
	"time"

	"github.com/IBM/FfDL/commons/config"

	"github.com/go-kit/kit/metrics/statsd"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	log "github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
)

//StartMetricsPusher ...
func StartMetricsPusher(label string, interval time.Duration, url string) chan struct{} {
	log.Info("Starting code to push out metrics")
	quit := make(chan struct{})
	ticker := time.NewTicker(interval)
	go func() {
		settings := gobreaker.Settings{
			Name: "PUSH METRICS",
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				return counts.Requests >= 3 && failureRatio >= 0.6
			},
			OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
				log.Infof("Circuit breaker  %s changing from status %v to state %v", name, from, to)
			},
			Timeout: 3 * time.Minute,
		}
		cb := gobreaker.NewCircuitBreaker(settings)
		for {
			select {
			case <-ticker.C:
				cb.Execute(func() (interface{}, error) {
					err := pushMetrics(label, url)
					return nil, err
				})
			case <-quit:
				log.Info("request to stop the metrics pusher for url %s", url)
				ticker.Stop()
				return
			}
		}
	}()

	return quit
}

func pushMetrics(job string, url string) error {
	labels := push.HostnameGroupingKey()
	labels["namespace"] = config.GetPodNamespaceForPrometheus()
	if err :=
		push.AddFromGatherer(job, labels, url, stdprometheus.DefaultGatherer); err != nil {
		log.WithError(err).Warnf("Failed to push metrics to url %s", url)
		return err
	}
	return nil
}

//StartStatsdMetricsPusher ... pushes metrics out to statsd server every 30s
func StartStatsdMetricsPusher(statsd *statsd.Statsd, pushInterval time.Duration) {
	log.Info("Starting code to push out metrics via statsd")
	report := time.NewTicker(pushInterval)
	//TODO
	//defer report.Stop()
	go statsd.SendLoop(report.C, "udp", "statsdexporter:9125")
}
