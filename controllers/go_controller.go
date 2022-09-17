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
	"net/http"

	"io/ioutil"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	shmilav1 "github.com/Guyeise1/go-operator/api/v1"
)

// GoReconciler reconciles a Go object
type GoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}
type secretData struct {
	Alias    string
	Password string
}

var goHostUrl = "http://localhost:14000" // TODO: make this from env

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
	cr := shmilav1.Go{}
	crErr := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, &cr)

	operatorNs := "go-operator-system"
	secret := getSecretObject(req.Name, req.Namespace, operatorNs)

	secErr := r.Get(ctx, client.ObjectKey{Namespace: secret.Namespace, Name: secret.Name}, &secret)

	if errors.IsNotFound(crErr) {
		fmt.Println("handle delete for " + req.Name)
		err := handleDelete(r, ctx, &secret)
		return ctrl.Result{}, err
	}

	if errors.IsNotFound(secErr) {
		fmt.Println("secret " + secret.Name + " Not found, creating...")
		return ctrl.Result{}, handleCreate(r, ctx, &cr, &secret)
	} else {
		return ctrl.Result{}, handleUpdate(&cr, &secret)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *GoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&shmilav1.Go{}).
		Complete(r)
}

func handleCreate(r *GoReconciler, ctx context.Context, cr *shmilav1.Go, secret *corev1.Secret) error {
	data := map[string]string{
		"alias":    cr.Spec.Alias,
		"password": "hello", // TODO: generate
	}
	secret.StringData = data
	secret.ResourceVersion = ""
	err1 := r.Create(ctx, secret)
	if err1 != nil {
		fmt.Println("failed to create secret")
		fmt.Println(err1)
		return fmt.Errorf("internal error")
	}

	err2 := handleUpdate(cr, secret)
	if err2 != nil {
		fmt.Println(err2)
		return fmt.Errorf("internal error")
	}

	return nil
}

func handleDelete(r *GoReconciler, ctx context.Context, secret *corev1.Secret) error {
	secretData, err := readSecret(secret)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("internal error")
	}

	fmt.Printf("body: %s\n", secretData)
	body := map[string]string{"alias": secretData.Alias, "password": secretData.Password}
	json, _ := json.Marshal(body)

	req, _ := http.NewRequest("DELETE", goHostUrl+"/api/v1/links", bytes.NewBuffer(json))
	req.Header.Add("content-type", "application/json")
	client := http.Client{}
	res, err4 := client.Do(req)

	fmt.Println("executed delete request")

	if err4 != nil {
		fmt.Println("Failed to delete !")
		fmt.Println(err)
		return fmt.Errorf("internal error")
	}

	// password is wrong
	if res.StatusCode/100 == 4 {
		fmt.Println("password is incorrect")
		return fmt.Errorf("internal error")
	}

	err = r.Delete(ctx, secret)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func handleUpdate(cr *shmilav1.Go, secret *corev1.Secret) error {
	sd, err := readSecret(secret)

	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("internal error")
	}
	body := map[string]string{"alias": sd.Alias, "url": cr.Spec.Url, "password": sd.Password}
	json, _ := json.Marshal(body)
	res, err := http.Post(goHostUrl+"/api/v1/links", "application/json", bytes.NewBuffer(json))

	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("internal error")
	}

	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		fmt.Printf("bad status code for update request %d\n", res.StatusCode)
		b, err2 := ioutil.ReadAll(res.Body)
		if err2 == nil {
			fmt.Println(string(b))
		}
		fmt.Printf("response body is ")
		return fmt.Errorf("internal error")
	}

	return nil

}

func getSecretObject(resourceName, namespace, operatorNs string) corev1.Secret {
	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "go-" + namespace + "-" + resourceName,
			Namespace: operatorNs,
		},
	}
}

func readSecret(secret *corev1.Secret) (*secretData, error) {
	if secret.StringData != nil {
		fmt.Println("Reading secret data from StringData")
		return &secretData{Alias: secret.StringData["alias"], Password: secret.StringData["password"]}, nil
	} else {
		fmt.Println("Reading secret data from Data")
		aliasBytes := string(secret.Data["alias"])
		passwordBytes := string(secret.Data["password"])

		return &secretData{Alias: string(aliasBytes), Password: string(passwordBytes)}, nil
	}
}
