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

// PermsClusterRoleBindingSpec defines the desired state of PermsClusterRoleBinding
type PermsClusterRoleBindingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of PermsClusterRoleBinding. Edit permsclusterrolebinding_types.go to remove/update
	Role            string           `json:"role"`
	Groups          []string         `json:"groups,omitempty"`
	Users           []string         `json:"user,omitempty"`
	Serviceaccounts []Serviceaccount `json:"serviceaccounts,omitempty"`
}

// PermsClusterRoleBindingStatus defines the observed state of PermsClusterRoleBinding
type PermsClusterRoleBindingStatus struct {
	Bindings   []string           `json:"bindings,omitempty"`
	Count      PCrbCount          `json:"count,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type PCrbCount struct {
	Users           string `json:"users,omitempty"`
	Groups          string `json:"groups,omitempty"`
	Serviceaccounts string `json:"serviceaccounts,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=permscrb;pcrb,scope=Cluster
//+kubebuilder:printcolumn:name=Users,type="string",JSONPath=".status.count.users"
//+kubebuilder:printcolumn:name=Groups,type=string,JSONPath=".status.count.groups"
//+kubebuilder:printcolumn:name=Serviceaccounts,type=string,JSONPath=".status.count.serviceaccounts"
//+kubebuilder:printcolumn:name=Available,type=string,JSONPath=".status.conditions[?(@.type==\"Available\")].status"
//+kubebuilder:printcolumn:name=Progressing,type=string,JSONPath=".status.conditions[?(@.type==\"Progressing\")].status"
//+kubebuilder:printcolumn:name=Degraded,type=string,JSONPath=".status.conditions[?(@.type==\"Degraded\")].status"

// PermsClusterRoleBinding is the Schema for the permsclusterrolebindings API
type PermsClusterRoleBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PermsClusterRoleBindingSpec   `json:"spec,omitempty"`
	Status PermsClusterRoleBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PermsClusterRoleBindingList contains a list of PermsClusterRoleBinding
type PermsClusterRoleBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PermsClusterRoleBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PermsClusterRoleBinding{}, &PermsClusterRoleBindingList{})
}
