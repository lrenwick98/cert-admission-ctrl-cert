package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// "github.com/golang/glog"
	"context"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/openshift/api/config/v1"
	ocp_clientset "github.com/openshift/client-go/config/clientset/versioned"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var ocpClient ocp_clientset.Interface
var config *rest.Config
var err error
var UrlSuffix *v1.DNS

var issuer string
var issuerKind string
var issuerGroup string

type myServerHandler struct {
}

// GetConfig function to get Kubernetes Config
func GetConfig() (*rest.Config, error) {
	// If an env variable is specified with the config locaiton, use that
	if len(os.Getenv("KUBECONFIG")) > 0 {
		return clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	}
	// If no explicit location, try the in-cluster config
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	// If no in-cluster config, try the default location in the user's home directory
	if usr, err := user.Current(); err == nil {
		if c, err := clientcmd.BuildConfigFromFlags(
			"", filepath.Join(usr.HomeDir, ".kube", "config")); err == nil {
			return c, nil
		}
	}
	return nil, err
}

func Log() zap.SugaredLogger {

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)

	logger, err := loggerConfig.Build()
	if err != nil {
		log.Fatal(err)
	}

	sugar := logger.Sugar()
	return *sugar
}

func main() {

	timestampLog := Log()
	timestampLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	timestampLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))

	issuer = os.Getenv("issuer")
	issuerKind = os.Getenv("issuer-kind")
	issuerGroup = os.Getenv("issuer-group")

	if issuer == "" || issuerKind == "" || issuerGroup == "" {
		timestampLog.Errorf("All environment variables are not set")
		return
	}

	ctx := context.TODO()

	config, err := GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	ocpclientset, err := ocp_clientset.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	ocpClient := ocpclientset.ConfigV1()

	UrlSuffix, _ = ocpClient.DNSes().Get(ctx, "cluster", meta_v1.GetOptions{})

	// check the Environment variable for User Define Placements
	certpem := "/opt/app-root/tls/tls.crt"
	keypem := "/opt/app-root/tls/tls.key"
	certs, err := tls.LoadX509KeyPair(certpem, keypem)
	if err != nil {
		//  glog.Errorf("Failed to load Certificate/Key Pair: %v", err)
		timestampLog.Errorf("Failed to load Certificate/Key Pair: %v", err)
	}

	// Setting the HTTP Server with TLS (HTTPS)
	server := &http.Server{
		Addr:      fmt.Sprintf(":%v", "8443"),
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{certs}},
	}
	// Setting 2 variable which are defined by an empty struct for each of the function depending on the URL path
	// the http request is calling
	// in our example we have 2 paths , one for the mutate and one for validate
	va := myServerHandler{}
	mux := http.NewServeMux()
	// Setting a function reference for the /validate URL
	mux.HandleFunc("/validate", va.valserve)
	server.Handler = mux
	// Starting a new channel to start the Server with TLS configuration we provided when we defined the server variable
	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			//  Failed to Listen and Serve Web Hook Server
			timestampLog.Errorf("Failed to Listen and Serve Web Hook Server: %v", err)
		}
	}()
	// The Server Is running on Port : 8080 by default
	timestampLog.Info("The Server Is running on Port : 8443")
	// Next we are going to setup the single handling for our HTTP server by sending the right signals to the channel
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	// Get Shutdown signal , shutting down the webhook Server gracefully...
	timestampLog.Info("Get Shutdown signal , shutting down the webhook Server gracefully...")
	server.Shutdown(context.Background())
}
