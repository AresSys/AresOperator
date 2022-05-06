package metadata

// AresQueue related
const (
	// AresQueueAnnotationKey is the annotation key of Pod to identify
	// which ares-queue it belongs to.
	AresQueueAnnotationKey = GroupName + "/ares-queue"

	// PreemptableAnnotationKey is the annotation key of Pod to determine
	// if it is preemptable
	PreemptableAnnotationKey = GroupName + "/preemptable"

	// JobPriorityAnnotationKey is the annotation key of AresJob/PodGroup
	// to indicate the priority of job
	JobPriorityAnnotationKey = GroupName + "/job-priority"
)

// AresScheduler related
const (
	// KubeMonopolizeNodesAnnotationKey is the annotation key of Pod to declare that
	// it wants to monopolize nodes with other pods which have the same value of this annotation.
	// KubeMonopolizeNodesAnnotationKey是Pod的annotation关键字，用以声明其想与含有同样value的一组Pod独占一组节点
	KubeMonopolizeNodesAnnotationKey = "ares.io/monopolize-nodes"
)
