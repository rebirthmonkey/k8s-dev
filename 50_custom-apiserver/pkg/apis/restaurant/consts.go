package restaurant

// MigrationPhase is a label for the condition of a migration at the current time
type MigrationPhase string

type ApprovalResult string

const (
	ResourceNamespaces                    = "namespaces"
	ResourceConfigs                       = "configs"
	ResourceJumpHosts                     = "jumphosts"
	ResourceTunnels                       = "tunnels"
	ResourceTeams                         = "teams"
	ResourceUsers                         = "users"
	ResourceProjects                      = "projects"
	ResourceAPITokens                     = "apitokens"
	ResourceCatalogs                      = "catalogs"
	ResourceBinaryMetadatas               = "binarymetadatas"
	ResourceFullvpcmigrations             = "fullvpcmigrations"
	ResourceMovetovpcs                    = "movetovpcs"
	ResourceFvmPrecheck                   = "fvmprecheck"
	ResourceAvailabilityZoneMigrations    = "availabilityzonemigrations"
	ResourceSubnetZoneModifications       = "subnetzonemodifications"
	ResourceResetCVMsTypes                = "resetcvmstypes"
	ResourceDiskTypeUpgrades              = "disktypeupgrades"
	ResourceObjectstoremigrations         = "objectstoremigrations"
	ResourceJointeamrequests              = "jointeamrequests"
	ResourceWorkflowdefinitions           = "workflowdefinitions"
	ResourceWorkflowexecutions            = "workflowexecutions"
	ResourceWorkflowexecutiongroups       = "workflowexecutiongroups"
	ResourceElasticSearchBackups          = "elasticsearchbackups"
	ResourceElasticSearchRestores         = "elasticsearchrestores"
	ResourceVirtualMachineMigrations      = "virtualmachinemigrations"
	ResourceClassictovpcs                 = "classictovpcs"
	ResourceTopologyMappings              = "topologymappings"
	ResourceVirtualMachineBatchMigrations = "virtualmachinebatchmigrations"
	ResourceHttpProxyServers              = "httpproxyservers"
	ResourceEIPPools                      = "eippools"

	ApprovalResultPending  = ApprovalResult("Pending")
	ApprovalResultPassed   = ApprovalResult("Passed")
	ApprovalResultRejected = ApprovalResult("Rejected")

	SubResourceStatus   = "status"
	SubResourceReset    = "reset"
	SubResourceRevert   = "revert"
	SubResourceContinue = "continue"
	SubResourcePurge    = "purge"
	SubResourceAudit    = "audit"

	// Pending means migration is uninitialized
	Pending       MigrationPhase = "Pending"
	PendingFailed MigrationPhase = "PendingFailed"
	// PreFlight means status inited. in this phase precheck requests will be issued to
	// check if all prerequisites are met. if the answer is no, entering PreFlightFailed phase
	PreFlight MigrationPhase = "PreFlight"
	// InFlight means prechecking is successful. in this phase we will issue actual migration requests,
	// and check if it's finished successfully. if the answer is no, entering Failed phase
	InFlight MigrationPhase = "InFlight"

	// Succeeded means the migration has completed successfully
	Succeeded MigrationPhase = "Succeeded"
	// PreFlightFailed means that dryrun failed
	PreFlightFailed MigrationPhase = "PreFlightFailed"
	// Failed means the migration failed
	Failed MigrationPhase = "Failed"

	// PhaseLabelKey for resource filtering
	PhaseLabelKey = "50_custom-apiserver/phase"
	// NoReconcilingBeforeKey prevent resource reconciling before ( UNIX timestamp, seconds )
	NoReconcilingBeforeKey = "50_custom-apiserver/no-reconciling-before"

	// ReconcilingBarrier prevent resource reconciling when barrier reached. value schema: "activity_id:phase"
	ReconcilingBarrier = "50_custom-apiserver/reconciling-barrier"
	// OngoingFinalizer for graceful deletion
	OngoingFinalizer = "50_custom-apiserver/ongoing"
)
