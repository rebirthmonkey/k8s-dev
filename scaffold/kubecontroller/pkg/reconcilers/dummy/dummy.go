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

package dummy

import (
	"context"
	"time"

	"github.com/rebirthmonkey/go/pkg/log"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr/registry"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
	demov1 "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/demo/v1"
)

var _ reconcile.Reconciler = &Reconciler{}

func init() {
	registry.Register(func(rmgr *reconcilermgr.ReconcilerManager) error {
		utilruntime.Must(demov1.AddToScheme(rmgr.GetScheme()))
		rmgr.With(&Reconciler{
			Client: rmgr.GetClient(),
			Scheme: rmgr.GetScheme(),
		})
		return nil
	})
}

// Reconciler reconciles a Dummy object
type Reconciler struct {
	client.Client
	*runtime.Scheme

	filter string
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

//+kubebuilder:rbac:groups=demo.wukong.com,resources=dummies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.wukong.com,resources=dummies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.wukong.com,resources=dummies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.WithValues("dummy", request.Name)
	logger.Info("=== Reconciling Dummy")

	dummy := &demov1.Dummy{}
	err := r.Client.Get(ctx, request.NamespacedName, dummy)
	if err != nil {
		logger.Errorf("%s", err)
		return reconcile.Result{}, err
	}

	logger.Infof("=== Sleep %d seconds", dummy.Spec.TransitionDefer)
	time.Sleep(time.Second * time.Duration(dummy.Spec.TransitionDefer))

	logger.Infof("=== Update data to %s", dummy.Spec.Data)
	dummy.Status.Data = dummy.Spec.Data
	err = r.Client.Status().Update(ctx, dummy)
	if err != nil {
		logger.Errorf("%s", err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.Dummy{}).
		Complete(r)
}
