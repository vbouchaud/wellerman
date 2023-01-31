/*
Copyright 2023.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ProjectPath struct {
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// +kubebuilder:validation:Required
	Path string `json:"path"`

	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	External bool `json:"external,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=true
	ArchiveOnDelete bool `json:"archive-on-delete,omitempty"`
}

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	Paths []ProjectPath `json:"paths"`
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Project is the Schema for the projects API
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
