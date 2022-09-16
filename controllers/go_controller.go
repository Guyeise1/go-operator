/*
Copyright 2022.

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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	shmilav1 "github.com/Guyeise1/go-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GoReconciler reconciles a Go object
type GoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=shmila.iaf,resources=goes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=shmila.iaf,resources=goes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=shmila.iaf,resources=goes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Go object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *GoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	fmt.Println("Shmila go operator is reconciling and kicking")
	cr := &shmilav1.Go{}
	r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, cr)
	fmt.Println("passed get cr")
	fmt.Printf("got the cr: %s", cr.Spec)
	fmt.Println("")
	secret := &corev1.Secret{}
	secretName := "go-" + req.Namespace + "-" + cr.Spec.Alias
	operatorNs := "go-operator-system"
	err := r.Get(ctx, client.ObjectKey{Name: secretName, Namespace: operatorNs}, secret)

	if err != nil && errors.IsNotFound(err) {
		// TODO: check error type
		data := make(map[string]string)
		data["password"] = "hello"

		secret = &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: operatorNs,
			},
			StringData: data,
		}
		r.Create(ctx, secret)
	}

	fmt.Println("success fetch secret")
	fmt.Printf("go secret: %s", secret)
	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&shmilav1.Go{}).
		Complete(r)
}
