package pm

const (
	Unlimited = int(^uint(0) >> 1)
)

type Metadata struct {
	// name of the handler
	Name string
	/// description of the handler
	Desc string
	// Objects is an optional feature of PhaseMachine framework
	// initializer of State.Objects of the handler.
	// Only id is mandatory, provide Group field if you want to enable grouping
	ObjectsInitializer func(res Resource) (objs []Object, err error)
	// max batch size for Objects handling, meanful grouping disabled
	BatchSize int

	// fields for view only
	// root handler only fields
	View ViewType
	// Deprecated: with Objects feature, the framework will provide a reasonable default
	ObjectDetailsFunc func(res Resource) ObjectDetails
	ProgressFunc      func(res Resource) Progress
}
