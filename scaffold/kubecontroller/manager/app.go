package manager

import (
	"github.com/rebirthmonkey/go/pkg/app"
	"github.com/rebirthmonkey/go/pkg/log"
)

// NewApp creates an App object with default parameters.
func NewApp(basename string) *app.App {
	opts := NewOptions()
	application := app.NewApp("kubecontroller",
		basename,
		app.WithOptions(opts),
		app.WithDescription("kubecontroller description"),
		app.WithRunFunc(run(opts)),
	)

	return application
}

// run launches the App object.
func run(opts *Options) app.RunFunc {
	return func(basename string) error {
		log.Info("[App] Run")
		manager, err := NewManager(opts)
		if err != nil {
			return err
		}

		return manager.PrepareRun().Run()
	}
}
