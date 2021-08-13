package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"

	"github.com/sirupsen/logrus"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhlogrus "github.com/slok/kubewebhook/v2/pkg/log/logrus"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhvalidating "github.com/slok/kubewebhook/v2/pkg/webhook/validating"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ingressHostValidator struct {
	hostRegex *regexp.Regexp
	logger    kwhlog.Logger
}

func (v *ingressHostValidator) Validate(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhvalidating.ValidatorResult, error) {
	ingress, ok := obj.(*extensionsv1beta1.Ingress)

	if !ok {
		return nil, fmt.Errorf("not an ingress")
	}

	for _, r := range ingress.Spec.Rules {
		if !v.hostRegex.MatchString(r.Host) {
			v.logger.Infof("ingress %s denied, host %s is not valid for regex %s", ingress.Name, r.Host, v.hostRegex)
			return &kwhvalidating.ValidatorResult{
				Valid:   false,
				Message: fmt.Sprintf("%s ingress host doesn't match %s regex", r.Host, v.hostRegex),
			}, nil
		}
	}

	v.logger.Infof("ingress %s is valid", ingress.Name)
	return &kwhvalidating.ValidatorResult{
		Valid:   true,
		Message: "all hosts in the ingress are valid",
	}, nil
}

type config struct {
	certFile  string
	keyFile   string
	hostRegex string
	addr      string
}

func initFlags() *config {
	cfg := &config{}

	fl := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fl.StringVar(&cfg.certFile, "tls-cert-file", "", "TLS certificate file")
	fl.StringVar(&cfg.keyFile, "tls-key-file", "", "TLS key file")
	fl.StringVar(&cfg.addr, "listen-addr", ":8080", "The address to start the server")
	fl.StringVar(&cfg.hostRegex, "ingress-host-regex", "", "The ingress host regex that matches valid ingresses")

	_ = fl.Parse(os.Args[1:])
	return cfg
}

func main() {
	logrusLogEntry := logrus.NewEntry(logrus.New())
	logrusLogEntry.Logger.SetLevel(logrus.DebugLevel)
	logger := kwhlogrus.NewLogrus(logrusLogEntry)

	cfg := initFlags()

	// Create our validator
	rgx, err := regexp.Compile(cfg.hostRegex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid regex: %s", err)
		os.Exit(1)
		return
	}
	vl := &ingressHostValidator{
		hostRegex: rgx,
		logger:    logger,
	}

	vcfg := kwhvalidating.WebhookConfig{
		ID:        "ingressHostValidator",
		Obj:       &extensionsv1beta1.Ingress{},
		Validator: vl,
		Logger:    logger,
	}
	wh, err := kwhvalidating.NewWebhook(vcfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook: %s", err)
		os.Exit(1)
	}

	// Serve the webhook.
	logger.Infof("Listening on %s", cfg.addr)
	err = http.ListenAndServeTLS(cfg.addr, cfg.certFile, cfg.keyFile, kwhhttp.MustHandlerFor(kwhhttp.HandlerConfig{
		Webhook: wh,
		Logger:  logger,
	}))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error serving webhook: %s", err)
		os.Exit(1)
	}
}
