/*
Copyright 2024.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EtcdBackupSpec defines the desired state of EtcdBackup
type EtcdBackupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	/*
			ETCDCTL_API=3 etcdctl --endpoints=https://127.0.0.1:2379
		   --cacert=/etc/ssl/etcd/ssl/ca.pem --cert=/etc/ssl/etcd/ssl/member-node1.pem
		   --key=/etc/ssl/etcd/ssl/member-node1-key.pem  snapshot save /data/etcd_backup_dir/etcd-snapshot.db
	*/
	// Specific Backup Etcd Endpoints.
	EtcdUrl string `json:"etcdUrl"`

	// Certificate.
	CaCert string `json:"caCert"`
	Cert   string `json:"cert"`
	Key    string `json:"key"`

	// Backup Directory.
	BackupDir string `json:"backupDir"`
}

type EtcdBackupPhase string

var (
	EtcdBackupPhaseBackingUp EtcdBackupPhase = "BackingUp"
	EtcdBackupPhaseCompleted EtcdBackupPhase = "Completed"
	EtcdBackupPhaseFailed    EtcdBackupPhase = "Failed"
)

// EtcdBackupStatus defines the observed state of EtcdBackup
type EtcdBackupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase defines the current operation that the backup process is taking.
	Phase EtcdBackupPhase `json:"phase,omitempty"`
	// StartTime is the times that this backup entered the `BackingUp' phase.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// CompletionTime is the time that this backup entered the `Completed' phase.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// EtcdBackup is the Schema for the etcdbackups API
type EtcdBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EtcdBackupSpec   `json:"spec,omitempty"`
	Status EtcdBackupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EtcdBackupList contains a list of EtcdBackup
type EtcdBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EtcdBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EtcdBackup{}, &EtcdBackupList{})
}
