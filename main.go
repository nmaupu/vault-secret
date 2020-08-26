/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"

	vaultsecret "github.com/nmaupu/vault-secret/controllers"
	appVersion "github.com/nmaupu/vault-secret/pkg/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	maupuv1beta1 "github.com/nmaupu/vault-secret/api/v1beta1"
	// +kubebuilder:scaffold:imports
)

const (
	// WatchMultiNamespacesEnvVar godoc
	WatchMultiNamespacesEnvVar = "WATCH_MULTINAMESPACES"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var log = logf.Log.WithName("cmd")

// stringArrayFlag is a way to provide a flag multiple times as command line argument
type stringArrayFlag []string

func (i *stringArrayFlag) String() string {
	return "stringArrayFlag"
}
func (i *stringArrayFlag) Set(v string) error {
	*i = append(*i, v)
	return nil
}

func printVersion() {
	log.Info(fmt.Sprintf("Vault-secret operator version: %v", appVersion.Version))
	log.Info(fmt.Sprintf("Go Version: %s", goruntime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", goruntime.GOOS, goruntime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(maupuv1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var labels stringArrayFlag

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Var(&labels, "filter-label", "Process only VaultSecret custom resources containing the given labels")

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// Log current version information
	printVersion()

	// Labels filtering
	var labelsFilter = make(map[string]string)
	for _, v := range labels {
		toks := strings.Split(v, "=")
		if len(toks) != 2 {
			log.Error(errors.New("Incorrect label filter"), v)
		} else {
			key := toks[0]
			val := toks[1]
			log.Info(fmt.Sprintf("Adding label filter: %s = %s", key, val))
			labelsFilter[key] = val
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

	mgrOptions := manager.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "cd2da758.maupu.org",
		Namespace:          namespace,
	}

	if len(multiNamespaces) > 0 {
		log.Info(fmt.Sprintf("Using WATCH_MULTINAMESPACES value = %+v", multiNamespaces))
		mgrOptions.NewCache = cache.MultiNamespacedCacheBuilder(multiNamespaces)
		mgrOptions.Namespace = ""
	} else {
		log.Info(fmt.Sprintf("Using WATCH_NAMESPACE value = \"%s\"", namespace))
	}

	// Creating the manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&vaultsecret.VaultSecretReconciler{
		Client:       mgr.GetClient(),
		Log:          ctrl.Log.WithName("controllers").WithName("VaultSecret"),
		Scheme:       mgr.GetScheme(),
		LabelsFilter: labelsFilter,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VaultSecret")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
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
