package reconcilermgr

type ReconcilerBuilderList []func(*ReconcilerManager) error

func (rb *ReconcilerBuilderList) Register(funcs ...func(*ReconcilerManager) error) {
	for _, f := range funcs {
		*rb = append(*rb, f)
	}
}

func (rb *ReconcilerBuilderList) AddToManager(rmgr *ReconcilerManager) error {
	for _, f := range *rb {
		if err := f(rmgr); err != nil {
			return err
		}
	}
	return nil
}

var (
	ReconcilerBuilders ReconcilerBuilderList
)

func Register(funcs ...func(*ReconcilerManager) error) {
	ReconcilerBuilders.Register(funcs...)
}

func AddToManager(rmgr *ReconcilerManager) {
	ReconcilerBuilders.AddToManager(rmgr)
}
