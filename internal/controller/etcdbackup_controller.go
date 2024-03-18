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

package controller

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/snapshot"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	etcdv1alpha1 "k8s2/etcd-backup/api/v1alpha1"
)

// EtcdBackupReconciler reconciles a EtcdBackup object
type EtcdBackupReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Log      logr.Logger
}

//+kubebuilder:rbac:groups=etcd.chenjie.info,resources=etcdbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etcd.chenjie.info,resources=etcdbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etcd.chenjie.info,resources=etcdbackups/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EtcdBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *EtcdBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO(user): your logic here
	_ = log.FromContext(ctx)
	loglog := r.Log.WithValues("etcdbackup", req.NamespacedName)
	backup := &etcdv1alpha1.EtcdBackup{}
	// Get backup and checkout is not found
	if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
		if client.IgnoreNotFound(err) != nil {
			loglog.Info("Reconciling EtcdBackup", "getting backup error", err)
			return ctrl.Result{}, fmt.Errorf("getting backup error: %s", err)
		}
		// Backup deleted
		loglog.Info("Reconciling EtcdBackup", "not found. . ", "Ignoring")
		return ctrl.Result{}, nil
	}
	// Check backup has been deleted
	if !backup.DeletionTimestamp.IsZero() {
		loglog.Info("Reconciling EtcdBackup", "has been deleted. . ", "Ignoring")
		return ctrl.Result{}, nil
	}
	// Check status phase
	if backup.Status.Phase != "" {
		loglog.Info("Reconciling EtcdBackup", "status phase is not empty ", "Ignoring")
		return ctrl.Result{}, nil
	}
	// update status
	newBackup := backup.DeepCopy()
	newBackup.Status.Phase = etcdv1alpha1.EtcdBackupPhaseBackingUp
	newBackup.Status.StartTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Patch(ctx, newBackup, client.MergeFrom(backup)); err != nil {
		loglog.Info("Reconciling EtcdBackup", "update status error", err)
		return ctrl.Result{}, fmt.Errorf("update status error: %s", err)
	}
	r.Recorder.Event(backup, corev1.EventTypeNormal, "BackupStart", "Backuping")

	tlsInfo := transport.TLSInfo{
		CertFile:      backup.Spec.Cert,
		KeyFile:       backup.Spec.Key,
		TrustedCAFile: backup.Spec.CaCert,
	}
	tlsConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		loglog.Info("Reconciling EtcdBackup", "tls config error", err)
		return ctrl.Result{}, fmt.Errorf("tls config error: %s", err)
	}
	cfg := clientv3.Config{
		Endpoints:   []string{backup.Spec.EtcdUrl},
		DialTimeout: 5 * time.Second,
		TLS:         tlsConfig,
	}
	err = os.MkdirAll(backup.Spec.BackupDir, 0755)
	if err != nil {
		loglog.Info("Reconciling EtcdBackup", "mkdir error", err)
		return ctrl.Result{}, fmt.Errorf("mkdir error: %s", err)
	}
	backupPath := backup.Spec.BackupDir + "/" + strconv.Itoa(time.Now().Hour()) + strconv.Itoa(time.Now().Minute()) + "-snapshot.db"
	err = snapshot.Save(ctx, zap.NewRaw(zap.UseDevMode(true)), cfg, backupPath)
	if err != nil {
		loglog.Info("Reconciling EtcdBackup", "snapshot save error", err)
		// update status
		newBackup = backup.DeepCopy()
		newBackup.Status.Phase = etcdv1alpha1.EtcdBackupPhaseFailed
		if err := r.Status().Patch(ctx, newBackup, client.MergeFrom(backup)); err != nil {
			loglog.Info("Reconciling EtcdBackup", "update status error", err)
			return ctrl.Result{}, fmt.Errorf("update status error: %s", err)
		}
		r.Recorder.Event(backup, corev1.EventTypeWarning, "BackupFailed", "Backup failed. See backup pod logs for details.")
		return ctrl.Result{}, fmt.Errorf("snapshot save error: %s", err)
	}

	// update status
	newBackup = backup.DeepCopy()
	newBackup.Status.Phase = etcdv1alpha1.EtcdBackupPhaseCompleted
	newBackup.Status.CompletionTime = &metav1.Time{Time: time.Now()}
	if err := r.Status().Patch(ctx, newBackup, client.MergeFrom(backup)); err != nil {
		loglog.Info("Reconciling EtcdBackup", "update status error", err)
		return ctrl.Result{}, fmt.Errorf("update status error: %s", err)
	}
	r.Recorder.Event(backup, corev1.EventTypeNormal, "BackupSucceeded", "Backup completed successfully")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etcdv1alpha1.EtcdBackup{}).
		Complete(r)
}
