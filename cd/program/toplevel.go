package program

// Names this CD gives to the resources it registers at the top level of the
// stack, outside the defang-<cloud>:index:Project component.
//
// Everything else this CD creates hangs off that component, so its URN carries
// a defang-* package. These few do not, so the pre-flight check that refuses to
// operate on another CD's state (cd/legacy_state.go) has no other way to tell
// them apart from a stray resource left by an older CD.
//
// ADDING A TOP-LEVEL RESOURCE? Add its name here too, or the next deploy of
// every existing stack will abort.
var TopLevelResourceNames = map[string]bool{
	ProjectPbName:        true, // aws.go, gcp.go, azure.go
	SelfDestructName:     true, // selfdestruct_aws.go, selfdestruct_gcp.go
	SelfDestructJobName:  true, // selfdestruct_azure.go
	SelfDestructRoleName: true, // selfdestruct_azure.go
}

const (
	// ProjectPbName is the ProjectUpdate protobuf blob uploaded after a
	// successful deploy.
	ProjectPbName = "project-pb"
	// SelfDestructName is the AWS schedule + role and the GCP scheduler job
	// that run `defang cd down` on this stack at its TTL.
	SelfDestructName = "self-destruct"
	// SelfDestructJobName is the Azure Container Apps job that does the same.
	// It is deterministic so every redeploy updates the same job.
	SelfDestructJobName = "defang-self-destruct"
	// SelfDestructRoleName lets the Azure trigger job start the shared CD job.
	SelfDestructRoleName = "self-destruct-starter"
)
