/*


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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IBMVPCMachineSpec defines the desired state of IBMVPCMachine
type IBMVPCMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name of the instance
	Name string `json:"name,omitempty"`

	// Image is the id of OS image which would be install on the instance.
	// Example: r134-ed3f775f-ad7e-4e37-ae62-7199b4988b00
	// TODO: allow user to specify a image name is much reasonable. Example: ibm-ubuntu-18-04-1-minimal-amd64-2
	Image string `json:"image,required"`

	// Zone is the place where the instance should be created. Example: us-south-3
	// TODO: Actually zone is transparent to user. The field user can access is location. Example: Dallas 2
	Zone string `json:"zone,required"`

	// Profile indicates the flavor of instance. Example: bx2-8x32	means 8 vCPUs	32 GB RAM	16 Gbps
	// TODO: add a reference link of profile
	Profile string `json:"profile,required"`

	// PrimaryNetworkInterface is required to specify subnet
	PrimaryNetworkInterface NetworkInterface `json:"primaryNetworkInterface"`
}

// IBMVPCMachineStatus defines the observed state of IBMVPCMachine
type IBMVPCMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// IBMVPCMachine is the Schema for the ibmvpcmachines API
type IBMVPCMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IBMVPCMachineSpec   `json:"spec,omitempty"`
	Status IBMVPCMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IBMVPCMachineList contains a list of IBMVPCMachine
type IBMVPCMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IBMVPCMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IBMVPCMachine{}, &IBMVPCMachineList{})
}
