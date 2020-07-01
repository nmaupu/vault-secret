package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	"github.com/nmaupu/vault-secret/pkg/apis"
	"github.com/nmaupu/vault-secret/pkg/controller"
	"github.com/nmaupu/vault-secret/pkg/controller/vaultsecret"

	appVersion "github.com/nmaupu/vault-secret/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	WatchMultiNamespacesEnvVar = "WATCH_MULTINAMESPACES"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)
var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Vault-secret operator version: %v", appVersion.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Filter events on a labels
	var labels *[]string = pflag.StringArray("filter-label", []string{}, "Process only Vaultsecret custom resources containing the given label")

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	// Labels filtering
	for _, v := range *labels {
		toks := strings.Split(v, "=")
		if len(toks) != 2 {
			log.Error(errors.New("Incorrect label filter"), v)
		} else {
			key := toks[0]
			val := toks[1]
			log.Info(fmt.Sprintf("Adding label filter: %s = %s", key, val))
			vaultsecret.AddLabelFilter(key, val)
		}
	}

	// Get namespace to watch from WATCH_NAMESPACE environment variable
	// If set, use it. Otherwise, try WATCH_MULTINAMESPACES environment variable
	// If not set, use cluster wide configuration
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		// WATCH_NAMESPACE not found
		log.Info("WATCH_NAMESPACE env var not set")
	}
	multiNamespaces, err := GetWatchMultiNamespaces()
	if err != nil {
		// WATCH_MULTINAMESPACES not found
		log.Info(fmt.Sprintf("%s not set", WatchMultiNamespacesEnvVar))
	}
	if namespace == "" && len(multiNamespaces) == 0 {
		log.Info(fmt.Sprintf("WATCH_NAMESPACE and %s are not set, operator is cluster wide", WatchMultiNamespacesEnvVar))
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()
	// Become the leader before proceeding
	err = leader.Become(ctx, "vault-secret-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	// Create manager options struct depending on namespace(s) possibilities
	mgrOptions := manager.Options{
		Namespace:          namespace,
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	}
	// WATCH_MULTINAMESPACES is taking over WATCH_NAMESPACE
	if len(multiNamespaces) > 0 {
		log.Info(fmt.Sprintf("Using WATCH_MULTINAMESPACES value = %+v", multiNamespaces))
		mgrOptions.NewCache = cache.MultiNamespacedCacheBuilder(multiNamespaces)
		mgrOptions.Namespace = ""
	} else {
		log.Info(fmt.Sprintf("Using WATCH_NAMESPACE value = \"%s\"", namespace))
	}
	// Creating the manager
	mgr, err := manager.New(cfg, mgrOptions)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err = serveCRMetrics(cfg); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}

// GetWatchMultiNamespaces returns the namespaces list the operator should be watching for changes
// Very similar to WATCH_NAMESPACE but for multiple namespaces
func GetWatchMultiNamespaces() ([]string, error) {
	var namespaces []string
	ns, found := os.LookupEnv(WatchMultiNamespacesEnvVar)
	if !found {
		return namespaces, fmt.Errorf("%s env var is not set", WatchMultiNamespacesEnvVar)
	}

	namespaces = strings.Split(ns, ",")
	return namespaces, nil
}
