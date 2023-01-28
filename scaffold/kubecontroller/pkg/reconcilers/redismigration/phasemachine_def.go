package redismigration

import (
	"context"
	"errors"
	"fmt"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"reflect"
	"strings"
	"time"

	"github.com/rebirthmonkey/k8s-dev/pkg/apis"
	"github.com/rebirthmonkey/k8s-dev/pkg/pm"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr/registry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	controllerapis "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
	databasev1 "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/database/v1"
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
	apis.SetPhaseMachine(controllerapis.ResourceRedisMigrations, PhaseMachineDef)
	registry.Register(func(rmgr *reconcilermgr.ReconcilerManager) error {
		utilruntime.Must(databasev1.AddToScheme(rmgr.GetScheme()))
		rmgr.WithPhaseMachine(&databasev1.RedisMigration{}, pm.New(PhaseMachineDef()))
		return nil
	})
}

func PhaseMachineDef() *pm.DefaultDef {
	def := &pm.DefaultDef{
		ResourceType:   reflect.TypeOf(&databasev1.RedisMigration{}),
		RequeueDefer:   5 * time.Second,
		InitialPhase:   Init,
		TerminalPhases: TerminalPhases,
		FailedPhases:   FailedPhases,
		FetchResourceFunc: func(ctx context.Context, key string) (pm.Resource, error) {
			cli := ctx.Value(apis.ContextKeyClient).(client.Client)
			obj := &databasev1.RedisMigration{}
			err := cli.Get(ctx, pm.ParseNamespacedName(key), obj)
			return obj, err
		},
		PersistStatusFunc: func(ctx context.Context, res pm.Resource) error {
			cli := ctx.Value(apis.ContextKeyClient).(client.Client)
			obj := res.(*databasev1.RedisMigration)
			return cli.Status().Update(ctx, obj)
		},
		SetPhaseFunc: func(res pm.Resource, phase pm.Phase) {
			obj := res.(*databasev1.RedisMigration)
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
				obj := res.(*databasev1.RedisMigration)
				fmt.Printf("准备为迁移%s初始化源和目标信息\n", obj.Name)

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
				obj := res.(*databasev1.RedisMigration)
				if obj.Status.SourceStatus == nil || obj.Status.DestStatus == nil {
					reason := "源或者目标信息缺失，迁移失败"
					fmt.Printf(reason + "\n")
					rstate.Error = errors.New(reason)
					return
				}
				fmt.Printf("正在迁移Redis实例……\n")
				time.Sleep(time.Second * 30)
				fmt.Printf("Redis实例%s成功迁移至%s\n", obj.Status.SourceStatus["name"], obj.Status.DestStatus["name"])

				rstate.Done = true
				return
			}, pm.Metadata{}),
		},
	}
	def.Normalize()
	return def
}
