package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"crypto/tls"

	"demo/pkg/webhook"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/golang/glog"
)

func main() {
	var parameters webhook.WhSvrParameters
	flag.IntVar(&parameters.Port, "port", 443, "Webhook server port.")
	flag.StringVar(&parameters.CertFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.KeyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	pair, err := tls.LoadX509KeyPair(parameters.CertFile, parameters.KeyFile)
	if err != nil {
		glog.Errorf("Filed to load key pair: %v", err)
	}

	whsvr := &webhook.WebhookServer{
		Clientset: clientset,
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", parameters.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.Serve)
	whsvr.Server.Handler = mux

	// start webhook server in new rountine
	go func() {
		if err := whsvr.Server.ListenAndServeTLS("", ""); err != nil {
			glog.Errorf("Filed to listen and serve webhook server: %v", err)
		}
	}()

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	glog.Infof("Got OS shutdown signal, shutting down wenhook server gracefully...")
	whsvr.Server.Shutdown(context.Background())
}
