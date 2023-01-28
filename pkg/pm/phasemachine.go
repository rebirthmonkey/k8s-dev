package pm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/jinzhu/copier"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/thoas/go-funk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// Uninitialized as a convention, empty string represents uninitialized phase
	Uninitialized = Phase("")
	// Unspecified indicates that the handler doesn't set next phase explicitly
	Unspecified = Phase("")

	// default field names
	defaultStatusField = "Status"
	defaultPhaseField  = "Phase"
	defaultStateField  = "State"
)

const (
	ContextKeyClient        = "ControllersContextKeyClient"
	ContextKeyNoCacheClient = "ControllersContextKeyNoCacheClient"
	ContextKeyLogger        = "ControllersContextKeyLogger"
	ContextKeyMutex         = "ControllersContextKeyMutex"
)

// Interface PhaseMachine interfaces
type Interface interface {
	reconcile.Reconciler
	PhaseMachineDef
	GetDef() PhaseMachineDef
}

// PhaseMachineDef defines a phase machine
type PhaseMachineDef interface {
	// PrepareContext prepare context with res if necessary
	PrepareContext(ctx context.Context, res Resource) context.Context
	// FetchResource  fetch resource by key from backend storage
	FetchResource(ctx context.Context, key string) (Resource, error)
	// PersistStatus save status of the specified resource in backend storage
	PersistStatus(ctx context.Context, res Resource) error
	// GetStatus get resource status
	GetStatus(res Resource) interface{}
	// SetPause set pause status
	SetPause(res Resource, pause bool) error
	// GetPause get pause status
	GetPause(res Resource) bool
	// GetPhase get current phase
	GetPhase(res Resource) Phase
	// SetPhase set next phase
	SetPhase(res Resource, phase Phase)
	// NextPhase deduce next phase with information provided by the reconciling state
	NextPhase(rstate ReconcileState) Phase
	// GetState get current state
	GetState(res Resource, phase Phase) State
	// UpdateState invoked after a phase handler succeeded or fatally failed
	UpdateState(res Resource, phase Phase, state State)
	// IsTerminalPhase checks if the specified phase is a terminal phase
	IsTerminalPhase(phase Phase) bool
	// IsFailedPhase checks if the specified phase is a failed terminal phase
	IsFailedPhase(phase Phase) bool
	// IsFatal checks if an error is fatal
	IsFatal(err error) bool
	// HandlerFor get reconcile action for the specified phase
	HandlerFor(phase Phase) Handler
	// ForEachHandler iterates over all phases with handler attached
	ForEachHandler(func(phase Phase, handler Handler))
	// GetRequeueDefer returns defer before retrying ( on non-fatal error )
	GetRequeueDefer() time.Duration
}

// DefaultDef helper for functional programming
type DefaultDef struct {
	// ResourceType required
	ResourceType reflect.Type
	// RequeueDefer  default defer before retrying ( on non-fatal error )
	RequeueDefer time.Duration
	// PrepareContextFunc this func will exec after resource fetched, if you need some
	// context about res, this func may help you a lot
	PrepareContextFunc func(ctx context.Context, res Resource) context.Context
	// get a pointer of pause field
	GetPauseFieldFunc func(res Resource) *bool
	// FetchResourceFunc  required
	FetchResourceFunc func(ctx context.Context, key string) (Resource, error)
	// PersistStatusFunc required
	PersistStatusFunc func(ctx context.Context, res Resource) error
	StatusField       string
	// Phase getter/setter, if nil, the framework will use reflection to get/set phase
	GetPhaseFunc func(res Resource) Phase
	SetPhaseFunc func(res Resource, phase Phase)
	PausePhases  []Phase
	PhaseField   string
	// State getter/setter, if nil, the framework will use reflection to get/set state
	GetStateFunc    func(res Resource, phase Phase) (state State)
	UpdateStateFunc func(res Resource, phase Phase, state State)
	StateField      string
	// Handlers required
	Handlers       map[Phase]Handler
	IsFatalFunc    func(err error) bool
	InitialPhase   Phase
	TerminalPhases []Phase
	FailedPhases   []Phase
	// NormalTrans, ErrorTrans required
	NormalTrans, ErrorTrans map[Phase]Phase
}

func (c *DefaultDef) ForEachHandler(callback func(phase Phase, handler Handler)) {
	for phase, handler := range c.Handlers {
		callback(phase, handler)
	}
}

func (c *DefaultDef) PrepareContext(ctx context.Context, res Resource) context.Context {
	if c.PrepareContextFunc != nil {
		return c.PrepareContextFunc(ctx, res)
	}
	return ctx
}

func (c *DefaultDef) FetchResource(ctx context.Context, key string) (Resource, error) {
	return c.FetchResourceFunc(ctx, key)
}

func (c *DefaultDef) SetPause(res Resource, pause bool) error {
	if c.GetPauseFieldFunc == nil {
		return errors.New("GetPauseFieldFunc undefined")
	}
	sp := c.GetPauseFieldFunc(res)
	*sp = pause
	return nil
}

func (c *DefaultDef) GetPause(res Resource) bool {
	if c.GetPauseFieldFunc == nil {
		return false
	}
	sp := c.GetPauseFieldFunc(res)
	return *sp
}

func (c *DefaultDef) Reset(res Resource, updateMore ...func(name string, state *State)) error {
	failed := c.GetPhase(res)
	if !funk.Contains(c.FailedPhases, failed) {
		return nil
	}
	reversed := make(map[Phase]Phase)
	for from, to := range c.ErrorTrans {
		reversed[to] = from
	}
	phase, ok := reversed[failed]
	if !ok {
		return fmt.Errorf("current phase [%s] can't be resetted", phase)
	}
	state := c.GetState(res, phase)
	state.Reset(string(phase), updateMore...)
	c.UpdateState(res, phase, state)
	c.SetPhase(res, phase)
	return nil
}

func (c *DefaultDef) PersistStatus(ctx context.Context, res Resource) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return c.PersistStatusFunc(ctx, res)
	})
}

func (c *DefaultDef) GetStatus(res Resource) interface{} {
	sval := c.getStatusVal(res)
	return sval.Interface()
}

func (c *DefaultDef) getStatusVal(res Resource) reflect.Value {
	rval := reflect.ValueOf(res)
	if rval.Kind() == reflect.Ptr {
		rval = rval.Elem()
	}
	sval := rval.FieldByName(c.StatusField)
	return sval
}

func (c *DefaultDef) GetPhase(res Resource) (phase Phase) {
	if c.GetPhaseFunc == nil {
		sval := c.getStatusVal(res)
		pval := sval.FieldByName(c.PhaseField)
		if pval.Kind() == reflect.Ptr {
			pval = pval.Elem()
		}
		var p string
		if pval.IsValid() {
			p = pval.String()
		} else {
			p = ""
		}
		phase = Phase(p)
	} else {
		phase = c.GetPhaseFunc(res)
	}
	if phase == "" {
		phase = c.InitialPhase
	}
	return
}

func (c *DefaultDef) SetPhase(res Resource, phase Phase) {
	if c.SetPhaseFunc == nil {
		sval := c.getStatusVal(res)
		pval := sval.FieldByName(c.PhaseField)
		if pval.Kind() == reflect.Ptr {
			pval = pval.Elem()
		}
		pval.Set(reflect.ValueOf(phase))
	} else {
		c.SetPhaseFunc(res, phase)
	}
}

func (c *DefaultDef) GetState(res Resource, phase Phase) (state State) {
	if c.GetStateFunc != nil {
		return c.GetStateFunc(res, phase)
	} else {
		statusVal := c.getStatusVal(res)
		stateVal := statusVal.FieldByName(c.StateField)
		if stateVal.IsZero() {
			return
		} else {
			val := stateVal.MapIndex(reflect.ValueOf(phase))
			if !val.IsValid() {
				return
			} else {
				copier.Copy(&state, val.Interface())
				return
			}
		}
	}
}

func (c *DefaultDef) GetFailedReason(res Resource) string {
	var phase Phase
	for phase = range c.NormalTrans {
		state := c.GetState(res, phase)
		r := failedState(state)
		if r != "" {
			return r
		}
	}
	return failedState(c.GetState(res, c.NormalTrans[phase]))
}

func failedState(state State) string {
	for _, obj := range state.Objects {
		if obj.Error != "" {
			return obj.Error
		}
	}
	if state.Error != "" {
		return state.Error
	}
	for _, s := range state.State {
		if r := failedState(s); r != "" {
			return r
		}
	}
	return ""
}

func (c *DefaultDef) UpdateState(res Resource, phase Phase, state State) {
	if c.UpdateStateFunc != nil {
		c.UpdateStateFunc(res, phase, state)
	} else {
		statusVal := c.getStatusVal(res)
		stateVal := statusVal.FieldByName(c.StateField)
		if stateVal.IsZero() {
			stateVal = reflect.MakeMapWithSize(stateVal.Type(), 0)
		}
		val := reflect.New(stateVal.Type().Elem())
		copier.Copy(val.Interface(), state)
		stateVal.SetMapIndex(reflect.ValueOf(phase), val.Elem())
		statusVal.FieldByName(c.StateField).Set(stateVal)
	}
}

func (c *DefaultDef) NextPhase(rstate ReconcileState) Phase {
	if rstate.Done {
		if rstate.Next == Unspecified {
			// Phase handler didn't give next phase
			trans := c.NormalTrans
			if rstate.Error != nil {
				trans = c.ErrorTrans
			}
			return trans[rstate.current]
		} else {
			return rstate.Next
		}
	} else {
		// undone, reentering this phase
		return rstate.current
	}
}

func (c *DefaultDef) GetRequeueDefer() time.Duration {
	return c.RequeueDefer
}

func (c *DefaultDef) IsTerminalPhase(phase Phase) bool {
	return funk.Contains(c.TerminalPhases, phase)
}

func (c *DefaultDef) IsFailedPhase(phase Phase) bool {
	return funk.Contains(c.FailedPhases, phase)
}

func (c *DefaultDef) IsFatal(err error) bool {
	if c.IsFatalFunc == nil {
		return err != nil
	} else {
		return c.IsFatalFunc(err)
	}
}

func (c *DefaultDef) HandlerFor(phase Phase) Handler {
	return c.Handlers[phase]
}

func (c *DefaultDef) Normalize() (err error) {
	if c.StatusField == "" {
		c.StatusField = defaultStatusField
	}
	rtype := c.ResourceType
	if rtype.Kind() == reflect.Ptr {
		rtype = rtype.Elem()
	}
	rval := reflect.New(rtype).Elem()
	sval := rval.FieldByName(c.StatusField)
	if sval.Kind() == reflect.Ptr {
		sval = sval.Elem()
	}
	c.ForEachHandler(func(phase Phase, handler Handler) {
		if handler.Name() == "" {
			handler.setName(string(phase))
		}
	})
	if c.UpdateStateFunc == nil || c.GetStateFunc == nil {
		if c.StateField == "" {
			c.StateField = defaultStateField
		}
		sval = sval.FieldByName(c.StateField)
		stype := sval.Type()
		if stype.Kind() != reflect.Map {
			return errors.New(fmt.Sprintf("kind of field %s.%s must be map", c.StatusField, c.StateField))
		} else {
			ktype := stype.Key()
			if ktype.Kind() != reflect.String {
				return errors.New(fmt.Sprintf("key type of field %s.%s must be string", c.StatusField, c.StateField))
			}
			vtype := stype.Elem()
			if vtype.Kind() != reflect.Struct {
				return errors.New(fmt.Sprintf("value type of field %s.%s must be struct", c.StatusField, c.StateField))
			}
		}
	}
	if c.SetPhaseFunc == nil || c.GetPhaseFunc == nil {
		if c.PhaseField == "" {
			c.PhaseField = defaultPhaseField
		}
	}
	reversed := map[Phase]Phase{}
	for from, to := range c.ErrorTrans {
		if old, ok := reversed[to]; ok {
			return fmt.Errorf("unresettable condition: [%s] and [%s] have same failed [%s]", old, from, to)
		}
		reversed[to] = from
	}

	return
}

var _ PhaseMachineDef = &DefaultDef{}

func New(def PhaseMachineDef) Interface {
	return &phaseMachineImpl{def}
}

// phaseMachineImpl implements the core phase machine logic
type phaseMachineImpl struct {
	PhaseMachineDef
}

func (p *phaseMachineImpl) GetDef() PhaseMachineDef {
	return p.PhaseMachineDef
}

type PhaseMachineAware interface {
	SetPhaseMachine(def PhaseMachineDef)
}

func (p *phaseMachineImpl) Reconcile(ctx context.Context, request ctrl.Request) (result reconcile.Result, err error) {
	logger := ctrl.Log.WithName(request.NamespacedName.String())
	ctx = context.WithValue(ctx, ContextKeyLogger, logger)
	ctx = context.WithValue(ctx, ContextKeyMutex, &sync.Mutex{})
	var res Resource
	res, err = p.FetchResource(ctx, request.NamespacedName.String())
	if err != nil {
		logger.V(2).Error(err, "Failed to fetch resource")
		return result, client.IgnoreNotFound(err)
	}
	if p.GetPause(res) {
		logger.V(2).Info("Reconcilation paused, nothing to do")
		return result, nil
	}
	ctx = p.PhaseMachineDef.PrepareContext(ctx, res)
	phase := p.GetPhase(res)
	if p.IsTerminalPhase(phase) {
		logger.V(2).Info("Resource is in terminal phase, nothing to do")
		return result, nil
	}

	beforeHash, _ := p.hash(p.GetStatus(res))
	beforeState := p.GetState(res, phase)

	hdl := p.HandlerFor(phase)
	phahdl, phaware := hdl.(PhaseMachineAware)
	if phaware {
		phahdl.SetPhaseMachine(p)
	}
	var rstate ReconcileState
	rstate.handler = string(phase)
	if hdl == nil {
		msg := fmt.Sprintf("Handler for phase %s not found", phase)
		logger.V(2).Info(msg)
		rstate.Done = true
		rstate.Updated = false
		rstate.Error = errors.New(msg)
		rstate.failed = true
		rstate.fatal = true
	} else {
		logger.V(2).Info(fmt.Sprintf("Prepare to handle phase %s", phase))
		startTime := metav1.Now()
		rstate = hdl.Handle(ctx, res, beforeState)
		rstate.handler = hdl.Name()
		if rstate.startTime == nil {
			rstate.startTime = &startTime
		}
		if rstate.handler == "" {
			rstate.handler = string(phase)
		}
		if rstate.Error != nil {
			rstate.Done = true
			rstate.failed = true
			rstate.fatal = true
		}
		if rstate.Done || rstate.Error != nil {
			endTime := metav1.Now()
			rstate.endTime = &endTime
		}
	}
	rstate.current = phase
	return p.proceed(ctx, res, rstate, beforeHash)
}

func (p *phaseMachineImpl) proceed(ctx context.Context, res Resource, rstate ReconcileState,
	beforeHash uint64) (result reconcile.Result, err error) {
	logger := ctx.Value(ContextKeyLogger).(logr.Logger)
	failed := rstate.failed
	var fatal bool
	if failed {
		fatal = p.checkFatal(&rstate)
		rstate.Done = fatal
	}

	state := rstate.ToState()
	p.UpdateState(res, rstate.current, state)

	rstate.Next = p.NextPhase(rstate)
	p.SetPhase(res, rstate.Next)

	afterHash, _ := p.hash(p.GetStatus(res))
	if rstate.Done || beforeHash != afterHash {
		rstate.Updated = true
	}
	result.RequeueAfter = p.GetRequeueDefer()
	if rstate.Updated {
		logger.V(2).Info("Prepare to persist resource status")
		err = p.PersistStatus(ctx, res)
		if err != nil {
			// Any errors occurred here considered as non-fatal
			logger.Error(err, "Failed to persist resource status, requeue forcibly")
			return result, nil
		}
	}
	if err != nil {
		logger.Error(err, "Current phase failed", "rstate", rstate)
	}

	return result, nil
}

func (p *phaseMachineImpl) hash(val interface{}) (uint64, error) {
	data, err := json.Marshal(val)
	if err != nil {
		return 0, err
	}
	return hashstructure.Hash(data, hashstructure.FormatV2, &hashstructure.HashOptions{
		ZeroNil:         true,
		IgnoreZeroValue: true,
		SlicesAsSets:    false,
		UseStringer:     false,
	})
}

func (p *phaseMachineImpl) checkFatal(rstate *ReconcileState) bool {
	if funk.IsEmpty(rstate.components) {
		rstate.fatal = rstate.Error != nil && (p.isNonFatalErrorPtr(rstate.Error) || p.IsFatal(rstate.Error))
	} else {
		for hdl, crstate := range rstate.components {
			crstate := crstate
			rstate.fatal = rstate.fatal || p.checkFatal(&crstate)
			rstate.components[hdl] = crstate
		}
	}
	return rstate.fatal
}

func (p *phaseMachineImpl) isNonFatalErrorPtr(err error) bool {
	return reflect.TypeOf(err).AssignableTo(reflect.TypeOf(&nonFatalError{}))
}

func RecommandedRequeueDefer() time.Duration {
	return time.Second * 5
}
