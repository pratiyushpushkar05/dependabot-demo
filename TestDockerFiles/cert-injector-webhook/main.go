package main

import (
	"crypto/sha256"
	"flag"
	"os"

	"github.com/golang/glog"
	hook "github.com/uipath/service-fabric-utils/cert-injector-webhook/webhook"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/ghodss/yaml"
)

var log = logf.Log.WithName("cert-initializer-injector-controller")

type HookParamters struct {
	certDir    string
	config     string
	port       int
	certInject bool
}

func loadConfig(configFile string) (*hook.ConfigTypes, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	var cfg hook.ConfigTypes
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func main() {
	var params HookParamters

	flag.IntVar(&params.port, "port", 8443, "Wehbook port")
	flag.BoolVar(&params.certInject, "certInject", false, "Inject cert")
	flag.StringVar(&params.certDir, "certDir", "/etc/webhook/config/certs/", "Wehbook certificate folder")
	flag.StringVar(&params.config, "config", "/etc/webhook/config/cert-initializer-injector-config.yaml", "Wehbook cert injector config")
	flag.Parse()

	logf.SetLogger(zap.New())

	entryLog := log.WithName("entrypoint")

	// Setup a Manager
	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    params.port,
			CertDir: params.certDir,
		}),
	})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	config, err := loadConfig(params.config)
	if err != nil {
		entryLog.Error(err, "unable to parse cert-injector config file")
		os.Exit(1)
	}

	// Setup webhooks
	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	// Create the admission decoder and assign it to the handler
	decoder := admission.NewDecoder(mgr.GetScheme())

	ci := &hook.CertInitializer{
		Name:         "webhook",
		Client:       mgr.GetClient(),
		Config:       config,
		CertInjector: params.certInject,
		Decoder:      decoder,
	}

	entryLog.Info("registering webhooks to the webhook server")
	hookServer.Register("/mutate", &webhook.Admission{Handler: ci})

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}
