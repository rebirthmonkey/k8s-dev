/*
Copyright 2023.

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
	"github.com/rebirthmonkey/go/pkg/log"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	toolv1 "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/tool/v1"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/pkg/workflow"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/pkg/workflow/activity"
)

var _ reconcile.Reconciler = &Reconciler{}

func init() {
	reconcilermgr.Register(func(rmgr *reconcilermgr.ReconcilerManager) error {
		utilruntime.Must(toolv1.AddToScheme(rmgr.GetScheme()))
		rmgr.With(&Reconciler{
			Client:      rmgr.GetClient(),
			Scheme:      rmgr.GetScheme(),
			Concurrence: rmgr.GetDefaultConcurrence(),
		})
		return nil
	})
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

// Reconciler reconciles a Workflow object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Concurrence int
	filter      string

	engine *workflow.Engine
}

//+kubebuilder:rbac:groups=tool.wukong.com,resources=workflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tool.wukong.com,resources=workflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tool.wukong.com,resources=workflows/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Workflow object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.WithValues("wf", req.Name)
	logger.Info("=== Reconciling Workflow")

	wf := &toolv1.Workflow{}
	err := r.Client.Get(ctx, req.NamespacedName, wf)
	if err != nil {
		logger.Errorf("%s", err)
		return reconcile.Result{}, err
	}

	switch wf.Status.Phase {
	case "":
		wf.Status.Phase = workflow.InitPhase
	case workflow.InitPhase:
		fmt.Println("================ InitPhase")

		wf.Status.Phase = workflow.ExecPhase
	case workflow.ExecPhase:
		fmt.Println("================ ExecPhase")

		wf.Status.Phase = workflow.FinalPhase
	case workflow.FinalPhase:
		fmt.Println("================ FinalPhase")
	}

	//wf.Status.Phase = r.engine.Reconcile(wf.Status.Phase)

	r.Client.Update(ctx, wf)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	template := workflow.NewTemplate()

	e1 := workflow.NewEvent("Start")
	e2 := workflow.NewEvent("Doing")
	e3 := workflow.NewEvent("End")
	template.AddVertex(e1)
	template.AddVertex(e2)
	template.AddVertex(e3)

	template.AddEdge(e1, e2)

	a1 := activity.NewBasic("a1")
	a2 := activity.NewShell("a2", "/tmp/test.sh", "aaa", "bbb")
	a3 := activity.NewGolang("a3", func(s interface{}) interface{} {
		fmt.Println("------------")
		return s.(string)
	}, "ccc")
	template.AddVertex(a1)
	template.AddVertex(a2)
	template.AddVertex(a3)

	template.AddEdge(e2, a1)
	template.AddEdge(a1, a2)
	template.AddEdge(a1, a3)

	g1 := workflow.NewGateway("g1")
	template.AddVertex(g1)

	template.AddEdge(a2, g1)
	template.AddEdge(a3, g1)
	template.AddEdge(g1, e3)

	r.engine = workflow.NewEngine(template)

	return ctrl.NewControllerManagedBy(mgr).
		For(&toolv1.Workflow{}).
		Complete(r)
}
