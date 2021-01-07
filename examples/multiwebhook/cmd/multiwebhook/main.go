package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	kwhprometheus "github.com/slok/kubewebhook/v2/pkg/metrics/prometheus"
	kwhwebhook "github.com/slok/kubewebhook/v2/pkg/webhook"

	"github.com/slok/kubewebhook/v2/examples/multiwebhook/pkg/webhook/mutating"
	"github.com/slok/kubewebhook/v2/examples/multiwebhook/pkg/webhook/validating"
)

const (
	gracePeriod = 3 * time.Second
	minReps     = 1
	maxReps     = 12
)

var (
	defLabels = map[string]string{
		"webhook": "multiwebhook",
		"test":    "kubewebhook",
	}
)

// Main is the main program.
type Main struct {
	flags  *Flags
	logger kwhlog.Logger
	stopC  chan struct{}
}

// Run will run the main program.
func (m *Main) Run() error {

	logrusLogEntry := logrus.NewEntry(logrus.New())
	logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	m.logger = kwhlogrus.NewLogrus(logrusLogEntry)

	// Create services.
	promReg := prometheus.NewRegistry()
	metricsRec, err := kwhprometheus.NewRecorder(kwhprometheus.RecorderConfig{Registry: promReg})
	if err != nil {
		return fmt.Errorf("could not create prometheus recorder: %w", err)
	}

	// Create webhooks
	mpw, err := mutating.NewPodWebhook(defLabels, m.logger)
	if err != nil {
		return err
	}
	mpw = kwhwebhook.NewMeasuredWebhook(metricsRec, mpw)
	mpwh, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: mpw, Logger: m.logger})
	if err != nil {
		return err
	}

	vdw, err := validating.NewDeploymentWebhook(minReps, maxReps, m.logger)
	if err != nil {
		return err
	}
	vdw = kwhwebhook.NewMeasuredWebhook(metricsRec, vdw)
	vdwh, err := kwhhttp.HandlerFor(kwhhttp.HandlerConfig{Webhook: vdw, Logger: m.logger})
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
