package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// House represents a house.
type House struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec HouseSpec `json:"spec,omitempty"`
}

// HouseSpec is the spec for a Team resource.
type HouseSpec struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Active  *bool  `json:"active,omitempty"`
	// +listType=map
	// +listMapKey=name
	Owners []User `json:"owners,omitempty"`
}

// User is an user.
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HouseList is a list of House resources.
type HouseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []House `json:"items"`
}
