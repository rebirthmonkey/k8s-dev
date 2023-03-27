package workflow

import (
	"fmt"

	"github.com/golang-collections/collections/queue"
	"github.com/golang-collections/collections/set"
	"github.com/goombaio/dag"
)

type Engine struct {
	Template *Template
	Phase    string
	Entries  *queue.Queue
	EntrySet *set.Set
}

func NewEngine(t *Template) *Engine {

	return &Engine{
		Template: t,
		Phase:    InitPhase,
		Entries:  queue.New(),
		EntrySet: set.New(),
	}
}

func (e *Engine) Execute() error {
	switch e.Phase {
	case InitPhase:
		fmt.Println("================ InitPhase")

		sourceVertexes := e.Template.DAG.SourceVertices()
		for _, sourcev := range sourceVertexes {
			fmt.Println("the source vertex is: ", sourcev.ID)
			e.EntrySet.Insert(sourcev)
			e.Entries.Enqueue(sourcev)
		}

		e.Phase = ExecPhase
		e.Execute()
	case ExecPhase:
		fmt.Println("================ ExecPhase")

		for e.Entries.Len() != 0 {
			val := e.Entries.Dequeue()
			e.EntrySet.Remove(val)

			e.Template.Vertex2Object[val.(*dag.Vertex)].Execute()

			vx, _ := e.Template.DAG.Successors(val.(*dag.Vertex))
			for _, pv := range vx {
				if e.EntrySet.Has(pv) == false {
					e.Entries.Enqueue(pv)
					e.EntrySet.Insert(pv)
				}
			}
		}

		e.Phase = FinalPhase
		e.Execute()
	case FinalPhase:
		fmt.Println("================ FinalPhase")
	}
	return nil
}
