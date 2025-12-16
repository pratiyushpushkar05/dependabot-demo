package main

import (
	"crypto/sha256"
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	hook "github.com/uipath/service-fabric-utils/admctl-webhook/webhook"
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

var log = logf.Log.WithName("admctl-webhook-controller")

type HookParamters struct {
	certDir string
	config  string
	port    int
}

func loadConfig(configFile string) (*hook.Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read admctl-webhook-config file")
	}

	glog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

	var cfg hook.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(err, "unable to parse admctl-webhook-config file")
	}

	return &cfg, nil
}

func main() {
	var params HookParamters

	flag.IntVar(&params.port, "port", 8443, "Webhook port")
	flag.StringVar(&params.certDir, "certDir", "/etc/webhook/config/certs/", "Wehbook certificate folder")
	flag.StringVar(&params.config, "config", "/etc/webhook/config/admctl-webhook-config.yaml", "Admission control webhook config")
	flag.Parse()

	logf.SetLogger(zap.New())

	entryLog := log.WithName("entrypoint")

	entryLog.Info("setting up manager")

	// Setup a new manager to provide shared dependencies and start components
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

	cfg, err := loadConfig(params.config)
	if err != nil {
		entryLog.Error(err, "unable to parse admctl-webhook-config file")
		os.Exit(1)
	}

	// Setup webhooks
	entryLog.Info("setting up webhook server")

	hookServer := mgr.GetWebhookServer()

	// Create the admission decoder and assign it to the handler
	decoder := admission.NewDecoder(mgr.GetScheme())

	ti := &hook.TolerationInjector{
		Name:    "webhook",
		Client:  mgr.GetClient(),
		Config:  cfg,
		Decoder: decoder,
	}

	entryLog.Info("registering webhooks to the webhook server")
	hookServer.Register("/mutate", &webhook.Admission{Handler: ti})

	entryLog.Info("starting manager")

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}
