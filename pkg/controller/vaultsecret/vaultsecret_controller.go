package vaultsecret

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	maupuv1beta1 "github.com/nmaupu/vault-secret/pkg/apis/maupu/v1beta1"
	nmvault "github.com/nmaupu/vault-secret/pkg/vault"
	appVersion "github.com/nmaupu/vault-secret/version"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

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
)

// LabelsFilter filters events on labels
var LabelsFilter map[string]string

// AddLabelFilter adds a label for filtering events
func AddLabelFilter(key, value string) {
	if LabelsFilter == nil {
		LabelsFilter = make(map[string]string)
	}

	LabelsFilter[key] = value
}

// Add creates a new VaultSecret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVaultSecret{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(OperatorAppName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Generic function for create, update and delete events
	// which verifies labels' filtering
	predFunc := func(e interface{}) bool {
		reqLogger := log.WithValues("func", "predFunc")
		var objectLabels map[string]string

		// Trying to determine what sort of event we have
		// https://tour.golang.org/methods/16
		switch e.(type) {
		case event.CreateEvent:
			reqLogger.Info("Create event")
			objectLabels = e.(event.CreateEvent).Meta.GetLabels()
		case event.UpdateEvent:
			reqLogger.Info("Update event")
			objectLabels = e.(event.UpdateEvent).MetaNew.GetLabels()
		case event.DeleteEvent:
			reqLogger.Info("Delete event")
			objectLabels = e.(event.DeleteEvent).Meta.GetLabels()
		case event.GenericEvent:
			reqLogger.Info("Generic event")
			objectLabels = e.(event.GenericEvent).Meta.GetLabels()
		default: // should never happen except if a new Event type is created
			return false
		}

		// If labels match, we process the event, otherwise, simply ignore it
		// Verifying that each labels configured are present in the target object
		for lfk, lfv := range LabelsFilter {
			if val, ok := objectLabels[lfk]; ok {
				if val != lfv {
					return false
				}
			} else {
				return false
			}
		}

		return true
	}
	// Create a predicate to filter incoming events on configured labels filter
	pred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return predFunc(e)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return predFunc(e)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return predFunc(e)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return predFunc(e)
		},
	}
	// Watch for changes to primary resource VaultSecret
	err = c.Watch(&source.Kind{Type: &maupuv1beta1.VaultSecret{}}, &handler.EnqueueRequestForObject{}, pred)
	if err != nil {
		return err
	}

	// Also watch for operator's created secrets
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &maupuv1beta1.VaultSecret{},
	}, pred)

	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileVaultSecret{}

// ReconcileVaultSecret reconciles a VaultSecret object
type ReconcileVaultSecret struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VaultSecret object and makes changes based on the state read
// and what is in the VaultSecret.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVaultSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VaultSecret")

	// Fetch the VaultSecret CRInstance
	CRInstance := &maupuv1beta1.VaultSecret{}
	err := r.client.Get(context.TODO(), request.NamespacedName, CRInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("IsNotFound error when retrieving the VaultSecret object")
			return reconcile.Result{}, nil
		}

		reqLogger.Info(fmt.Sprintf("Error reading the vaultsecret object, requeuing, err=%v", err))
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Only updating stuff if two updates are not too close from each other
	// See secretsLastUpdateTime and MinTimeMsBetweenSecretUpdate variables
	updateTimeKey := fmt.Sprintf("%s/%s", CRInstance.GetNamespace(), CRInstance.Spec.SecretName)
	secretsLastUpdateTimeMutex.Lock()
	defer secretsLastUpdateTimeMutex.Unlock()
	ti := secretsLastUpdateTime[updateTimeKey] // no problem if it does not exist: it returns a default time.Time object (set to zero)
	now := time.Now()
	if now.Sub(ti) > MinTimeMsBetweenSecretUpdate {
		// Define a new Secret object from CR specs
		secretFromCR, err := newSecretForCR(CRInstance)
		if err != nil && secretFromCR == nil {
			// An error occurred, requeue
			reqLogger.Error(err, "An error occurred when creating secret from CR, requeuing.")
			return reconcile.Result{}, err
		} else if err != nil && secretFromCR != nil {
			// Some vault path and/or fields are not found, update CR (status) and requeue
			if updateErr := r.client.Status().Update(context.TODO(), CRInstance); updateErr != nil {
				reqLogger.Error(updateErr, fmt.Sprintf("Some errors occurred but CR cannot be updated, requeuing, original error=%v", err))
			} else {
				reqLogger.Error(err, "Some errors have been issued in the CR status information, please check, requeuing")
			}
			return reconcile.Result{}, err
		}

		// Everything's ok

		// Set VaultSecret CRInstance as the owner and controller
		if err = controllerutil.SetControllerReference(CRInstance, secretFromCR, r.scheme); err != nil {
			reqLogger.Error(err, "An error occurred when setting controller reference, requeuing")
			return reconcile.Result{}, err
		}

		// Creating or updating secret resource from CR
		// Check if this Secret already exists
		found := &corev1.Secret{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: secretFromCR.Name, Namespace: secretFromCR.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			// Secret does not exist, creating it
			reqLogger.Info(fmt.Sprintf("Creating new Secret %s/%s", secretFromCR.Namespace, secretFromCR.Name))
			err = r.client.Create(context.TODO(), secretFromCR)
		} else {
			// Secret already exists - updating
			reqLogger.Info(fmt.Sprintf("Reconciling existing Secret %s/%s", found.Namespace, found.Name))
			err = r.client.Update(context.TODO(), secretFromCR)
		}

		// No problem creating or updating secret, updating CR info
		reqLogger.Info("Updating CR status information")
		if updateErr := r.client.Status().Update(context.TODO(), CRInstance); updateErr != nil {
			reqLogger.Error(updateErr, "Error occurred when updating CR status")
		}

		// Updating "update time" at the very end to take into account all potential requeue above
		secretsLastUpdateTime[updateTimeKey] = now
	}

	// finally returning given err (nil if no problem occurred, set to something otherwise)
	return reconcile.Result{RequeueAfter: CRInstance.Spec.SyncPeriod.Duration}, err
}

func newSecretForCR(cr *maupuv1beta1.VaultSecret) (*corev1.Secret, error) {
	reqLogger := log.WithValues("func", "newSecretForCR")
	operatorName := os.Getenv("OPERATOR_NAME")
	if operatorName == "" {
		operatorName = OperatorAppName
	}
	labels := map[string]string{
		"app.kubernetes.io/name":       OperatorAppName,
		"app.kubernetes.io/version":    appVersion.Version,
		"app.kubernetes.io/managed-by": operatorName,
		"crName":                       cr.Name,
		"crNamespace":                  cr.Namespace,
		"lastUpdate":                   time.Now().Format(TimeFormat),
	}

	// Adding filtered labels
	for key, val := range LabelsFilter {
		labels[key] = val
	}

	secretName := cr.Spec.SecretName
	if secretName == "" {
		secretName = cr.Name
	}

	secretType := cr.Spec.SecretType
	if secretType == "" {
		secretType = "Opaque"
	}

	for key, val := range cr.Spec.SecretLabels {
		labels[key] = val
	}

	// Authentication provider
	authProvider, err := cr.GetVaultAuthProvider()
	if err != nil {
		return nil, err
	}

	// Processing vault login
	vaultConfig := nmvault.NewVaultConfig(cr.Spec.Config.Addr)
	vaultConfig.Namespace = cr.Spec.Config.Namespace
	vaultConfig.Insecure = cr.Spec.Config.Insecure
	vclient, err := authProvider.Login(vaultConfig)
	if err != nil {
		return nil, err
	}

	// Init
	hasError := false
	secrets := map[string][]byte{}

	// Clear status slice
	cr.Status.Entries = nil

	// Creating secret data from CR
	// Each secret entry in the CR will need a vault read to get filled.
	// If the KV/path remain the same, it's useless to call the vault again
	// as all fields are returned in the original read.
	// As a result, we are storing temporarily vault's data and use it as a "cache" to avoid
	// overloading the vault server.
	// This will be GC'ed at the end of the func
	cache := make(map[string](map[string]interface{}), 0)

	// Sort by secret keys to avoid updating the resource if order changes
	specSecrets := cr.Spec.Secrets
	sort.Sort(maupuv1beta1.BySecretKey(specSecrets))

	// Creating secret data from CR
	for _, s := range specSecrets {
		var err error
		errMessage := ""
		rootErrMessage := ""
		status := true

		// Vault read
		var sec map[string]interface{}
		cacheKey := fmt.Sprintf("%s/%s", s.KvPath, s.Path)
		if cacheVal, ok := cache[cacheKey]; ok {
			sec = cacheVal
			err = nil
		} else {
			reqLogger.Info(fmt.Sprintf("Reading from vault with the following info, path=%s, kvVersion=%d", cacheKey, s.KvVersion))
			sec, err = nmvault.Read(vclient, s.KvPath, s.Path, s.KvVersion)
			if err != nil || sec != nil { // only cache value if there is no error or a sec returned
				cache[cacheKey] = sec
			}
		}

		if err != nil {
			hasError = true
			if err != nil {
				rootErrMessage = err.Error()
			}
			errMessage = "Problem occurred getting secret"
			status = false
		} else if sec == nil || sec[s.Field] == nil || sec[s.Field] == "" {
			hasError = true
			if err != nil {
				rootErrMessage = err.Error()
			}
			errMessage = "Secret field not found in vault"
			status = false
		} else {
			status = true
			secrets[s.SecretKey] = ([]byte)(sec[s.Field].(string))
		}

		// Updating CR Status field
		cr.Status.Entries = append(cr.Status.Entries, maupuv1beta1.VaultSecretStatusEntry{
			Secret:    s,
			Status:    status,
			Message:   errMessage,
			RootError: rootErrMessage,
		})
	}

	// Handle return
	// Error is returned along with secret if it occurred at least once during loop
	// In case of error, we return a half populated secret object that caller has to handle itself
	var retErr error
	retErr = nil
	if hasError {
		retErr = fmt.Errorf("Secret %s cannot be created, see CR Status field for details", cr.Spec.SecretName)
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Data: secrets,
		Type: secretType,
	}, retErr
}
