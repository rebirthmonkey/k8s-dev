package banana

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/rebirthmonkey/k8s-dev/pkg/apis"
	"github.com/rebirthmonkey/k8s-dev/pkg/pm"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	controllerapis "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
	demov1 "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/demo/v1"
)

const (
	Init    = pm.Phase("Init")
	Migrate = pm.Phase("Migrate")
	Success = pm.Phase("Success")
	Fail    = pm.Phase("Fail")
)

var (
	TerminalPhases = []pm.Phase{Fail, Success}
	FailedPhases   = []pm.Phase{Fail}
)

func init() {
	apis.SetPhaseMachine(controllerapis.ResourceBananas, PhaseMachineDef)
	reconcilermgr.Register(func(rmgr *reconcilermgr.ReconcilerManager) error {
		rmgr.WithPhaseMachine(&demov1.Banana{}, pm.New(PhaseMachineDef()))
		return nil
	})
}

func PhaseMachineDef() *pm.DefaultDef {
	def := &pm.DefaultDef{
		ResourceType:   reflect.TypeOf(&demov1.Banana{}),
		RequeueDefer:   5 * time.Second,
		InitialPhase:   Init,
		TerminalPhases: TerminalPhases,
		FailedPhases:   FailedPhases,
		FetchResourceFunc: func(ctx context.Context, key string) (pm.Resource, error) {
			cli := ctx.Value(apis.ContextKeyClient).(client.Client)
			obj := &demov1.Banana{}
			err := cli.Get(ctx, pm.ParseNamespacedName(key), obj)
			return obj, err
		},
		PersistStatusFunc: func(ctx context.Context, res pm.Resource) error {
			cli := ctx.Value(apis.ContextKeyClient).(client.Client)
			obj := res.(*demov1.Banana)
			return cli.Status().Update(ctx, obj)
		},
		SetPhaseFunc: func(res pm.Resource, phase pm.Phase) {
			obj := res.(*demov1.Banana)
			obj.Status.Phase = phase
		},
		NormalTrans: map[pm.Phase]pm.Phase{
			Init:    Migrate,
			Migrate: Success,
		},
		ErrorTrans: map[pm.Phase]pm.Phase{
			Init:    Fail,
			Migrate: Fail,
		},
		Handlers: map[pm.Phase]pm.Handler{
			Init: pm.HandlerFunc(func(ctx context.Context, res pm.Resource, last pm.State) (rstate pm.ReconcileState) {
				obj := res.(*demov1.Banana)
				fmt.Printf("准备为 %s 开始\n", obj.Name)

				if obj.Spec.Source != "" {
					obj.Status.SourceStatus = map[string]string{
						"id":   obj.Spec.Source,
						"name": strings.ToUpper(obj.Spec.Source),
					}
				}
				if obj.Spec.Dest != "" {
					obj.Status.DestStatus = map[string]string{
						"id":   obj.Spec.Dest,
						"name": strings.ToUpper(obj.Spec.Dest),
					}
				}

				rstate.Done = true
				return
			}, pm.Metadata{}),
			Migrate: pm.HandlerFunc(func(ctx context.Context, res pm.Resource, last pm.State) (rstate pm.ReconcileState) {
				obj := res.(*demov1.Banana)
				if obj.Status.SourceStatus == nil || obj.Status.DestStatus == nil {
					reason := "Banana信息缺失，执行失败"
					fmt.Printf(reason + "\n")
					rstate.Error = errors.New(reason)
					return
				}
				fmt.Printf("正在执行Banana……\n")
				time.Sleep(time.Second * 30)
				fmt.Printf("Banana%s执行成功%s\n", obj.Status.SourceStatus["name"], obj.Status.DestStatus["name"])

				rstate.Done = true
				return
			}, pm.Metadata{}),
		},
	}
	def.Normalize()
	return def
}
