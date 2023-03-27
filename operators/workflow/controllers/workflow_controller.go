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

	"github.com/goombaio/dag"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	toolv1 "github.com/rebirthmonkey/k8s-dev/operators/workflow/api/v1"
	"github.com/rebirthmonkey/k8s-dev/operators/workflow/pkg/workflow"
	"github.com/rebirthmonkey/k8s-dev/operators/workflow/pkg/workflow/activity"
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Engine *workflow.Engine
}

//+kubebuilder:rbac:groups=tool.wukong.com,resources=workflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tool.wukong.com,resources=workflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tool.wukong.com,resources=workflows/finalizers,verbs=update

func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	fmt.Println("================ Reconcile")

	wf := &toolv1.Workflow{}
	err := r.Client.Get(ctx, req.NamespacedName, wf)
	if err != nil {
		return reconcile.Result{}, err
	}

	switch wf.Status.Phase {
	case "":
		wf.Status.Phase = workflow.InitPhase
	case workflow.InitPhase:
		fmt.Println("================ InitPhase")

		t := workflow.NewTemplate()
		r.Engine = workflow.NewEngine(t)

		for _, e := range wf.Spec.Events {
			fmt.Println("Adding event: ", e)
			r.Engine.Template.AddVertex(e, workflow.NewEvent(e))
		}

		for _, a := range wf.Spec.Activities {
			fmt.Println("Adding activity: ", a)
			r.Engine.Template.AddVertex(a, activity.NewBasic(a))
		}

		for _, g := range wf.Spec.Gateways {
			fmt.Println("Adding gateway: ", g)
			r.Engine.Template.AddVertex(g, workflow.NewGateway(g))
		}

		for _, e := range wf.Spec.Edges {
			fmt.Println("Adding edges: ", e)
			r.Engine.Template.AddEdge(e.Obj1, e.Obj2)
		}

		wf.Status.Phase = workflow.ExecPhase
	case workflow.ExecPhase:
		fmt.Println("================ ExecPhase")

		sourceVertexes := r.Engine.Template.DAG.SourceVertices()
		for _, sourcev := range sourceVertexes {
			fmt.Println("the source vertex is: ", sourcev.ID)
			r.Engine.EntrySet.Insert(sourcev)
			r.Engine.Entries.Enqueue(sourcev)
		}

		for r.Engine.Entries.Len() != 0 {
			val := r.Engine.Entries.Dequeue()
			r.Engine.EntrySet.Remove(val)

			r.Engine.Template.Vertex2Object[val.(*dag.Vertex)].Execute()

			vx, _ := r.Engine.Template.DAG.Successors(val.(*dag.Vertex))
			for _, pv := range vx {
				if r.Engine.EntrySet.Has(pv) == false {
					r.Engine.Entries.Enqueue(pv)
					r.Engine.EntrySet.Insert(pv)
				}
			}
		}

		wf.Status.Phase = workflow.FinalPhase
	case workflow.FinalPhase:
		fmt.Println("================ FinalPhase")
		return ctrl.Result{}, nil
	}

	if err := r.Status().Update(ctx, wf); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{Requeue: true}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&toolv1.Workflow{}).
		Complete(r)
}
