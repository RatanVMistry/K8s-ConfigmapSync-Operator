/*
Copyright 2025.

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

package controller

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "github.com/RatanVMistry/K8s-ConfigmapSync-Operator/api/v1"
	// "github.com/onsi/ginkgo/v2/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigSyncReconciler reconciles a ConfigSync object
type ConfigSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.my.domain,resources=configsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.my.domain,resources=configsyncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.my.domain,resources=configsyncs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConfigSync object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *ConfigSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	ctx = context.Background()
	log := log.FromContext(ctx)

	// Fetch the ConfigSync instance
	configSync := &appsv1.ConfigSync{}
	if err := r.Get(ctx, req.NamespacedName, configSync); err != nil {
		log.Error(err, "unable to fetch ConfigSync")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//Fetch the source configmap
	sourceConfigMap := &corev1.ConfigMap{}
	sourceConfigMapName := types.NamespacedName{
		Namespace: configSync.Spec.SourceNamespace,
		Name:      configSync.Spec.ConfigMapName,
	}
	if err := r.Get(ctx, sourceConfigMapName, sourceConfigMap); err != nil {
		return ctrl.Result{}, err
	}

	// Create or Update the destination configmap in the destination namespace
	destinationConfigMap := &corev1.ConfigMap{}
	destinationConfigMapName := types.NamespacedName{
		Namespace: configSync.Spec.DestinationNamespace,
		Name:      configSync.Spec.ConfigMapName,
	}
	if err := r.Get(ctx, destinationConfigMapName, destinationConfigMap); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Creating ConfigMap in destination namespace", "namespace", configSync.Spec.DestinationNamespace, "name", configSync.Spec.ConfigMapName)
			destinationConfigMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: configSync.Spec.DestinationNamespace,
					Name:      configSync.Spec.ConfigMapName,
				},
				Data:       sourceConfigMap.Data,
			}
			destinationConfigMap.Namespace = configSync.Spec.DestinationNamespace
			destinationConfigMap.Name = configSync.Spec.ConfigMapName
			if err := r.Create(ctx, destinationConfigMap); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		log.Info("Updating ConfigMap in destination namespace", "namespace", configSync.Spec.DestinationNamespace, "name", configSync.Spec.ConfigMapName)
		destinationConfigMap.Data = sourceConfigMap.Data
			if err := r.Update(ctx, destinationConfigMap); err != nil {
				return ctrl.Result{}, err
			}
		}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.ConfigSync{}).
		Named("configsync").
		Complete(r)
}
