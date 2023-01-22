package pm

import (
	"bytes"
	"cloud.tencent.com/teleport/pkg/controllers"
	"cloud.tencent.com/teleport/pkg/utils"
	"context"
	"encoding/json"
	"fmt"
	"github.com/antchfx/jsonquery"
	"github.com/go-logr/logr"
	"strings"
)

type objectsHandlerFunc struct {
	fn         func(ctx context.Context, objects []Object) (handled []Object, err error)
	meta       Metadata
	components []Handler
	pmdef      PhaseMachineDef
}

func (h *objectsHandlerFunc) SetPhaseMachine(def PhaseMachineDef) {
	h.pmdef = def
}

func (h *objectsHandlerFunc) Meta() Metadata {
	return h.meta
}

func (h *objectsHandlerFunc) Components() []Handler {
	return h.components
}

func (h *objectsHandlerFunc) setName(name string) {
	h.meta.Name = name
	if h.meta.Desc == "" {
		h.meta.Desc = name
	}
}

func (h *objectsHandlerFunc) Handle(ctx context.Context, resource Resource, last State) (state ReconcileState) {
	logger := ctx.Value(controllers.ContextKeyLogger).(logr.Logger).WithValues("handler", h.Name())
	if len(last.Objects) == 0 {
		logger.V(2).Info("Objects not initialized yet, prepare to invoke initializer")
		var err error
		if last.Objects, err = h.initializeObjects(resource); err != nil {
			state.Error = err
			return
		}
		state.Updated = true
	}
	objs := h.batch(last)
	if len(objs) == 0 {
		logger.V(2).Info("No object need to be handled")
		state.Done = true
		state.Objects = last.Objects
		var warnObjs []string
		var errObjs []string
		for _, obj := range last.Objects {
			if obj.Warning != "" {
				warnObjs = append(warnObjs, obj.Id)
			}
			if obj.Error != "" {
				errObjs = append(errObjs, obj.Id)
			}
		}
		errObjsLen := len(errObjs)
		warnObjsLen := len(warnObjs)
		if state.Error == nil && errObjsLen > 0 {
			state.Error = fmt.Errorf("errors occurred on objects")
		} else if warnObjsLen > 0 {
			state.Warning = fmt.Sprintf("warnings occurred on objects")
		}
		return
	}
	handled, err := h.fn(ctx, objs)
	state.Error = err
	var changed bool
	state.Objects, changed = h.syncResult(handled, &last)
	if !state.Updated {
		state.Updated = state.Error != nil || changed
	}
	return
}

func (h *objectsHandlerFunc) syncResult(handled []Object, last *State) (res []Object, changed bool) {
	res = last.Objects
	for _, obj := range handled {
		for i := range res {
			if res[i].Id == obj.Id {
				res[i] = obj
				if !changed {
					changed = obj.Done || obj.Error != ""
				}
				continue
			}
		}
	}
	return
}

func (h *objectsHandlerFunc) Name() string {
	return h.meta.Name
}

func (h *objectsHandlerFunc) initializeObjects(res Resource) ([]Object, error) {
	return h.meta.ObjectsInitializer(res)
}

// fetch a batch of undone objects from state, these objects:
//   must be in the same Group
//   objects count must not be > h.meta.BatchSize, if no grouping ( Group field of any object is empty )
//     h.meta.BatchSize == 0 means no batching, return all the elements of same group
func (h *objectsHandlerFunc) batch(state State) []Object {
	group := ""
	var ret []Object
	for _, obj := range state.Objects {
		if obj.Done {
			continue
		}
		if group == "" {
			group = obj.Group
		}
		if group == obj.Group {
			ret = append(ret, obj)
		}
		grouping := group != ""
		// a group of objects should always be handled in one batch
		//               zeron value means no batch size limit
		if !grouping && (h.meta.BatchSize != 0 && len(ret) == h.meta.BatchSize) {
			break
		}
	}
	return ret
}

// ObjectsHandlerFunc wraps fn as a handler
// parameters of fn:
//   objects: a batch of objects which the handler should handle, safe to modify
// return values of fn:
//   handled: subset of objects, updated, will be merged into handler state by the framework. could be nil if did nothing
//   err: indicates that a global error ( not specific to an object ) occurred
func ObjectsHandlerFunc(fn func(ctx context.Context, objects []Object) (handled []Object, err error), meta Metadata) Handler {
	if meta.Name == "" {
		meta.Name = utils.GetFuncName(fn)
		if strings.Index(meta.Name, ".") > -1 {
			meta.Name = ""
		}
	}
	if meta.Desc == "" {
		meta.Desc = meta.Name
	}
	return &objectsHandlerFunc{
		fn:   fn,
		meta: meta,
	}
}

func PathObjectsInitializer(objsPath, idPath, namePath, groupPath string) func(res Resource) (objs []Object, err error) {
	return func(res Resource) (objs []Object, err error) {
		var resJson []byte
		resJson, err = json.Marshal(res)
		var doc *jsonquery.Node
		doc, err = jsonquery.Parse(bytes.NewBuffer(resJson))
		for _, objNode := range jsonquery.Find(doc, objsPath) {
			obj := Object{}
			if idPath == "" {
				obj.Id = objNode.Data
			} else {
				idNode := jsonquery.FindOne(objNode, idPath)
				if idNode == nil {
					return nil, fmt.Errorf("id not found from node %s", objNode.Data)
				}
				obj.Id = idNode.Data
			}
			if namePath != "" {
				nameNode := jsonquery.FindOne(objNode, namePath)
				if nameNode != nil {
					obj.Name = nameNode.Data
				}
			}
			if groupPath != "" {
				groupNode := jsonquery.FindOne(objNode, groupPath)
				if groupNode != nil {
					obj.Group = groupNode.Data
				}
			}
			objs = append(objs, obj)
		}
		return objs, nil
	}
}
