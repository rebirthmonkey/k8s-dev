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
	"strings"
	"time"

	"github.com/rebirthmonkey/go/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr/registry"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
	demov1 "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/demo/v1"
)

var _ reconcile.Reconciler = &Reconciler{}

func init() {
	registry.Register(func(rmgr *reconcilermgr.ReconcilerManager) error {
		utilruntime.Must(corev1.AddToScheme(rmgr.GetScheme())) // because we will use Pod.
		utilruntime.Must(demov1.AddToScheme(rmgr.GetScheme()))
		rmgr.With(&Reconciler{
			Client:      rmgr.GetClient(),
			Scheme:      rmgr.GetScheme(),
			Concurrence: rmgr.GetDefaultConcurrence(),
		})
		return nil
	})
}

// Reconciler reconciles a At object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Concurrence int
	filter      string
}

func (r *Reconciler) KeyFilter(filter string) {
	r.filter = filter
}

func (r *Reconciler) For() string {
	return apis.ResourceAts
}

func (r *Reconciler) AfterCacheSync(mgr ctrl.Manager) error {
	// TODO initialization code to be executed after client cache synchronized
	return nil
}

//+kubebuilder:rbac:groups=demo.wukong.com,resources=ats,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.wukong.com,resources=ats/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.wukong.com,resources=ats/finalizers,verbs=update

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.WithValues("at", request.Name)
	logger.Info("=== Reconciling At")

	at := &demov1.At{}
	err := r.Get(context.TODO(), request.NamespacedName, at)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request - return and don't requeue:
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if at.Status.Phase == "" {
		at.Status.Phase = demov1.AtPhasePending
	}

	switch at.Status.Phase { // state machine: PENDING -> RUNNING -> DONE
	case demov1.AtPhasePending:
		logger.Info("=== Phase: PENDING")

		logger.Infof("Checking schedule: %s", at.Spec.Schedule)
		d, err := timeUntilSchedule(at.Spec.Schedule)
		if err != nil {
			logger.Errorf("Schedule parsing failure %s", err)
			return reconcile.Result{}, err
		}

		logger.Infof("Schedule parsing done with Result %d", d)
		if d > 0 {
			return reconcile.Result{RequeueAfter: d}, nil
		}

		logger.Infof("It's time! Ready to execute the cmd: %s", at.Spec.Command)
		at.Status.Phase = demov1.AtPhaseRunning
	case demov1.AtPhaseRunning:
		logger.Info("=== Phase: RUNNING")
		pod := newPodForCR(at)
		// Set at as the owner and controller
		if err := controllerutil.SetControllerReference(at, pod, r.Scheme); err != nil {
			return reconcile.Result{}, err // requeue with error
		}

		found := &corev1.Pod{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
		if err != nil {
			if errors.IsNotFound(err) { // Try to see if the pod already exists
				err2 := r.Create(context.TODO(), pod)
				if err2 != nil {
					return reconcile.Result{}, err2
				}
				logger.Infof("Pod %s launched", pod.Name)
			}
			at.Status.Phase = demov1.AtPhaseDone
			return reconcile.Result{}, err
		}

		if found.Status.Phase == corev1.PodFailed || found.Status.Phase == corev1.PodSucceeded {
			logger.Infof("Container terminated with reason: %s, and message: %s", found.Status.Reason, found.Status.Message)
			at.Status.Phase = demov1.AtPhaseDone
		}

		at.Status.Phase = demov1.AtPhaseDone
		return reconcile.Result{}, nil
	case demov1.AtPhaseDone:
		logger.Info("=== Phase: DONE")
		return reconcile.Result{}, nil
	default:
		logger.Info("=== Phase: NOP")
		return reconcile.Result{}, nil
	}

	// Update the at, setting the status to the respective phase
	err = r.Status().Update(context.TODO(), at)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *demov1.At) *corev1.Pod {
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
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.At{}).
		Complete(r)
}
