package v1

const (
	// ReplicaIndexLabel represents the label key for the replica-index, e.g. the value is 0, 1, 2.. etc
	ReplicaIndexLabel = "ares.io/replica-index"

	// ReplicaTypeLabel represents the label key for the replica-type, e.g. the value is ps , worker etc.
	ReplicaTypeLabel = "ares.io/replica-type"

	// GroupNameLabel represents the label key for group name, e.g. the value is ares.io
	GroupNameLabel = "ares.io/group-name"

	// JobNameLabel represents the label key for the job name, the value is job name
	JobNameLabel = "ares.io/job-name"

	// JobRoleLabel represents the label key for the job role, e.g. the value is master
	JobRoleLabel = "ares.io/job-role"
)
