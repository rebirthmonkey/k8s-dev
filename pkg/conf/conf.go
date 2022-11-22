package conf

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	HttpProxy              = "http-proxy"
	APIExtsURL             = "apiexts-url"
	MetricsBindAddress     = "metrics-bind-address"
	HealthProbeBindAddress = "health-probe-bind-address"
	EnableLeaderElection   = "enable-leader-election"
	LeaderElectionID       = "leader-election-id"
	Namespace              = "namespace"
	Port                   = "port"
	RetryDefer             = "retry-defer"
	AsyncUpdateDefer       = "async-update-defer"
	ConfirmDefer           = "confirm-defer"
	PausedRequeueDefer     = "paused-requeue-defer"
	ConfirmTimeout         = "confirm-timeout"
	ReconcileConcurrence   = "reconcile-concurrence"
	SyncPeriod             = "sync-period"
	BearerToken            = "bearer-token"

	WDTerraformModulesRoot = "workflowdefinitions.terraform.modules-root"
	WDBinaryRoot           = "workflowdefinitions.binary.root"
	HttProxyRoot           = "workflowdefinitions.binary.http-proxy-root"

	WEActivityLogRoot                    = "workflowexecutions.activity-log-root"
	WEActivityWorkspaceRoot              = "workflowexecutions.activity-workspace-root"
	WEScriptDisableRemoteAPIServerAccess = "workflowexecutions.script-activity.disable-remote-apiserver-access"

	OnlyReconcileLabeled = "only-reconcile-labeled"
	ControllerManagerId  = "controller-manager-id"
)

var v *viper.Viper

type camelCase struct {
}

// Replace returns a copy of s with all replacements performed.
func (c camelCase) Replace(s string) string {
	return strings.ReplaceAll(s, "-", "_")
}

// Init config
func Init(configPath string) error {
	v = viper.NewWithOptions(viper.EnvKeyReplacer(camelCase{}))
	v.SetDefault(MetricsBindAddress, ":8001")
	v.SetDefault(HealthProbeBindAddress, ":8002")
	v.SetDefault(EnableLeaderElection, false)
	v.SetDefault(Namespace, "")
	v.SetDefault(Port, 9443)
	v.SetDefault(RetryDefer, 15*time.Second)
	v.SetDefault(AsyncUpdateDefer, 15*time.Second)
	v.SetDefault(PausedRequeueDefer, 30*time.Second)
	v.SetDefault(ConfirmDefer, 15*time.Second)
	v.SetDefault(ConfirmTimeout, 15*30*time.Second)
	v.SetDefault(SyncPeriod, 5*time.Minute)
	v.SetDefault(ReconcileConcurrence, 1)
	v.AutomaticEnv()
	v.SetConfigFile(configPath)
	return v.ReadInConfig()
}

// Get config
func Get(key string) string {
	return v.GetString(key)
}

// GetSlice config
func GetSlice(key string) []string {
	return v.GetStringSlice(key)
}

// GetInt config
func GetInt(key string) int {
	return v.GetInt(key)
}

// GetBool config
func GetBool(key string) bool {
	return v.GetBool(key)
}

// GetDuration config
func GetDuration(key string) time.Duration {
	return v.GetDuration(key)
}

func Set(key string, val interface{}) {
	v.Set(key, val)
}
