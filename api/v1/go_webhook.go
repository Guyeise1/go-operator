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

package v1

import (
	"fmt"
	"net/http"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var golog = logf.Log.WithName("go-resource")

func (r *Go) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-shmila-iaf-v1-go,mutating=false,failurePolicy=fail,sideEffects=None,groups=shmila.iaf,resources=goes,verbs=create;update,versions=v1,name=vgo.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Go{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Go) ValidateCreate() error {
	goHost := os.Getenv("GO_API_SERVER")
	golog.Info("validate create", "name", r.Name)
	res, err := http.Get(goHost + "/api/v1/links/" + r.Spec.Alias)

	if err != nil {
		return err
	} else if res.StatusCode != 404 {
		return fmt.Errorf("alias " + r.Spec.Alias + " already exists")
	}

	golog.Info("validate create succeess", "request", goHost+"/api/v1/links/"+r.Spec.Alias, "response", res.StatusCode)
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Go) ValidateUpdate(old runtime.Object) error {
	golog.Info("validate update", "name", r.Name)
	prev := old.(*Go)
	if prev.Spec.Alias != r.Spec.Alias {
		return fmt.Errorf("alias field can not be changed")
	} else {
		return nil
	}
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Go) ValidateDelete() error {
	golog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
