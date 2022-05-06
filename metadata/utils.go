package metadata

// Is the given key at job level?
func IsJobLevelKey(key string) bool {
	switch key {
	case AresQueueAnnotationKey,
		PreemptableAnnotationKey,
		JobPriorityAnnotationKey:
		return true
	default:
		return false
	}
}
