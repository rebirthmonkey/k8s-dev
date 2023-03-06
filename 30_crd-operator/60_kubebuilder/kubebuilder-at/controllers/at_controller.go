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

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	atv1 "github.com/rebirthmonkey/k8s-dev/kubebuilder-at/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("at-controller")

var _ reconcile.Reconciler = &AtReconciler{}

// AtReconciler reconciles a At object
type AtReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=at.wukong.com,resources=ats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=at.wukong.com,resources=ats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=at.wukong.com,resources=ats/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *AtReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("namespace", request.Namespace, "at", request.Name)
	reqLogger.Info("=== Reconciling At")

	// Fetch the At instance
	atInstance := &atv1.At{}
	err := r.Get(context.TODO(), request.NamespacedName, atInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not foundPod, could have been deleted after reconcile request - return and don't requeue:
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// If no phase set, default to pending (the initial phase):
	if atInstance.Status.Phase == "" {
		atInstance.Status.Phase = atv1.PhasePending
	}

	// the state diagram PENDING -> RUNNING -> DONE
	switch atInstance.Status.Phase {
	case atv1.PhasePending:
		reqLogger.Info("Phase: PENDING")

		// As long as we haven't executed the command yet, we need to check if it's time already to act:
		reqLogger.Info("Checking schedule", "Target", atInstance.Spec.Schedule)
		// Check if it's already time to execute the command with a tolerance of 2 seconds:
		d, err := timeUntilSchedule(atInstance.Spec.Schedule)
		if err != nil {
			reqLogger.Error(err, "Schedule parsing failure")
			// Error reading the schedule. Wait until it is fixed.
			return reconcile.Result{}, err
		}

		reqLogger.Info("Schedule parsing done", "Result", fmt.Sprintf("diff=%v", d))
		if d > 0 {
			// Not yet time to execute the command, wait until the scheduled time
			return reconcile.Result{RequeueAfter: d}, nil
		}

		reqLogger.Info("It's time!", "Ready to execute", atInstance.Spec.Command)
		atInstance.Status.Phase = atv1.PhaseRunning
	case atv1.PhaseRunning:
		reqLogger.Info("Phase: RUNNING")

		executionPod := newExecutionPod(atInstance)

		// Set At atInstance as the owner and controller
		if err := controllerutil.SetControllerReference(atInstance, executionPod, r.Scheme); err != nil {
			// requeue with error
			return reconcile.Result{}, err
		}

		foundPod := &corev1.Pod{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: executionPod.Name, Namespace: executionPod.Namespace}, foundPod)
		// Try to see if the executionPod already exists and if not
		// (which we expect) then create a one-shot executionPod as per spec:
		if err != nil && errors.IsNotFound(err) {
			err = r.Create(context.TODO(), executionPod) // launch the execution pod
			if err != nil {
				// requeue with error
				return reconcile.Result{}, err
			}
			reqLogger.Info("Pod launched", "name", executionPod.Name)
			atInstance.Status.Phase = atv1.PhaseDone
		} else if err != nil {
			// requeue with error
			return reconcile.Result{}, err
		} else if foundPod.Status.Phase == corev1.PodFailed || foundPod.Status.Phase == corev1.PodSucceeded {
			reqLogger.Info("Pod terminated", "reason", foundPod.Status.Reason, "message", foundPod.Status.Message)
			atInstance.Status.Phase = atv1.PhaseDone
		} else {
			// don't requeue because it will happen automatically when the executionPod status changes
			return reconcile.Result{}, nil
		}

	case atv1.PhaseDone:
		reqLogger.Info("Phase: DONE")
		return reconcile.Result{}, nil

	default:
		reqLogger.Info("NOP")
		return reconcile.Result{}, nil
	}

	// Update the At instance, setting the status to the respective phase:
	err = r.Status().Update(context.TODO(), atInstance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Don't requeue. We should be reconcile because either the executionPod or the CR changes.
	return reconcile.Result{}, nil
}

// newExecutionPod returns a busybox pod with the same name/namespace as the At
func newExecutionPod(cr *atv1.At) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-execution-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: strings.Split(cr.Spec.Command, " "),
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
		},
	}
}

// timeUntilSchedule parses the schedule string and returns the time until the schedule.
// When it is overdue, the duration is negative.
func timeUntilSchedule(schedule string) (time.Duration, error) {
	now := time.Now().UTC()
	layout := "2006-01-02T15:04:05Z"
	s, err := time.Parse(layout, schedule)
	if err != nil {
		return time.Duration(0), err
	}
	return s.Sub(now), nil
}

// SetupWithManager sets up the controller with the APIExtManager.
func (r *AtReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&atv1.At{}).
		Complete(r)
}
