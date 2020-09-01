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
	maupuv1beta1 "github.com/nmaupu/vault-secret/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// SetupWithManager godoc
func (r *VaultSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&maupuv1beta1.VaultSecret{}).
		Owns(&corev1.Secret{}).
		WithEventFilter(r.filterLabelsPredicate()).
		Complete(r)
}

func (r *VaultSecretReconciler) filterLabelsPredicate() predicate.Predicate {
	predFunc := func(e interface{}) bool {
		log := r.Log.WithValues("func", "predFunc")
		var objectLabels map[string]string

		// Trying to determine what sort of event we have
		// https://tour.golang.org/methods/16
		switch e.(type) {
		case event.CreateEvent:
			log.Info("Create event")
			objectLabels = e.(event.CreateEvent).Meta.GetLabels()
		case event.UpdateEvent:
			log.Info("Update event")
			objectLabels = e.(event.UpdateEvent).MetaNew.GetLabels()
		case event.DeleteEvent:
			log.Info("Delete event")
			objectLabels = e.(event.DeleteEvent).Meta.GetLabels()
		case event.GenericEvent:
			log.Info("Generic event")
			objectLabels = e.(event.GenericEvent).Meta.GetLabels()
		default: // should never happen except if a new Event type is created
			return false
		}

		// If labels match, we process the event, otherwise, simply ignore it
		// Verifying that each labels configured are present in the target object
		for lfk, lfv := range r.LabelsFilter {
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

	return predicate.Funcs{
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
}
