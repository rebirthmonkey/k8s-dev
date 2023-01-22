package pm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"cloud.tencent.com/teleport/pkg/controllers"
	"cloud.tencent.com/teleport/pkg/utils"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AutoName indicates that handler name should be automatically generated.
	// for composite root ( handler for a phase ) handler, use phase name as handler name
	// for non-root handler func, use function name as handler name, DO NOT use anonymous function
	AutoName = ""
)

// Handler represents the reconciling action for a specific phase
type Handler interface {
	// Handle the phase, State holds the last reconciling state if exists
	Handle(ctx context.Context, resource Resource, last State) ReconcileState
	// Name get handler name
	Name() string
	setName(name string)
	// Meta get handler metadata
	Meta() Metadata
	Components() []Handler
}

type handlerFunc struct {
	fn         func(ctx context.Context, resource Resource, last State) ReconcileState
	meta       Metadata
	components []Handler
}

func (h *handlerFunc) Meta() Metadata {
	return h.meta
}

func (h *handlerFunc) Components() []Handler {
	return h.components
}

func (h *handlerFunc) setName(name string) {
	h.meta.Name = name
	if h.meta.Desc == "" {
		h.meta.Desc = name
	}
}

func (h *handlerFunc) Handle(ctx context.Context, resource Resource, last State) ReconcileState {
	rstate := h.fn(ctx, resource, last)
	if rstate.Error != nil || rstate.Done {
		rstate.Updated = true
	}
	return rstate
}

func (h *handlerFunc) Name() string {
	return h.meta.Name
}

func HandlerFunc(fn func(ctx context.Context, resource Resource, last State) (rstate ReconcileState), meta Metadata) Handler {
	if meta.Name == "" {
		meta.Name = utils.GetFuncName(fn)
		if strings.Index(meta.Name, ".") > -1 {
			meta.Name = ""
		}
	}
	if meta.Desc == "" {
		meta.Desc = meta.Name
	}
	return &handlerFunc{
		fn:   fn,
		meta: meta,
	}
}

type compositeHandler struct {
	components []Handler
	parallel   bool
	meta       Metadata
}

func (c *compositeHandler) Meta() Metadata {
	return c.meta
}

func (c *compositeHandler) Components() []Handler {
	return c.components
}

func (c *compositeHandler) setName(name string) {
	c.meta.Name = name
	if c.meta.Desc == "" {
		c.meta.Desc = name
	}
}

func (c *compositeHandler) Handle(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
	logger := ctx.Value(controllers.ContextKeyLogger).(logr.Logger)
	hdlCount := len(c.components)
	if hdlCount == 0 {
		rstate.Error = errors.New("invalid composiste handler, no component defined")
		rstate.failed = true
		return
	}
	if c.parallel {
		logger.V(2).Info(fmt.Sprintf("Prepare to invoke %d handlers parallelly", hdlCount))
		return c.HandleParallelly(ctx, resource, last)
	} else {
		logger.V(2).Info(fmt.Sprintf("Prepare to invoke %d handlers serially", hdlCount))
		return c.HandleSerially(ctx, resource, last)
	}
}

func (c *compositeHandler) HandleParallelly(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
	cctx, cancel := context.WithCancel(ctx)
	canceled := false
	compCh := make(chan ReconcileState)
	waitCh := make(chan struct{})
	rstate.Done = true
	rstate.components = make(map[string]ReconcileState)
	for _, hdl := range c.components {
		name := hdl.Name()
		rstate.components[name] = ReconcileState{
			handler: name,
		}
	}
	go func() {
		var compWg sync.WaitGroup
		compWg.Add(len(c.components))
		for _, hdl := range c.components {
			go func(hdl Handler) {
				defer compWg.Done()
				startTime := metav1.Now()
				srstate := last.sub(hdl.Name())
				var s ReconcileState
				if srstate.Done {
					s.FromState(srstate)
				} else {
					s = hdl.Handle(cctx, resource, srstate)
				}
				s.handler = hdl.Name()
				s.startTime = &startTime
				endTime := metav1.Now()
				s.endTime = &endTime
				s.failed = s.Error != nil
				compCh <- s
			}(hdl)
		}
		compWg.Wait()
		close(waitCh)
	}()

	for {
		select {
		case s := <-compCh:
			rstate.components[s.handler] = s
			rstate.Updated = rstate.Updated || s.Updated
			rstate.Done = rstate.Done && s.Done
			rstate.Error = multiErrors(s.Error, rstate.Error)
			if rstate.startTime == nil {
				rstate.startTime = s.startTime
			}
			rstate.endTime = s.endTime
			rstate.failed = rstate.failed || s.failed
			if rstate.Error != nil && !canceled {
				// immediately notify all handlers in progress, so they can exit
				// as soon as possible
				cancel()
				canceled = true
			}
		case <-waitCh:
			close(compCh)
			goto END
		}
	}
END:
	return
}

func (c *compositeHandler) HandleSerially(ctx context.Context, resource Resource, last State) (rstate ReconcileState) {
	rstate.Done = true
	rstate.components = make(map[string]ReconcileState)
	for _, hdl := range c.components {
		name := hdl.Name()
		rstate.components[name] = ReconcileState{
			handler: name,
		}
	}
	rstate.startTime = last.StartTime
	for idx, hdl := range c.components {
		final := idx == len(c.components)-1
		startTime := metav1.Now()
		srstate := last.sub(hdl.Name())
		var s ReconcileState
		if srstate.Done {
			s.FromState(srstate)
		} else {
			s = hdl.Handle(ctx, resource, srstate)
		}
		s.handler = hdl.Name()
		if s.startTime == nil {
			if srstate.StartTime == nil {
				s.startTime = &startTime
			} else {
				s.startTime = srstate.StartTime
			}
		}
		if s.Done || s.Error != nil {
			if s.endTime == nil {
				endTime := metav1.Now()
				s.endTime = &endTime
			}
		}
		s.failed = s.Error != nil
		rstate.components[s.handler] = s
		rstate.Updated = rstate.Updated || s.Updated
		rstate.Done = rstate.Done && s.Done
		rstate.failed = rstate.failed || s.failed
		rstate.Error = multiErrors(s.Error, rstate.Error)
		if rstate.failed || !rstate.Done {
			if !final {
				rstate.Done = false
			}
			break
		}
	}
	return
}

func (c *compositeHandler) Name() string {
	return c.meta.Name
}

var _ Handler = &compositeHandler{}

// NewCompositeHandler creates a new composite handler
// if parallel is true, all handlers must acquire lock from context before modifying resource status
func NewCompositeHandler(components []Handler, parallel bool, meta Metadata) Handler {
	ch := &compositeHandler{
		components: components,
		parallel:   parallel,
		meta:       meta,
	}
	if ch.meta.Desc == "" {
		ch.meta.Desc = ch.meta.Name
	}
	return ch
}
