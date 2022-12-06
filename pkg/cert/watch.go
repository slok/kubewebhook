package cert

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type Certificate struct {
	currentCert *tls.Certificate
	sync.RWMutex
	watcher *fsnotify.Watcher

	certFilePath string
	keyFilePath  string
}

// New parses and reads the supplied certificate and key.
func New(certFile string, keyFile string) (*Certificate, error) {
	crt := &Certificate{
		certFilePath: certFile,
		keyFilePath:  keyFile,
	}

	if err := crt.loadCert(); err != nil {
		return nil, fmt.Errorf("unable to load certificate: %w", err)
	}

	return crt, nil
}

// loadCert reads the certificate and key files, parses them,
// and updates the current TLS certificate.
func (crt *Certificate) loadCert() error {
	c, err := tls.LoadX509KeyPair(crt.certFilePath, crt.keyFilePath)
	if err != nil {
		return err
	}

	crt.Lock()
	crt.currentCert = &c
	crt.Unlock()

	fmt.Println("Updated current TLS certificate")

	return nil
}

// GetCertificate fetches the currently loaded certificate, which may be nil.
func (crt *Certificate) GetCert(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	crt.RLock()
	defer crt.RUnlock()
	return crt.currentCert, nil
}

// Start starts monitoring certificate and private key changes.
func (crt *Certificate) Start(ctx context.Context) error {
	var err error
	crt.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	crts := []string{crt.certFilePath, crt.keyFilePath}
	for _, c := range crts {
		if err := crt.watcher.Add(c); err != nil {
			return err
		}
	}

	go crt.watch()

	// Wait until the context is done
	<-ctx.Done()

	return crt.watcher.Close()
}

// If an event occurs as a result of file monitoring, read from the channels
func (crt *Certificate) watch() {
	for {
		select {
		case event := <-crt.watcher.Events:
			crt.createOrUpdateEvent(event)
		case err := <-crt.watcher.Errors:
			log.Fatal("Certificate Watch Error%w", err)
		}
	}
}

// If the file is recreated, start watch anew.
// If the file content has been modified, read it anew.
func (crt *Certificate) createOrUpdateEvent(event fsnotify.Event) error {
	if event.Op&fsnotify.Remove == fsnotify.Remove {
		if err := crt.watcher.Add(event.Name); err != nil {
			return err
		}
	}

	if err := crt.loadCert(); err != nil {
		return err
	}

	return nil
}
