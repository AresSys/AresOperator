package v1

/****************
 * Task Topology
 ****************/
const (
	// pod annotation
	// https://github.com/volcano-sh/apis/blob/master/pkg/apis/batch/v1alpha1/labels.go#L21

	// TaskSpecKey task spec key used in pod annotation
	TaskSpecKey = "volcano.sh/task-spec"
)

const (
	// podgroup annotation
	// https://github.com/volcano-sh/volcano/blob/v1.3.0/pkg/scheduler/plugins/task-topology/util.go#L35

	// JobAffinityAnnotations is the key to read in task-topology affinity arguments from podgroup annotations
	JobAffinityAnnotations = "volcano.sh/task-topology-affinity"
	// JobAntiAffinityAnnotations is the key to read in task-topology anti-affinity arguments from podgroup annotations
	JobAntiAffinityAnnotations = "volcano.sh/task-topology-anti-affinity"
	// TaskOrderAnnotations is the key to read in task-topology task order arguments from podgroup annotations
	TaskOrderAnnotations = "volcano.sh/task-topology-task-order"
)
