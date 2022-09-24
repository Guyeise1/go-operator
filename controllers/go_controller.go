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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	shmilav1 "github.com/Guyeise1/go-operator/api/v1"
	"github.com/Guyeise1/go-operator/libs/environment"
)

// GoReconciler reconciles a Go object
type GoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	cleanApiFinalizer = "clean.api.finalizer"
)

var goHostUrl = environment.GetVariables().GoApiURL

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
	cr := shmilav1.Go{}
	retry := reconcile.Result{RequeueAfter: 120 * time.Second}
	complete := reconcile.Result{}
	err := r.Get(ctx, req.NamespacedName, &cr)
	if errors.IsNotFound(err) {
		return complete, nil
	} else if err != nil {
		return retry, err
	}

	if !cr.DeletionTimestamp.IsZero() {
		err = r.CleanupGo(ctx, &cr, req)
		if err != nil {
			return retry, err
		} else {
			return complete, nil
		}
	} else {
		if !controllerutil.ContainsFinalizer(&cr, cleanApiFinalizer) {
			controllerutil.AddFinalizer(&cr, cleanApiFinalizer)
			// On complete, this will retrigger the loop, therefore must return.
			err = r.Update(ctx, &cr)
			if err != nil {
				return retry, err
			} else {
				return complete, nil
			}
		} else {
			err = PersistInGoAPI(ctx, &cr)
			if err != nil {
				return retry, err
			} else {
				return complete, nil
			}
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *GoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// go cleanupLoop(mgr, time.Duration(environment.GetVariables().CleanIntervalSeconds)*time.Second)
	return ctrl.NewControllerManagedBy(mgr).
		For(&shmilav1.Go{}).
		Complete(r)
}

func (r *GoReconciler) CleanupGo(ctx context.Context, cr *shmilav1.Go, req ctrl.Request) error {
	m := map[string]string{
		"alias":    cr.Spec.Alias,
		"password": environment.GetVariables().Password,
	}
	json, _ := json.Marshal(m)
	resp, err := http.Post(goHostUrl+"/api/v1/go-links/delete", "application/json", bytes.NewBuffer(json))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 && resp.StatusCode != 404 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to delete link bad status code %d , failed to read body - error: %s", resp.StatusCode, err)
		}
		return fmt.Errorf("failed to delete link bad status code %d , body is %s", resp.StatusCode, string(body))
	}

	controllerutil.RemoveFinalizer(cr, cleanApiFinalizer)
	err = r.Update(ctx, cr)

	if err != nil {
		return err
	}

	return nil
}

func PersistInGoAPI(ctx context.Context, cr *shmilav1.Go) error {
	m := map[string]string{
		"alias":        cr.Spec.Alias,
		"url":          cr.Spec.Url,
		"password":     environment.GetVariables().Password,
		"passwordHint": "managed by go-operator",
	}
	json, _ := json.Marshal(m)
	resp, err := http.Post(goHostUrl+"/api/v1/go-links", "application/json", bytes.NewBuffer(json))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to persist link bad status code %d , failed to read body - error: %s", resp.StatusCode, err)
		}

		return fmt.Errorf("failed to persist link, status code is %d, body is %s", resp.StatusCode, string(body))
	}

	return nil
}
