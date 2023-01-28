package pm

import (
	"errors"

	"github.com/rebirthmonkey/go/pkg/util"
	"github.com/thoas/go-funk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Resource API resource, declares the desired state of TestMigration ops
type Resource interface {
}

// Result reconciling result of a phase, handler or object
type Result string

const (
	// ResultPending  pending / unknown
	ResultPending = Result("Pending")
	// ResultDoing  doing
	ResultDoing = Result("Doing")
	// ResultSucceeded succeeded
	ResultSucceeded = Result("Succeeded")
	// ResultWarning succeeded with warning
	ResultWarning = Result("Warning")
	// ResultFailed failed
	ResultFailed = Result("Failed")
)

// Phase during reconciliation, a resource will traverse a series of phases, and
// finally reach a terminal phase, which stops the phase machine
type Phase string

// State phase handler ( or a component handler ) statistics, readonly to framework users
type State struct {
	// Done indicates if the handler logic is done ( irrespective of the result )
	Done bool
	// when the handler started, a nil value indicates the handler never start
	StartTime *metav1.Time
	// when the handler ended
	EndTime *metav1.Time
	// states of component handlers if exist
	State map[string]State
	// result of the handler, failed or not
	Failed bool
	// if Failed is false and this field is not empty, means that the handler
	// succeeded with a warning should be noticed by the user
	// Mainly used in prechecking
	Warning string
	// if Failed is true, this field contains human-readable short message
	Error string
	// if Failed is true, this field indicates whether if error is fatal.
	// Fatal error causes the reconciliation to terminiate immediately
	Fatal bool
	// Objects is an optional feature of PhaseMachine framework
	// status of all objects the handler should handle
	// this field shoule be initialized by ObjectsInitializer of the handler
	Objects []Object
}

// get component handler State
func (s State) sub(hdl string) State {
	if funk.IsEmpty(s.State) {
		return State{}
	} else {
		return s.State[hdl]
	}
}

func (s State) Result() Result {
	switch {
	case s.Done:
		switch {
		case s.Fatal || s.Failed:
			return ResultFailed
		case s.Warning != "":
			return ResultWarning
		default:
			return ResultSucceeded
		}
	case s.StartTime != nil:
		return ResultDoing
	default:
		return ResultPending
	}
}

func (state *State) Reset(name string, updateMore ...func(name string, state *State)) {
	if state.Error != "" || state.Fatal || state.Failed {
		if len(updateMore) > 0 {
			updateMore[0](name, state)
		}
		state.Error = ""
		state.Fatal = false
		state.Done = false
		state.EndTime = nil
		state.Failed = false
		for k, obj := range state.Objects {
			if obj.Error == "" {
				continue
			}
			obj.EndTime = nil
			obj.HandleId = ""
			obj.Error = ""
			obj.Warning = ""
			obj.Done = false
			state.Objects[k] = obj
		}
	}
	for k, s := range state.State {
		s.Reset(name+"."+k, updateMore...)
		state.State[k] = s
	}
}

// Object Handling status of an object
type Object struct {
	// identifier of the object, object type implied. for example, cvm-xdw736fg1
	Id string `json:"id"`
	// name of the object
	Name string `json:"name,omitempty"`
	// grouping identifier of the object. a group of objects
	// will be handled in a single batch, they may share single HandleId.
	// grouping should be all or nothing, i.e. all the objects should have or not have a grouping identifier
	Group string `json:"group,omitempty"`
	// opaque string for identifying handling operation(s) of the object
	// usually this field stores request id(s) related to the object
	HandleId string `json:"handleId,omitempty"`
	// indicates if handling of the object is done
	Done bool `json:"done"`
	// when handling of the object started.
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// when handling of the object ended.
	EndTime *metav1.Time `json:"endTime,omitempty"`
	// if handling of the object succeeded with warning
	Warning string `json:"warning,omitempty"`
	// if handling of the object is failed
	Error string `json:"error,omitempty"`
}

func (o Object) Result() Result {
	switch {
	case o.Done:
		switch {
		case o.Error != "":
			return ResultFailed
		case o.Warning != "":
			return ResultWarning
		default:
			return ResultSucceeded
		}
	case o.StartTime != nil:
		return ResultDoing
	default:
		return ResultPending
	}
}

// ReconcileState phase handler ( or a component handler ) state, writeable to framework users
type ReconcileState struct {
	// Done indicates if the handler logic is done ( irrespective of the result )
	Done bool
	// If status of the resourace has been modified ( in memory only )
	Updated bool
	// Whether to issue a warning message, only meanfull when no Error happened
	Warning string
	// If error occurred during the reconciliation
	Error error
	// Next phase, for jumping to any Phase, irrespective of trans matrix
	Next Phase
	// Objects is an optional feature of PhaseMachine framework
	// objects impacted objects of this reconciliation ( handler invocation )
	// should be merged into State.Objects by the framework
	Objects []Object

	// startTime when the handler started
	startTime *metav1.Time
	// endTime when the handler ended
	endTime *metav1.Time
	// handler name of the handler
	handler string
	// current phase
	current Phase
	// failed if the handler failed ( error is not nil )
	failed bool
	// fatal if the error is fatal
	fatal bool
	// components for composite handlers, this field contains ReconcileState of each component handler
	components map[string]ReconcileState
}

// ToState converts State field of the api resource into internal State struct
func (rstate ReconcileState) ToState() (state State) {
	composite := util.IsNotEmpty(rstate.components)
	state.StartTime = rstate.startTime
	state.EndTime = rstate.endTime
	state.Done = rstate.Done
	state.Failed = rstate.failed
	state.Fatal = rstate.fatal
	state.Objects = rstate.Objects
	state.Warning = rstate.Warning
	state.Error = func() string {
		if composite || rstate.Error == nil {
			return ""
		} else {
			err := rstate.Error.Error()
			return err
		}
	}()
	if composite {
		state.State = make(map[string]State)
		for h, s := range rstate.components {
			state.State[h] = s.ToState()
		}
	}
	return
}

func (rstate *ReconcileState) FromState(state State) {
	composite := util.IsNotEmpty(state.State)
	rstate.startTime = state.StartTime
	rstate.endTime = state.EndTime
	rstate.Done = state.Done
	rstate.failed = state.Failed
	rstate.fatal = state.Fatal
	rstate.Objects = state.Objects
	rstate.Warning = state.Warning
	rstate.Error = func() error {
		if rstate.failed {
			if rstate.fatal {
				return errors.New(state.Error)
			} else {
				return &nonFatalError{state.Error}
			}
		} else {
			return nil
		}
	}()
	if composite {
		rstate.components = make(map[string]ReconcileState)
		for h, s := range state.State {
			var srsate ReconcileState
			srsate.FromState(s)
			rstate.components[h] = srsate
		}
	}
	return
}
