package pm

import (
	"cloud.tencent.com/teleport/pkg/controllers"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

func TestPhaseMachine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "phaseMachineImpl Suite")
}

type TestMigration struct {
	Status TestMigrationStatus `json:"status"`
}

type TestVpc struct {
	Id      string                `json:"id,omitempty"`
	Subnets map[string]TestSubnet `json:"subnets,omitempty"`
}
type TestSubnet struct {
	Name      string                  `json:"name,omitempty"`
	Instances map[string]TestInstance `json:"instances,omitempty"`
}
type TestInstance struct {
	Disks []string `json:"disks,omitempty"`
}

type TestMigrationState struct {
	Done      bool                          `json:"done,omitempty"`
	Error     string                        `json:"error,omitempty"`
	StartTime *metav1.Time                  `json:"startTime,omitempty"`
	EndTime   *metav1.Time                  `json:"endTime,omitempty"`
	State     map[string]TestMigrationState `json:"state,omitempty"`
	Fatal     bool                          `json:"fatal,omitempty"`
	Failed    bool                          `json:"failed,omitempty"`
	Objects   []Object                      `json:"objects,omitempty"`
}

type TestMigrationStatus struct {
	Phase Phase                        `json:"phase,omitempty"`
	State map[Phase]TestMigrationState `json:"state,omitempty"`
	Vpc   TestVpc                      `json:"vpc"`
}

var (
	Resources map[string]*TestMigration
)

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(
		zap.WriteTo(GinkgoWriter),
		zap.UseDevMode(true),
		zap.Level(zapcore.Level(-5)),
	))
	Resources = make(map[string]*TestMigration)
})

var _ = AfterSuite(func() {
})

func reconcileAndWait(machine Interface, key string) {
	doneCh := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-ctx.Done():
				goto END
			default:
				res, err := machine.FetchResource(ctx, key)
				Expect(err).ShouldNot(HaveOccurred())
				phase := machine.GetPhase(res)
				if machine.IsTerminalPhase(phase) {
					goto END
				}

				machine.Reconcile(ctx, reconcile.Request{
					NamespacedName: ParseNamespacedName(key),
				})
			}
		}
	END:
		doneCh <- struct{}{}
	}()
	select {
	case <-doneCh:
		cancel()
	case <-time.After(time.Minute):
		Fail("Phasemachine execution timedout")
	}
}

var _ = Describe("phaseMachineImpl", func() {
	const (
		initialize = Phase("初始化")
		precheck   = Phase("资源预检")
		migrate    = Phase("资源迁移")
		success    = Phase("迁移成功")
		prefailed  = Phase("预检失败")
		failed     = Phase("迁移失败")
	)
	def := &DefaultDef{
		ResourceType: reflect.TypeOf(&TestMigration{}),
		RequeueDefer: time.Second * 5,
		FetchResourceFunc: func(ctx context.Context, key string) (Resource, error) {
			return Resources[key], nil
		},
		PersistStatusFunc: func(ctx context.Context, res Resource) error {
			return nil
		},
		IsFatalFunc: func(err error) bool {
			return !reflect.TypeOf(err).AssignableTo(reflect.TypeOf(nonFatalError{}))
		},
		InitialPhase: initialize,
		TerminalPhases: []Phase{
			prefailed, failed, success,
		},
		Handlers: map[Phase]Handler{
			initialize: HandlerFunc(func(ctx context.Context, res Resource, last State) (rstate ReconcileState) {
				logger := ctx.Value(controllers.ContextKeyLogger).(logr.Logger)
				rstate.Done = true
				logger.Info("resource initialization succeeded")
				return
			}, Metadata{}),
		},
		NormalTrans: map[Phase]Phase{
			initialize: precheck,
			precheck:   migrate,
			migrate:    success,
		},
		ErrorTrans: map[Phase]Phase{
			initialize: prefailed,
			precheck:   prefailed,
			migrate:    failed,
		},
	}
	err := def.Normalize()
	Expect(err).ShouldNot(HaveOccurred())
	machine := New(def)
	Context("with no handler for precheck", func() {
		It("should be terminated in prefailed", func() {
			key := "default/precheck-nohandler"
			m := &TestMigration{}
			Resources[key] = m
			reconcileAndWait(machine, key)
			Expect(m.Status.Phase).Should(BeEquivalentTo(prefailed))
			Expect(m.Status.State[initialize].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[initialize].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.State[precheck].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[precheck].Failed).Should(BeEquivalentTo(true))
			Expect(m.Status.State[precheck].Fatal).Should(BeEquivalentTo(true))
		})
	})
	Context("with composite handler for precheck without component", func() {
		It("should be terminated in prefailed", func() {
			key := "default/precheck-nocomponent"
			m := &TestMigration{}
			Resources[key] = m
			def := machine.GetDef().(*DefaultDef)
			def.Handlers[precheck] = NewCompositeHandler([]Handler{}, false, Metadata{})
			reconcileAndWait(machine, key)
			Expect(m.Status.Phase).Should(BeEquivalentTo(prefailed))
			Expect(m.Status.State[precheck].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[precheck].Failed).Should(BeEquivalentTo(true))
			Expect(m.Status.State[precheck].Fatal).Should(BeEquivalentTo(true))
			Expect(m.Status.State[precheck].Error).Should(ContainSubstring("invalid composiste handler"))
		})
	})
	Context("with composite handler for precheck execute parallelly", func() {
		It("should pass precheck", func() {
			key := "default/precheck-parallel"
			m := &TestMigration{}
			Resources[key] = m
			def := machine.GetDef().(*DefaultDef)
			def.Handlers[precheck] = NewCompositeHandler([]Handler{
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					time.Sleep(time.Second)
					rstate.Done = true
					return
				}, Metadata{Name: "实例预检"}),
				NewCompositeHandler([]Handler{
					HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
						time.Sleep(time.Second)
						rstate.Done = true
						return
					}, Metadata{Name: "私有网络预检"}),
					HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
						time.Sleep(time.Second)
						rstate.Done = true
						return
					}, Metadata{Name: "子网预检"}),
				}, true, Metadata{Name: "网络预检"}),
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					time.Sleep(time.Second)
					rstate.Done = true
					return
				}, Metadata{Name: "存储预检"}),
			}, true, Metadata{})
			reconcileAndWait(machine, key)
			Expect(m.Status.State[initialize].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[initialize].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.State[precheck].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[precheck].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.Phase).Should(BeEquivalentTo(failed))
			bytes, _ := yaml.Marshal(m)
			println(string(bytes))
		})
	})
	Context("with composite handler for migrate without component", func() {
		It("should be terminated in failed", func() {
			key := "default/migrate-nohandler"
			m := &TestMigration{}
			Resources[key] = m
			def := machine.GetDef().(*DefaultDef)
			def.Handlers[migrate] = NewCompositeHandler([]Handler{}, false, Metadata{})
			reconcileAndWait(machine, key)
			Expect(m.Status.Phase).Should(BeEquivalentTo(failed))
			Expect(m.Status.State[initialize].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[initialize].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.State[precheck].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[precheck].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.Phase).Should(BeEquivalentTo(failed))
		})
	})
	Context("with composite handler for precheck execute serially", func() {
		It("should succeed", func() {
			key := "default/normal-codepath"
			m := &TestMigration{}
			Resources[key] = m
			def := machine.GetDef().(*DefaultDef)
			def.Handlers[migrate] = NewCompositeHandler([]Handler{
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					rstate.Done = true
					return
				}, Metadata{Name: "实例迁移"}),
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					rstate.Done = true
					return
				}, Metadata{Name: "存储迁移"}),
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					rstate.Done = true
					return
				}, Metadata{Name: "网络迁移"}),
			}, false, Metadata{Name: "资源迁移"})
			reconcileAndWait(machine, key)
			Expect(m.Status.State[initialize].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[initialize].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.State[precheck].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[precheck].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.State[migrate].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[migrate].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.Phase).Should(BeEquivalentTo(success))
			bytes, _ := yaml.Marshal(m)
			println(string(bytes))
		})
		It("should fail if any component returns error", func() {
			key := "default/migrate-storage-fail"
			const storageMsg = "failed to migarate storage"
			m := &TestMigration{}
			Resources[key] = m
			def := machine.GetDef().(*DefaultDef)
			def.Handlers[migrate] = NewCompositeHandler([]Handler{
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					rstate.Done = true
					return
				}, Metadata{Name: "实例迁移"}),
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					rstate.Error = errors.New(storageMsg)
					return
				}, Metadata{Name: "存储迁移"}),
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					rstate.Done = true
					return
				}, Metadata{Name: "网络迁移"}),
			}, false, Metadata{Name: "资源迁移"})
			reconcileAndWait(machine, key)
			Expect(m.Status.State[migrate].Failed).Should(BeEquivalentTo(true))
			Expect(m.Status.State[migrate].State["实例迁移"].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.State[migrate].State["存储迁移"].Failed).Should(BeEquivalentTo(true))
			Expect(m.Status.State[migrate].State["存储迁移"].Fatal).Should(BeEquivalentTo(true))
			Expect(m.Status.State[migrate].State["网络迁移"].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.State[migrate].State["网络迁移"].Done).Should(BeEquivalentTo(false))
			Expect(m.Status.State[migrate].State["网络迁移"].StartTime).Should(BeNil())
		})
		It("should get noticed with last reconciliation state", func() {
			key := "default/migrate-storage-nonfatal"
			const storageMsg = "failed to migarate storage"
			m := &TestMigration{}
			Resources[key] = m
			def := machine.GetDef().(*DefaultDef)
			def.Handlers[migrate] = NewCompositeHandler([]Handler{
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					Expect(last.Done).Should(BeFalse())
					rstate.Done = true
					return
				}, Metadata{Name: "容器迁移"}),
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					if last.StartTime == nil {
						rstate.Done = false
					} else {
						rstate.Done = true
					}
					return
				}, Metadata{Name: "实例迁移"}),
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					if last.Failed {
						Expect(last.Fatal).Should(BeEquivalentTo(false))
						rstate.Error = errors.New(storageMsg)
					} else {
						rstate.Error = &nonFatalError{s: "connection closed"}
					}
					rstate.Updated = true
					return
				}, Metadata{Name: "存储迁移"}),
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					rstate.Done = true
					return
				}, Metadata{Name: "网络迁移"}),
			}, false, Metadata{Name: "资源迁移"})
			reconcileAndWait(machine, key)
			Expect(m.Status.State[migrate].Failed).Should(BeEquivalentTo(true))
			Expect(m.Status.State[migrate].State["容器迁移"].Done).Should(BeEquivalentTo(true))
			Expect(m.Status.State[migrate].State["容器迁移"].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.State[migrate].State["实例迁移"].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.State[migrate].State["存储迁移"].Failed).Should(BeEquivalentTo(true))
			Expect(m.Status.State[migrate].State["存储迁移"].Fatal).Should(BeEquivalentTo(true))
			Expect(m.Status.State[migrate].State["网络迁移"].Failed).Should(BeEquivalentTo(false))
			Expect(m.Status.State[migrate].State["网络迁移"].Done).Should(BeEquivalentTo(false))
			Expect(m.Status.State[migrate].State["网络迁移"].StartTime).Should(BeNil())
		})
	})
	Context("with composite handler for migrate with objects feature enabled", func() {
		It("should be terminated in failed", func() {
			key := "default/migrate-objects"
			m := &TestMigration{}
			Resources[key] = m
			def := machine.GetDef().(*DefaultDef)
			def.Handlers[initialize] = HandlerFunc(func(ctx context.Context, res Resource, last State) (rstate ReconcileState) {
				logger := ctx.Value(controllers.ContextKeyLogger).(logr.Logger)
				rstate.Done = true
				m := res.(*TestMigration)
				m.Status.Vpc = TestVpc{
					Id: "TestVpc-1000",
					Subnets: map[string]TestSubnet{
						"TestSubnet-1100": {
							Name: "测试子网一",
							Instances: map[string]TestInstance{
								"cvm-1110": {
									Disks: []string{"cbs-1111", "cbs-1112"},
								},
								"cvm-1120": {
									Disks: []string{"cbs-1121", "cbs-1122"},
								},
							},
						},
						"TestSubnet-1200": {
							Name: "测试子网二",
							Instances: map[string]TestInstance{
								"cvm-1210": {
									Disks: []string{"cbs-1211", "cbs-1212"},
								},
							},
						},
					},
				}
				mjson, _ := json.MarshalIndent(m, "", "  ")
				logger.Info(fmt.Sprintf("Migration with objects \n%s", mjson))
				return
			}, Metadata{})
			def.Handlers[migrate] = NewCompositeHandler([]Handler{
				ObjectsHandlerFunc(func(ctx context.Context, objects []Object) (handled []Object, err error) {
					for idx := range objects {
						now := metav1.Now()
						objects[idx].StartTime = &now
						objects[idx].EndTime = &now
						objects[idx].Done = true
					}
					handled = objects
					return
				}, Metadata{
					Name: "子网迁移",
					ObjectsInitializer: PathObjectsInitializer(
						"/status/vpc/subnets/*",
						// idPath, namePath, groupPath are relative to current object node
						".", "./name/text()", "../../id/text()"),
				}),
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					rstate.Done = true
					return
				}, Metadata{Name: "实例迁移"}),
				HandlerFunc(func(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
					rstate.Done = true
					return
				}, Metadata{Name: "存储迁移"}),
			}, false, Metadata{Name: "资源迁移"})

			reconcileAndWait(machine, key)
		})
	})
})
