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

package controllers

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"
	maupuv1beta1 "github.com/nmaupu/vault-secret/api/v1beta1"
	nmvault "github.com/nmaupu/vault-secret/pkg/vault"
	appVersion "github.com/nmaupu/vault-secret/version"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var _ reconcile.Reconciler = (*VaultSecretReconciler)(nil)

const (
	// OperatorAppName is the name of the operator
	OperatorAppName = "vaultsecret-operator"
	// TimeFormat is the time format to indicate last updated field
	TimeFormat = "2006-01-02_15-04-05"
	// MinTimeMsBetweenSecretUpdate avoid a secret to be updated too often
	MinTimeMsBetweenSecretUpdate = time.Millisecond * 500
)

var (
	log = logf.Log.WithName(OperatorAppName)

	// secretsLastUpdateTime store last updated time of a secret to avoid reconciling too often
	// the same secret if it changes very fast (like with database KV backend or OTP)
	secretsLastUpdateTime      = make(map[string]time.Time)
	secretsLastUpdateTimeMutex sync.Mutex

	// LabelsFilter filters events on labels
	LabelsFilter map[string]string
)

// VaultSecretReconciler reconciles a VaultSecret object
type VaultSecretReconciler struct {
	client.Client
	Log          logr.Logger
	Scheme       *runtime.Scheme
	LabelsFilter map[string]string
}

// AddLabelFilter adds a label for filtering events
func AddLabelFilter(key, value string) {
	if LabelsFilter == nil {
		LabelsFilter = make(map[string]string)
	}

	LabelsFilter[key] = value
}

// +kubebuilder:rbac:groups=maupu.org,resources=vaultsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=maupu.org,resources=vaultsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch

// Reconcile reads that state of the cluster for a VaultSecret object and makes changes based on the state read
// and what is in the VaultSecret.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *VaultSecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling VaultSecret")
	ctx := context.Background()

	// Fetch the VaultSecret CRInstance
	CRInstance := &maupuv1beta1.VaultSecret{}
	err := r.Get(ctx, req.NamespacedName, CRInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("VaultSecret resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		log.Info(fmt.Sprintf("Error reading the VaultSecret object, requeuing, err=%v", err))
		return ctrl.Result{}, err
	}

	// Only updating stuff if two updates are not too close from each other
	// See secretsLastUpdateTime and MinTimeMsBetweenSecretUpdate variables
	updateTimeKey := fmt.Sprintf("%s/%s", CRInstance.GetNamespace(), CRInstance.Spec.SecretName)
	secretsLastUpdateTimeMutex.Lock()
	defer secretsLastUpdateTimeMutex.Unlock()
	ti := secretsLastUpdateTime[updateTimeKey] // no problem if it does not exist: it returns a default time.Time object (set to zero)
	now := time.Now()
	if now.Sub(ti) > MinTimeMsBetweenSecretUpdate {
		operatorName := os.Getenv("OPERATOR_NAME")
		if operatorName == "" {
			operatorName = OperatorAppName
		}

		labels := map[string]string{
			"app.kubernetes.io/name":       OperatorAppName,
			"app.kubernetes.io/version":    appVersion.Version,
			"app.kubernetes.io/managed-by": operatorName,
			"crName":                       CRInstance.Name,
			"crNamespace":                  CRInstance.Namespace,
			"lastUpdate":                   time.Now().Format(TimeFormat),
		}

		// Adding filtered labels
		for key, val := range LabelsFilter {
			labels[key] = val
		}

		secretName := CRInstance.Spec.SecretName
		if secretName == "" {
			secretName = CRInstance.Name
		}

		secretType := CRInstance.Spec.SecretType
		if secretType == "" {
			secretType = "Opaque"
		}

		for key, val := range CRInstance.Spec.SecretLabels {
			labels[key] = val
		}

		var secretData map[string][]byte
		var statusEntries []maupuv1beta1.VaultSecretStatusEntry
		var operationResult controllerutil.OperationResult

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: req.Namespace},
		}

		err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			var err error
			operationResult, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, secret, func() error {
				// As type field is immutable we quickly update the resource before reading from vault.
				// We expect a genuine error from the api server.
				if secret.Type != secretType && secret.Type != "" {
					secret.Type = secretType
					return nil
				}

				// Only read secret data once
				if secretData == nil {
					secretData, statusEntries, err = r.readSecretData(CRInstance)
					if err != nil {
						return err
					}
				}

				// Set labels
				if secret.Labels == nil {
					secret.Labels = make(map[string]string)
				}
				for k, v := range labels {
					secret.Labels[k] = v
				}

				// Set data
				if secret.Data == nil {
					secret.Data = make(map[string][]byte)
				}
				for key, data := range secretData {
					secret.Data[key] = data
				}
				secret.Type = secretType
				secret.Annotations = CRInstance.Spec.SecretAnnotations

				if err = controllerutil.SetControllerReference(CRInstance, secret, r.Scheme); err != nil {
					return err
				}

				// Here no error occurred, check if some field failed to update (status will be updated later on)
				for i := range statusEntries {
					if !statusEntries[i].Status {
						return fmt.Errorf("Some error occurred reading from vault")
					}
				}

				return nil
			})
			return err
		})

		// Update the VaultSecret Status only if it changed
		var statusEntriesErr error
		if statusEntries != nil && !equality.Semantic.DeepEqual(CRInstance.Status.Entries, statusEntries) {
			CRInstance.Status.Entries = statusEntries
			if statusEntriesErr = r.Client.Status().Update(context.TODO(), CRInstance); err != nil {
				reqLogger.Error(err, "Failed to update VaultSecret status")
				return reconcile.Result{}, statusEntriesErr
			}
		}

		if err != nil || statusEntriesErr != nil {
			// If the resource is invalid then next reconcile is unlikely to succeed so we don't requeue
			if errors.IsInvalid(err) {
				reqLogger.Error(err, "Failed to update VaultSecret")
				return reconcile.Result{}, nil
			}

			if err == nil {
				err = statusEntriesErr
			}
			return reconcile.Result{}, err
		}

		switch operationResult {
		case controllerutil.OperationResultCreated:
			reqLogger.Info("Secret created", "Secret.Name", secretName)
		case controllerutil.OperationResultUpdated:
			reqLogger.Info("Secret updated", "Secret.Name", secretName)
		}

		// Check if some errors occurred while reading vault and log it
		for i := range statusEntries {
			if !statusEntries[i].Status {
				reqLogger.Info("Some errors occurred while reading secrets, see VaultSecret status for details")
				break
			}
		}
	}

	return reconcile.Result{RequeueAfter: CRInstance.Spec.SyncPeriod.Duration}, err
}

func (r *VaultSecretReconciler) readSecretData(cr *maupuv1beta1.VaultSecret) (map[string][]byte, []maupuv1beta1.VaultSecretStatusEntry, error) {
	reqLogger := log.WithValues("func", "readSecretData")

	// Authentication provider
	authProvider, err := cr.GetVaultAuthProvider(r.Client)
	if err != nil {
		return nil, nil, err
	}

	// Processing vault login
	vaultConfig := nmvault.NewConfig(cr.Spec.Config.Addr)
	vaultConfig.Namespace = cr.Spec.Config.Namespace
	vaultConfig.Insecure = cr.Spec.Config.Insecure
	vClient, err := authProvider.Login(vaultConfig)
	if err != nil {
		return nil, nil, err
	}

	vaultClient := nmvault.NewCachedClient(vClient)

	// Init
	secrets := map[string][]byte{}

	// Sort by secret keys to avoid updating the resource if order changes
	specSecrets := append(make([]maupuv1beta1.VaultSecretSpecSecret, 0, len(cr.Spec.Secrets)), cr.Spec.Secrets...)
	sort.Sort(maupuv1beta1.BySecretKey(specSecrets))

	statusEntries := make([]maupuv1beta1.VaultSecretStatusEntry, 0, len(cr.Spec.Secrets))

	// Creating secret data from CR
	for _, s := range specSecrets {
		var err error
		errMessage := ""
		rootErrMessage := ""
		var status bool

		// Vault read
		reqLogger.Info("Reading vault", "KvPath", s.KvPath, "Path", s.Path, "KvVersion", s.KvVersion)
		secret, err := vaultClient.Read(s.KvVersion, s.KvPath, s.Path)

		if err != nil {
			rootErrMessage = err.Error()
			errMessage = "Problem occurred while reading secret"
			status = false
		} else if secret == nil || secret[s.Field] == nil || secret[s.Field] == "" {
			errMessage = "Field does not exist"
			status = false
		} else {
			status = true
			secrets[s.SecretKey] = ([]byte)(secret[s.Field].(string))
		}

		// Updating CR Status field
		statusEntries = append(statusEntries, maupuv1beta1.VaultSecretStatusEntry{
			Secret:    s,
			Status:    status,
			Message:   errMessage,
			RootError: rootErrMessage,
		})
	}

	// Error is returned along with secret if it occurred at least once during loop
	// In case of error, we only return secrets that we could read. The caller has to handle itself.
	return secrets, statusEntries, nil
}
