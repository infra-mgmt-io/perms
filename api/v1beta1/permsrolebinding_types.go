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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PermsRoleBindingSpec defines the desired state of PermsRoleBinding
type PermsRoleBindingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of PermsRoleBinding. Edit permsrolebinding_types.go to remove/update
	Kind            string           `json:"kind"`
	Role            string           `json:"role"`
	Groups          []string         `json:"groups,omitempty"`
	Users           []string         `json:"user,omitempty"`
	Serviceaccounts []Serviceaccount `json:"serviceaccounts,omitempty"`
}

type Serviceaccount struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// PermsRoleBindingStatus defines the observed state of PermsRoleBinding
type PermsRoleBindingStatus struct {
	Bindings   []string           `json:"bindings,omitempty"`
	Count      PrbCount           `json:"count,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type PrbCount struct {
	Users           string `json:"users,omitempty"`
	Groups          string `json:"groups,omitempty"`
	Serviceaccounts string `json:"serviceaccounts,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=permsrolebindings,shortName=permsrb;prb
//+kubebuilder:printcolumn:name=Users,type="string",JSONPath=".status.count.users"
//+kubebuilder:printcolumn:name=Groups,type=string,JSONPath=".status.count.groups"
//+kubebuilder:printcolumn:name=Serviceaccounts,type=string,JSONPath=".status.count.serviceaccounts"
//+kubebuilder:printcolumn:name=Available,type=string,JSONPath=".status.conditions[?(@.type==\"Available\")].status"
//+kubebuilder:printcolumn:name=Progressing,type=string,JSONPath=".status.conditions[?(@.type==\"Progressing\")].status"
//+kubebuilder:printcolumn:name=Degraded,type=string,JSONPath=".status.conditions[?(@.type==\"Degraded\")].status"

// PermsRoleBinding is the Schema for the permsrolebindings API
type PermsRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PermsRoleBindingSpec   `json:"spec,omitempty"`
	Status PermsRoleBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PermsRoleBindingList contains a list of PermsRoleBinding
type PermsRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PermsRoleBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PermsRoleBinding{}, &PermsRoleBindingList{})
}
