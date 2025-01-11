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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
		if apierrors.IsNotFound(err) {
			log.Info("ConfigSync resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch ConfigSync")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// sync the configmaps
	if len(configSync.Spec.ConfigMapNames) > 0 {
		for _, configMapName := range configSync.Spec.ConfigMapNames {
			if err := r.syncConfigMap(ctx, configMapName, configSync.Spec.SourceNamespace, configSync.Spec.DestinationNamespaces); err != nil {
				log.Error(err, "Failed to sync ConfigMap", "configMapName", configMapName)
				return ctrl.Result{}, err
			}
		}
	}

	// sync the secrets
	if len(configSync.Spec.SecretNames) > 0 {
		for _, secretName := range configSync.Spec.SecretNames {
			if err := r.syncSecret(ctx, secretName, configSync.Spec.SourceNamespace, configSync.Spec.DestinationNamespaces); err != nil {
				log.Error(err, "Failed to sync Secret", "secretName", secretName)
				return ctrl.Result{}, err
			}
		}
	}
	log.Info("ConfigSync reconciled")
	return ctrl.Result{}, nil
}

func (r *ConfigSyncReconciler) syncConfigMap(ctx context.Context, name string, sourceNamespace string, destinationNamespaces []string) error {
	// Fetch the source configmap
	log := log.FromContext(ctx)

	sourceConfigMap := &corev1.ConfigMap{}
	sourceConfigMapName := types.NamespacedName{
		Namespace: sourceNamespace,
		Name:      name,
	}
	if err := r.Get(ctx, sourceConfigMapName, sourceConfigMap); err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "Source ConfigMap not found", "namespace", sourceNamespace, "name", name)
			return nil
		}

		return err
	}

	// Create or Update the destination configmap in the destination namespace
	for _, destinationNamespace := range destinationNamespaces {
		destinationConfigMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: destinationNamespace,
				Name:      name,
			},
		}
		_, err := ctrl.CreateOrUpdate(ctx, r.Client, destinationConfigMap, func() error {
			destinationConfigMap.Data = sourceConfigMap.Data
			destinationConfigMap.Labels = sourceConfigMap.Labels
			return nil
		})
		if err != nil {
			log.Error(err, "Failed to sync ConfigMap", "namespace", destinationNamespace, "name", name)
			return err
		}
		log.Info("ConfigMap synced", "namespace", destinationNamespace, "name", name)
	}

	return nil
}

func (r *ConfigSyncReconciler) syncSecret(ctx context.Context, name string, sourceNamespace string, destinationNamespaces []string) error {
	// Fetch the source secret
	log := log.FromContext(ctx)

	sourceSecret := &corev1.Secret{}
	sourceSecretName := types.NamespacedName{
		Namespace: sourceNamespace,
		Name:      name,
	}
	if err := r.Get(ctx, sourceSecretName, sourceSecret); err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "Source Secret not found", "namespace", sourceNamespace, "name", name)
			return nil
		}

		return err
	}

	// Create or Update the destination secret in the destination namespace
	for _, destinationNamespace := range destinationNamespaces {
		destinationSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: destinationNamespace,
				Name:      name,
			},
		}
		_, err := ctrl.CreateOrUpdate(ctx, r.Client, destinationSecret, func() error {
			destinationSecret.Data = sourceSecret.Data
			destinationSecret.Type = sourceSecret.Type
			destinationSecret.Labels = sourceSecret.Labels
			return nil
		})
		if err != nil {
			log.Error(err, "Failed to sync Secret", "namespace", destinationNamespace, "name", name)
			return err
		}
		log.Info("Secret synced", "namespace", destinationNamespace, "name", name)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.ConfigSync{}).
		Named("configsync").
		Complete(r)
}
