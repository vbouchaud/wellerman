/*
Copyright 2023.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TeamSpec defines the desired state of Team
type TeamSpec struct {
	Comment string `json:"comment,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems:=1
	Subjects []string `json:"subjects"`
}

// TeamStatus defines the observed state of Team
type TeamStatus struct {
	Conditions        []metav1.Condition `json:"conditions"`
	DistinguishedName string             `json:"dn,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Team is the Schema for the teams API
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec,omitempty"`
	Status TeamStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}
