package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/log"
	"github.com/slok/kubewebhook/pkg/observability/metrics"
	jaeger "github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"

	"github.com/slok/kubewebhook/examples/multiwebhook/pkg/webhook/mutating"
	"github.com/slok/kubewebhook/examples/multiwebhook/pkg/webhook/validating"
)

const (
	gracePeriod   = 3 * time.Second
	jaegerService = "multi-webhook"
)

var (
	defLabels = map[string]string{
		"webhook": "multiwebhook",
		"test":    "kubewebhook",
	}
	minReps = 1
	maxReps = 12
)

// Main is the main program.
type Main struct {
	flags  *Flags
	logger log.Logger
	stopC  chan struct{}
}

// Run will run the main program.
func (m *Main) Run() error {

	m.logger = &log.Std{
		Debug: m.flags.Debug,
	}

	// Create services.
	promReg := prometheus.NewRegistry()
	metricsRec := metrics.NewPrometheus(promReg)
	tracer, closer, err := m.createTracer(jaegerService)
	if err != nil {
		return err
	}
	defer closer.Close()

	// Create webhooks
	mpw, err := mutating.NewPodWebhook(defLabels, tracer, metricsRec, m.logger)
	if err != nil {
		return err
	}
	mpwh, err := whhttp.HandlerFor(mpw)
	if err != nil {
		return err
	}

	vdw, err := validating.NewDeploymentWebhook(minReps, maxReps, tracer, metricsRec, m.logger)
	if err != nil {
		return err
	}
	vdwh, err := whhttp.HandlerFor(vdw)
	if err != nil {
		return err
	}

	// Create the servers and set them listenig.
	errC := make(chan error)

	// Serve webhooks.
	// TODO: Move to it's own service.
	go func() {

		m.logger.Infof("webhooks listening on %s...", m.flags.ListenAddress)
		mux := http.NewServeMux()
		mux.Handle("/webhooks/mutating/pod", mpwh)
		mux.Handle("/webhooks/validating/deployment", vdwh)
		errC <- http.ListenAndServeTLS(
			m.flags.ListenAddress,
			m.flags.CertFile,
			m.flags.KeyFile,
			mux,
		)
	}()

	// Serve metrics.
	metricsHandler := promhttp.HandlerFor(promReg, promhttp.HandlerOpts{})
	go func() {
		m.logger.Infof("metrics listening on %s...", m.flags.MetricsListenAddress)
		errC <- http.ListenAndServe(m.flags.MetricsListenAddress, metricsHandler)
	}()

	// Run everything
	defer m.stop()

	sigC := m.createSignalChan()
	select {
	case err := <-errC:
		if err != nil {
			m.logger.Errorf("error received: %s", err)
			return err
		}
		m.logger.Infof("app finished successfuly")
	case s := <-sigC:
		m.logger.Infof("signal %s received", s)
		return nil
	}

	return nil
}

func (m *Main) stop() {
	m.logger.Infof("stopping everything, waiting %s...", gracePeriod)

	close(m.stopC)

	// Stop everything and let them time to stop.
	time.Sleep(gracePeriod)
}

func (m *Main) createSignalChan() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	return c
}

func (m *Main) createTracer(service string) (opentracing.Tracer, io.Closer, error) {
	cfg := &jaegerconfig.Configuration{
		ServiceName: service,
		Sampler: &jaegerconfig.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jaegerconfig.ReporterConfig{
			LogSpans: true,
		},
	}
	tracer, closer, err := cfg.NewTracer(jaegerconfig.Logger(jaeger.NullLogger))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot init Jaeger: %s", err)
	}
	return tracer, closer, nil
}

func main() {
	m := Main{
		flags: NewFlags(),
		stopC: make(chan struct{}),
	}

	err := m.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
	os.Exit(0)

}
