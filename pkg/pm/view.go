package pm

import (
	"github.com/thoas/go-funk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ViewType string

const (
	ViewTypeNone         = "none"
	ViewTypeCheckList    = "checklist"
	ViewTypeObjectDetail = "objectdetail"
	ViewTypeProgress     = "progress"
)

// StatusView frontend-oriented interface for rendering PhaseMachine status
type StatusView interface {
	// PhasesSummary phase summary for normal transitions
	PhasesSummary() []PhaseSummary
	PhaseDetail(phase Phase) PhaseDetail
}

func CreateStatusView(pmdef *DefaultDef, res Resource) StatusView {
	return &statusView{
		pmdef: pmdef,
		res:   res,
	}
}

type statusView struct {
	pmdef *DefaultDef
	res   Resource
}

type PhaseSummary struct {
	Phase     Phase        `json:"phase"`
	Desc      string       `json:"desc"`
	Result    Result       `json:"result"`
	Timestamp *metav1.Time `json:"timestamp"`
}

type PhaseDetail struct {
	View ViewType    `json:"view"`
	Data interface{} `json:"data"`
}

// CheckList data structure for view type checklist
type CheckList struct {
	Result    Result       `json:"result"`
	Desc      string       `json:"desc"`
	Timestamp *metav1.Time `json:"timestamp"`
	Checks    []CheckList  `json:"checks"`
}

// ObjectDetails data structure for view type objectdetail
// from object type ( e.g.  eni,ins,eip... ) to ObjectDetail
type ObjectDetails map[string][]ObjectDetail

// ObjectDetail result of each handler on a specific object
type ObjectDetail struct {
	// object identifier
	Id string `json:"id"`
	// object name
	Name string `json:"name"`
	// from handler desc to result
	Results map[string]ObjectHandleResult `json:"results"`
}
type ObjectHandleResult struct {
	Result    Result       `json:"result"`
	Timestamp *metav1.Time `json:"timestamp"`
	RequestId string       `json:"requestId"`
}

// Progress data structure for view type progress
type Progress struct {
	DoneCount  uint64 `json:"doneCount"`
	TotalCount uint64 `json:"totalCount"`
	Percent    uint8  `json:"percent"`
}

func (s *statusView) PhasesSummary() []PhaseSummary {
	def := s.pmdef
	phase := def.InitialPhase
	pvs := make([]PhaseSummary, 0)
	appendPhase := func(phase Phase) {
		if !funk.Contains(def.TerminalPhases, phase) {
			pvs = append(pvs, PhaseSummary{
				Phase:  phase,
				Desc:   def.HandlerFor(phase).Meta().Desc,
				Result: def.GetState(s.res, phase).Result(),
			})
			return
		}
		doneRes := doneResultOf(pvs)
		pvs = append(pvs, PhaseSummary{
			Phase: Phase("Done"),
			Desc:  "完成",
			Result: func(phase Phase) Result {
				for _, v := range s.pmdef.ErrorTrans {
					if v == phase {
						return ResultFailed
					}
				}
				return doneRes
			}(phase),
		})
	}
	appendPhase(phase)
	for {
		var ok bool
		phase, ok = def.NormalTrans[phase]
		if !ok {
			break
		}
		appendPhase(phase)
	}
	return pvs
}

func (s *statusView) PhaseDetail(phase Phase) (pd PhaseDetail) {
	handler := s.pmdef.HandlerFor(phase)
	if handler == nil {
		return PhaseDetail{
			View: ViewTypeNone,
			Data: nil,
		}
	}
	meta := handler.Meta()
	pd.View = meta.View
	switch pd.View {
	case ViewTypeCheckList:
		pd.Data = s.checkList(s.res, phase)
	case ViewTypeObjectDetail:
		objectDetailsFunc := meta.ObjectDetailsFunc
		if objectDetailsFunc != nil {
			pd.Data = objectDetailsFunc(s.res)
		} else {
			// assume that Objects feature enabled
			pd.Data = s.objectsDetail(s.res, phase)
		}
	case ViewTypeProgress:
		pd.Data = meta.ProgressFunc(s.res)
	case ViewTypeNone, "":
	default:
	}
	return
}

func (s *statusView) checkList(res Resource, phase Phase) CheckList {
	state := s.pmdef.GetState(res, phase)
	def := s.pmdef
	hdl := def.HandlerFor(phase)
	desc := hdl.Meta().Desc
	hdls := hdl.Components()
	return s.checkListFromState(state, hdls, desc)
}

func (s *statusView) checkListFromState(state State, hdls []Handler, desc string) CheckList {
	list := CheckList{
		Result:    resultFromState(state),
		Timestamp: state.StartTime,
		Desc:      desc,
	}
	for _, hdl := range hdls {
		for k, v := range state.State {
			if hdl.Meta().Name == k {
				newdesc := hdl.Meta().Desc
				list.Checks = append(list.Checks, s.checkListFromState(v, hdl.Components(), newdesc))
			}
		}
	}
	return list
}

func (s *statusView) objectsDetail(res Resource, phase Phase) ObjectDetails {
	// traverse state tree of the phase, get all objects, convert to ObjectDetails
	// TODO oliveryang
	return nil
}

func resultFromState(s State) Result {
	if s.Done && s.Error != "" {
		return ResultFailed
	} else if s.Done && s.Error == "" {
		return ResultSucceeded
	} else if s.StartTime != nil {
		return ResultDoing
	} else {
		return ResultPending
	}
}

func doneResultOf(pvs []PhaseSummary) Result {
	var phases []Result
	for _, pv := range pvs {
		phases = append(phases, pv.Result)
	}
	if funk.Contains(phases, ResultFailed) {
		return ResultFailed
	} else if funk.Contains(phases, ResultPending) {
		return ResultPending
	} else if funk.Contains(phases, ResultDoing) {
		return ResultDoing
	} else {
		return ResultSucceeded
	}
}
