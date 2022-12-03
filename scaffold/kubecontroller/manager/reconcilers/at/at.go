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

package at

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rebirthmonkey/go/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	atv1 "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/demo/v1"
)

// Add creates a new At Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &AtReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("at-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to At
	err = c.Watch(&source.Kind{Type: &atv1.At{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch pods created by At
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &atv1.At{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &AtReconciler{}

// AtReconciler reconciles a At object
type AtReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=demo.wukong.com,resources=ats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.wukong.com,resources=ats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.wukong.com,resources=ats/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *AtReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("namespace", request.Namespace, "at", request.Name)
	reqLogger.Info("=== Reconciling At")
	// Fetch the At instance
	instance := &atv1.At{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request - return and don't requeue:
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// If no phase set, default to pending (the initial phase):
	if instance.Status.Phase == "" {
		instance.Status.Phase = atv1.PhasePending
	}

	// Now let's make the main case distinction: implementing
	// the state diagram PENDING -> RUNNING -> DONE
	switch instance.Status.Phase {
	case atv1.PhasePending:
		reqLogger.Info("Phase: PENDING")
		// As long as we haven't executed the command yet, we need to check if it's time already to act:
		reqLogger.Infof("Checking schedule", "Target", instance.Spec.Schedule)
		// Check if it's already time to execute the command with a tolerance of 2 seconds:
		d, err := timeUntilSchedule(instance.Spec.Schedule)
		if err != nil {
			reqLogger.Errorf("Schedule parsing failure", err)
			// Error reading the schedule. Wait until it is fixed.
			return reconcile.Result{}, err
		}
		reqLogger.Infof("Schedule parsing done", "Result", fmt.Sprintf("diff=%v", d))
		if d > 0 {
			// Not yet time to execute the command, wait until the scheduled time
			return reconcile.Result{RequeueAfter: d}, nil
		}
		reqLogger.Infof("It's time!", "Ready to execute", instance.Spec.Command)
		instance.Status.Phase = atv1.PhaseRunning
	case atv1.PhaseRunning:
		reqLogger.Info("Phase: RUNNING")
		pod := newPodForCR(instance)
		// Set At instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, pod, r.Scheme); err != nil {
			// requeue with error
			return reconcile.Result{}, err
		}
		found := &corev1.Pod{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
		// Try to see if the pod already exists and if not
		// (which we expect) then create a one-shot pod as per spec:
		if err != nil && errors.IsNotFound(err) {
			err = r.Create(context.TODO(), pod)
			if err != nil {
				// requeue with error
				return reconcile.Result{}, err
			}
			reqLogger.Infof("Pod launched", "name", pod.Name)
		} else if err != nil {
			// requeue with error
			return reconcile.Result{}, err
		} else if found.Status.Phase == corev1.PodFailed || found.Status.Phase == corev1.PodSucceeded {
			reqLogger.Infof("Container terminated", "reason", found.Status.Reason, "message", found.Status.Message)
			instance.Status.Phase = atv1.PhaseDone
		} else {
			// don't requeue because it will happen automatically when the pod status changes
			return reconcile.Result{}, nil
		}
	case atv1.PhaseDone:
		reqLogger.Infof("Phase: DONE")
		return reconcile.Result{}, nil
	default:
		reqLogger.Info("NOP")
		return reconcile.Result{}, nil
	}

	// Update the At instance, setting the status to the respective phase:
	err = r.Status().Update(context.TODO(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Don't requeue. We should be reconcile because either the pod or the CR changes.
	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *atv1.At) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
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

// SetupWithManager sets up the controller with the Manager.
func (r *AtReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&atv1.At{}).
		Complete(r)
}
