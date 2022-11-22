package controller

const (
	ContextKeyClient        = "ControllersContextKeyClient"
	ContextKeyNoCacheClient = "ControllersContextKeyNoCacheClient"
	ContextKeyLogger        = "ControllersContextKeyLogger"
	ContextKeyMutex         = "ControllersContextKeyMutex"
	// ShouldBeReconciledBy indicates that the resource should be processed by specific controller manager
	ShouldBeReconciledBy    = "cloud.tencent.com/should-be-reconciled-by"
	MainControllerManagerId = "main"
)
