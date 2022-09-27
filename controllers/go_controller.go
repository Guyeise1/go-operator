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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	shmilav1 "github.com/Guyeise1/go-operator/api/v1"
	"github.com/Guyeise1/go-operator/internal/environment"
)

// GoReconciler reconciles a Go object
type GoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}
type secretData struct {
	Alias             string
	Password          string
	ResourceName      string
	ResourceNamespace string
}

var goHostUrl = environment.GetVariables().GoApiURL
var secretPrefix = environment.GetVariables().SecretPrefix
var complete = ctrl.Result{}
var retry = ctrl.Result{RequeueAfter: time.Duration(environment.GetVariables().RetryTimeSeconds) * time.Second}

var httpClient = http.Client{
	Timeout: time.Duration(environment.GetVariables().HttpRequestTimeoutSeconds) * time.Second,
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
	result := ctrl.Result{}

	cr := shmilav1.Go{}

	crErr := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &cr)

	setStatus(&cr, "go/"+cr.Spec.Alias+" -> "+cr.Spec.Url, Succees)

	defer r.Status().Update(ctx, &cr)

	fmt.Printf("[INFO - Reconcile] reconciling CR: %s-%s\n", cr.Namespace, cr.Name)

	operatorNs := environment.GetVariables().ControllerNamespace
	secret := getSecretObject(req.Name, req.Namespace, operatorNs)

	secErr := r.Get(ctx, client.ObjectKey{Namespace: secret.Namespace, Name: secret.Name}, &secret)

	if errors.IsNotFound(crErr) {
		fmt.Println("[INFO - reconcile] handle delete for " + req.Name)
		err := handleDelete(r.Client, ctx, &secret)
		return result, err
	}

	if errors.IsNotFound(secErr) {
		fmt.Println("[INFO - reconcile] secret " + secret.Name + " Not found, creating...")
		return r.handleCreate(ctx, &cr, &secret)
	} else if secErr != nil {
		fmt.Println("[ERROR - reconcile] error reading secret")
		fmt.Println(secErr)
		setStatus(&cr, "internal error - ERR_CODE=109", Failure)
		return retry, secErr
	} else {
		return r.handleUpdate(&cr, &secret)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *GoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	go cleanupLoop(mgr, time.Duration(environment.GetVariables().CleanIntervalSeconds)*time.Second)
	return ctrl.NewControllerManagedBy(mgr).
		For(&shmilav1.Go{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func randomPassword() string {
	letters := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	ret := make([]byte, 50)
	for i := 0; i < len(ret); i++ {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		ret[i] = letters[num.Int64()]
	}
	return string(ret)
}

func (r *GoReconciler) handleCreate(ctx context.Context, cr *shmilav1.Go, secret *corev1.Secret) (ctrl.Result, error) {
	fmt.Printf("[INFO - handleCreate] starting create process for CR: %s-%s\n", cr.Namespace, cr.Name)
	data := map[string]string{
		"alias":             cr.Spec.Alias,
		"password":          randomPassword(),
		"resourceName":      cr.Name,
		"resourceNamespace": cr.Namespace,
	}
	secret.StringData = data
	secret.ResourceVersion = ""
	err1 := r.Create(ctx, secret)
	if err1 != nil {
		fmt.Println("[ERROR - handleCreate] failed to create secret", secret.Name)
		fmt.Println(err1)
		setStatus(cr, "internal error - ERR_CODE=131", Failure)
		return retry, fmt.Errorf("internal error - ERR_CODE=131")
	}
	return r.handleUpdate(cr, secret)
}

func handleDelete(r client.Client, ctx context.Context, secret *corev1.Secret) error {
	fmt.Println("[INFO - handleDelete] starting delete process for (secret)", secret.Name)
	secretData, err := readSecret(secret)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("internal error - ERR_CODE=148")
	}

	body := map[string]string{"alias": secretData.Alias, "password": secretData.Password}
	json, _ := json.Marshal(body)

	res, err4 := httpClient.Post(goHostUrl+"/api/v1/go-links/delete", "application/json", bytes.NewBuffer(json))

	if err4 != nil {
		fmt.Printf("[ERROR - handleDelete] failed delete request (POST) %s, link: %s\n", goHostUrl+"/api/v1/go-links/delete", secretData.Alias)
		fmt.Println(err)
		return fmt.Errorf("internal error - ERR_CODE=159")
	} else if res.StatusCode/100 != 2 && res.StatusCode != 404 {
		fmt.Printf("[ERROR - handleDelete] failed delete request (POST) %s, link: %s, response status %d\n", goHostUrl+"/api/v1/go-links/delete", secretData.Alias, res.StatusCode)
		fmt.Println(err)
		return fmt.Errorf("internal error - ERR_CODE=163")
	}

	err = r.Delete(ctx, secret)
	if err != nil {
		fmt.Println("[ERROR - handleDelete] failed to delete secret", secret.Name)
		fmt.Println(err)
		return fmt.Errorf("internal error - ERR_CODE=175")
	}
	fmt.Println("[INFO - handleDelete] success deleting (secret)", secret.Name)
	return nil
}

func (r *GoReconciler) handleUpdate(cr *shmilav1.Go, secret *corev1.Secret) (ctrl.Result, error) {
	fmt.Printf("[INFO - handleUpdate] starting updating process for %s-%s\n", cr.Name, cr.Namespace)
	sd, err := readSecret(secret)

	if err != nil {
		fmt.Println("[ERROR - handleUpdate] error reading secret " + secret.Name)
		fmt.Println(err)
		setStatus(cr, "internal error - ERR_CODE=187", Failure)
		return retry, fmt.Errorf("internal error - ERR_CODE=187")
	}
	body := map[string]string{
		"alias":        sd.Alias,
		"url":          cr.Spec.Url,
		"password":     sd.Password,
		"passwordHint": "managed by go-operator",
	}
	json, _ := json.Marshal(body)
	res, err := httpClient.Post(goHostUrl+"/api/v1/go-links", "application/json", bytes.NewBuffer(json))

	if err != nil {
		fmt.Println("[ERROR - handleUpdate] error in post " + goHostUrl)
		fmt.Println(err)
		setStatus(cr, "go api unavailable right now", Pending)
		return retry, fmt.Errorf("internal error - ERR_CODE=196")
	} else {
		fmt.Println("[INFO - handleUpdate] success posting link ", sd.Alias)
	}

	defer res.Body.Close()

	if res.StatusCode == 403 || res.StatusCode == 401 {
		fmt.Println("[WARN - handleUpdate] link already exists")
		setStatus(cr, "alias "+cr.Spec.Alias+" already taken", Failure)
		return retry, nil
	}

	if res.StatusCode/100 != 2 {
		fmt.Printf("[ERROR - handleUpdate] bad status code for update request for link %s the status is: %d\n", sd.Alias, res.StatusCode)
		b, err2 := ioutil.ReadAll(res.Body)
		if err2 == nil {
			fmt.Println(string(b))
		}
		setStatus(cr, "internal error - ERR_CODE=209", Failure)
		return retry, fmt.Errorf("internal error - ERR_CODE=209")
	}

	return complete, nil
}

func getSecretObject(resourceName, namespace, operatorNs string) corev1.Secret {
	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretPrefix + namespace + "-" + resourceName,
			Namespace: operatorNs,
		},
	}
}

func readSecret(secret *corev1.Secret) (*secretData, error) {
	if secret.StringData != nil {
		return &secretData{
			Alias:             secret.StringData["alias"],
			Password:          secret.StringData["password"],
			ResourceName:      secret.StringData["resourceName"],
			ResourceNamespace: secret.StringData["resourceNamespace"],
		}, nil
	} else if secret.Data != nil {
		return &secretData{
			Alias:             string(secret.Data["alias"]),
			Password:          string(secret.Data["password"]),
			ResourceName:      string(secret.Data["resourceName"]),
			ResourceNamespace: string(secret.Data["resourceNamespace"]),
		}, nil
	} else {
		return nil, fmt.Errorf("both Data and StringData are nil in secret " + secret.Name)
	}
}

func cleanupLoop(mgr ctrl.Manager, interval time.Duration) {
	fmt.Println("[INFO - cleanupLoop] starting cleanup loop")
	for {
		cleanup(mgr)
		time.Sleep(interval)
	}
}

func cleanup(mgr ctrl.Manager) {
	fmt.Println("[INFO - cleanup] starting cleanup process")
	secrets := corev1.SecretList{}
	if err := mgr.GetClient().List(
		context.TODO(),
		&secrets,
		&client.ListOptions{Namespace: environment.GetVariables().ControllerNamespace},
	); err != nil {
		fmt.Println("[ERROR - cleanup] failed to list secrets")
		fmt.Println(err)
		return
	}

	for _, secret := range secrets.Items {
		if strings.HasPrefix(secret.Name, secretPrefix) {
			sd, err := readSecret(&secret)
			if err != nil {
				fmt.Println("[ERROR - cleanup] failed to read data from secret ", secret)
				fmt.Println(err)
			} else {
				cr := shmilav1.Go{}
				if err := mgr.GetClient().Get(
					context.TODO(),
					client.ObjectKey{
						Name:      sd.ResourceName,
						Namespace: sd.ResourceNamespace,
					},
					&cr); errors.IsNotFound(err) {
					handleDelete(mgr.GetClient(), context.TODO(), &secret)
				}
			}
		}
	}
}

const (
	Failure string = "Failure"
	Succees string = "Active"
	Pending string = "Pending"
)

func setStatus(cr *shmilav1.Go, message, state string) {
	cr.Status = shmilav1.GoStatus{
		Message:       message,
		State:         state,
		ReconcileTime: time.Now().Format(time.RFC3339),
	}
}
