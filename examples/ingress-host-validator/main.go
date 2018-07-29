package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"

	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	whhttp "github.com/slok/kubewebhook/pkg/http"
	"github.com/slok/kubewebhook/pkg/log"
	validatingwh "github.com/slok/kubewebhook/pkg/webhook/validating"
)

type ingressHostValidator struct {
	hostRegex *regexp.Regexp
	logger    log.Logger
}

func (v *ingressHostValidator) Validate(_ context.Context, obj metav1.Object) (bool, validatingwh.ValidatorResult, error) {
	ingress, ok := obj.(*extensionsv1beta1.Ingress)

	if !ok {
		return false, validatingwh.ValidatorResult{}, fmt.Errorf("not an ingress")
	}

	for _, r := range ingress.Spec.Rules {
		if !v.hostRegex.MatchString(r.Host) {
			v.logger.Infof("ingress %s denied, host %s is not valid for regex %s", ingress.Name, r.Host, v.hostRegex)
			res := validatingwh.ValidatorResult{
				Valid:   false,
				Message: fmt.Sprintf("%s ingress host doesn't match %s regex", r.Host, v.hostRegex),
			}
			return false, res, nil
		}
	}

	v.logger.Infof("ingress %s is valid", ingress.Name)
	res := validatingwh.ValidatorResult{
		Valid:   true,
		Message: "all hosts in the ingress are valid",
	}
	return false, res, nil
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

	fl.Parse(os.Args[1:])
	return cfg
}

func main() {
	logger := &log.Std{Debug: true}

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

	vcfg := validatingwh.WebhookConfig{
		Name: "ingressHostValidator",
		Obj:  &extensionsv1beta1.Ingress{},
	}
	wh, err := validatingwh.NewWebhook(vcfg, vl, nil, nil, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating webhook: %s", err)
		os.Exit(1)
	}

	// Serve the webhook.
	logger.Infof("Listening on %s", cfg.addr)
	err = http.ListenAndServeTLS(cfg.addr, cfg.certFile, cfg.keyFile, whhttp.MustHandlerFor(wh))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error serving webhook: %s", err)
		os.Exit(1)
	}
}
