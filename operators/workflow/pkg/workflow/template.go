package workflow

import (
	"fmt"

	"github.com/goombaio/dag"
)

type Template struct {
	DAG *dag.DAG

	Objects       map[string]Object
	Object2Vertex map[Object]*dag.Vertex
	Vertex2Object map[*dag.Vertex]Object
}

func NewTemplate() *Template {

	return &Template{
		DAG: dag.NewDAG(),

		Objects:       make(map[string]Object),
		Object2Vertex: make(map[Object]*dag.Vertex),
		Vertex2Object: make(map[*dag.Vertex]Object),
	}
}

func (t *Template) AddVertex(name string, obj interface{}) {
	t.Objects[name] = obj.(Object)
	vertex := dag.NewVertex(obj.(Object).GetName(), obj)
	t.Object2Vertex[obj.(Object)] = vertex
	t.Vertex2Object[vertex] = obj.(Object)
	err := t.DAG.AddVertex(vertex)
	if err != nil {
		fmt.Println(err)
	}
}

func (t *Template) AddEdge(obj1Name, obj2Name string) {
	err := t.DAG.AddEdge(t.Object2Vertex[t.Objects[obj1Name].(Object)], t.Object2Vertex[t.Objects[obj2Name].(Object)])
	if err != nil {
		fmt.Println(err)
	}
}

type Object interface {
	GetName() string
	Execute() error
}
