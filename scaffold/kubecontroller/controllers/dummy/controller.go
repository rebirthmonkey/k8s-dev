package dummy

import (
	"context"
	"time"

	uberzap "go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
	appv1 "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/app/v1"
	//kcctrl "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/pkg/controller"
	"github.com/rebirthmonkey/k8s-dev/pkg/controller/registry"
	"github.com/rebirthmonkey/k8s-dev/pkg/manager"
)

func init() {
	registry.Register(func(manager *manager.ReconcilerManager) error {
		manager.With(&Reconciler{
			Client:      manager.GetClient(),
			ZapOpts:     manager.GetLogOptions(),
			Concurrence: manager.GetDefaultConcurrence(),
		})
		return nil
	})
}

type Reconciler struct {
	client.Client
	ZapOpts     zap.Options
	Concurrence int
	filter      string
}

func (r *Reconciler) KeyFilter(filter string) {
	r.filter = filter
}

func (r *Reconciler) For() string {
	return apis.ResourceDummys
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.ZapOpts.ZapOpts = []uberzap.Option{uberzap.AddCallerSkip(-1)}
	//logger := zap.New(zap.UseFlagOptions(&r.ZapOpts))
	r.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).WithOptions(controller.Options{
		MaxConcurrentReconciles: r.Concurrence,
		//Log:                     logger,
	}).For(&appv1.Dummy{}).WithEventFilter(predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return r.predicate(event.Object)
		},
		DeleteFunc: func(event event.DeleteEvent) bool {
			return r.predicate(event.Object)
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			return r.predicate(event.ObjectNew)
		},
	}).Complete(r)
}

func (r *Reconciler) predicate(obj client.Object) bool {
	//if !kcctrl.FilterObject(r.filter, obj) {
	//	return false
	//}
	return true
}

func (r *Reconciler) AfterCacheSync(mgr ctrl.Manager) error {
	// TODO initialization code to be executed after client cache synchronized
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (result reconcile.Result, err error) {
	// TODO main controller code here
	dummy := &appv1.Dummy{}
	r.Client.Get(ctx, req.NamespacedName, dummy)
	time.Sleep(time.Second * time.Duration(dummy.Spec.TransitionDefer))
	dummy.Status.Data = dummy.Spec.Data
	err = r.Client.Status().Update(ctx, dummy)
	return
}
