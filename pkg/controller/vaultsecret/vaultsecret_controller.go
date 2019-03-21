package vaultsecret

import (
	"context"
	goerrors "errors"
	"fmt"
	maupuv1beta1 "github.com/nmaupu/vault-secret/pkg/apis/maupu/v1beta1"
	nmvault "github.com/nmaupu/vault-secret/pkg/vault"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "vaultsecret-controller"
)

var log = logf.Log.WithName(ControllerName)

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
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VaultSecret
	err = c.Watch(&source.Kind{Type: &maupuv1beta1.VaultSecret{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Also watch for operator's created secrets
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &maupuv1beta1.VaultSecret{},
	})
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
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Secret object from CR specs
	secretFromCR, err := newSecretForCR(CRInstance)
	if err != nil && secretFromCR == nil {
		// An error occured, requeue
		return reconcile.Result{}, err
	} else if err != nil && secretFromCR != nil {
		// Some vault path and/or fields are not found, update CR (status) and requeue
		reqLogger.Info("Some errors have been issued in the CR status information, please check")
		if updateErr := r.client.Status().Update(context.TODO(), CRInstance); updateErr != nil {
			reqLogger.Info(fmt.Sprintf("Error occured when updating CR status: %v", updateErr))
		}
		return reconcile.Result{}, err
	}

	// Everything's ok

	// Set VaultSecret CRInstance as the owner and controller
	if err = controllerutil.SetControllerReference(CRInstance, secretFromCR, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Creating or updating secret resource from CR
	// Check if this Secret already exists
	found := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: secretFromCR.Name, Namespace: secretFromCR.Namespace}, found)
	reqLogger.Info(fmt.Sprintf("found=%v, err=%v", found, err))
	if err != nil && errors.IsNotFound(err) {
		// Secret does not exist, creating it
		reqLogger.Info(fmt.Sprintf("Creating new Secret %s/%s", secretFromCR.Namespace, secretFromCR.Name))
		err = r.client.Create(context.TODO(), secretFromCR)
	} else {
		// Secret already exists - updating
		reqLogger.Info(fmt.Sprintf("Reconcile: Secret %s/%s already exists, updating", found.Namespace, found.Name))
		err = r.client.Update(context.TODO(), secretFromCR)
	}

	// No problem creating or updating secret, updating CR info
	reqLogger.Info("Updating CR status information")
	if updateErr := r.client.Status().Update(context.TODO(), CRInstance); updateErr != nil {
		reqLogger.Info(fmt.Sprintf("Error occured when updating CR status: %v", updateErr))
	}

	// finally return giving err (nil if not problem occured, set to something otherwise)
	return reconcile.Result{}, err
}

func newSecretForCR(cr *maupuv1beta1.VaultSecret) (*corev1.Secret, error) {
	labels := map[string]string{
		"crName":      cr.Name,
		"crNamespace": cr.Namespace,
		"controller":  ControllerName,
	}

	secretName := cr.Spec.SecretName
	if secretName == "" {
		secretName = cr.Name
	}

	targetNamespace := cr.Spec.TargetNamespace
	if targetNamespace == "" {
		targetNamespace = cr.Namespace
	}

	// Authentication provider
	authProvider, err := cr.GetVaultAuthProvider()
	if err != nil {
		return nil, err
	}

	// Processing vault login
	vaultConfig := nmvault.NewVaultConfig(cr.Spec.Config.Addr)
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
	for _, s := range cr.Spec.Secrets {
		errMessage := ""
		rootErrMessage := ""
		status := true

		// Vault read
		sec, err := nmvault.Read(vclient, s.Path)

		if err != nil {
			hasError = true
			if err != nil {
				rootErrMessage = err.Error()
			}
			errMessage = "Problem occured getting secret"
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
	// Error is returned along with secret if it occured at least once during loop
	// In case of error, we return a half populated secret object that caller has to handle itself
	var retErr error
	retErr = nil
	if hasError {
		retErr = goerrors.New(fmt.Sprintf("Secret %s cannot be created, see CR Status field for details", cr.Spec.SecretName))
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: targetNamespace,
			Labels:    labels,
		},
		Data: secrets,
	}, retErr
}
