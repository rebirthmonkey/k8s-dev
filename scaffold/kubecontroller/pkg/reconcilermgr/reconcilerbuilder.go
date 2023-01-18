package reconcilermgr

type ReconcilersBuilder []func(*ReconcilerManager) error

func (rb *ReconcilersBuilder) Register(funcs ...func(*ReconcilerManager) error) {
	for _, f := range funcs {
		*rb = append(*rb, f)
	}
}

func (rb *ReconcilersBuilder) AddToManager(rmgr *ReconcilerManager) error {
	for _, f := range *rb {
		if err := f(rmgr); err != nil {
			return err
		}
	}
	return nil
}
