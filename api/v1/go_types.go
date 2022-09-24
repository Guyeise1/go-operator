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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// defines the desired state of your GoLink
type GoSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^([a-z0-9א-ת]+)(-[a-z0-9א-ת]+)*$"
	// the shorten name for your link
	// format must kebab case e.g.: "my-first-go-link"
	Alias string `json:"alias"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^https?://.*$"
	// the url that go/your-alias will redirect to
	Url string `json:"url"`
}

// Status of your GoLink
type GoStatus struct {
	// +kubebuilder:validation:Optional
	Message string `json:"message"`
	// +kubebuilder:validation:Optional
	State string `json:"state"`
	// +kubebuilder:validation:Optional
	ReconcileTime string `json:"reconcileTime"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Alias",type="string",JSONPath=".spec.alias",description="The shorten name"
//+kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.url",description="The URL"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.state"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:printcolumn:name="Message",type="string",priority=1,JSONPath=".status.message"

// Go is the Schema for the goes API
type Go struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GoSpec   `json:"spec,omitempty"`
	Status GoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GoList contains a list of Go
type GoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Go `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Go{}, &GoList{})
}
