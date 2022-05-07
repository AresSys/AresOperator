package tensorflow

import (
	aresv1 "ares-operator/api/v1"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

const (
	// DefaultPortName is name of the port used to communicate between PS and
	// workers.
	DefaultPortName = "tfjob-port"
	// DefaultPort is default value of the port.
	DefaultPort = 2222
	// DefaultRestartPolicy is default RestartPolicy for TFReplicaSpec.
	DefaultRestartPolicy = commonv1.RestartPolicyNever
)

const (
	tfConfig = "TF_CONFIG"
)

// TaskSpec is the specification for a task (PS or worker) of the TFJob.
type TaskSpec struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

// ClusterSpec represents a cluster TensorFlow specification.
// https://www.tensorflow.org/deploy/distributed#create_a_tftrainclusterspec_to_describe_the_cluster
// It is a map from job names to network addresses.
type ClusterSpec map[string][]string

// TFConfig is a struct representing the distributed TensorFlow config.
// This struct is turned into an environment variable TF_CONFIG
// which is used by TensorFlow processes to configure themselves.
// https://www.tensorflow.org/api_docs/python/tf/estimator/RunConfig#methods
// https://cloud.google.com/ml-engine/docs/tensorflow/distributed-training-details
type TFConfig struct {
	// Cluster represents a TensorFlow ClusterSpec.
	// See: https://www.tensorflow.org/api_docs/python/tf/train/ClusterSpec
	Cluster ClusterSpec `json:"cluster"`
	Task    TaskSpec    `json:"task"`
	// Environment is used by tensorflow.contrib.learn.python.learn in versions <= 1.3
	// TODO(jlewi): I don't think it is used in versions TF >- 1.4. So we can eventually get rid of it.
	Environment string `json:"environment"`
}

// SparseClusterSpec enables a server to be configured without needing to know
// the identity of (for example) all other worker tasks.
// https://www.tensorflow.org/api_docs/python/tf/train/ClusterSpec
type SparseClusterSpec struct {
	Worker map[int32]string `json:"worker"`
	PS     []string         `json:"ps"`
}

type SparseTFConfig struct {
	Cluster SparseClusterSpec `json:"sparseCluster"`
	Task    TaskSpec          `json:"task"`
}

func convertClusterSpecToSparseClusterSpec(clusterSpec ClusterSpec, rtype string, index int32) SparseClusterSpec {
	sparseClusterSpec := SparseClusterSpec{Worker: map[int32]string{}, PS: []string{}}
	if rtype == string(aresv1.RolePS) {
		sparseClusterSpec.PS = append(sparseClusterSpec.PS, clusterSpec[rtype][index])
	} else if rtype == string(aresv1.RoleWorker) {
		sparseClusterSpec.PS = clusterSpec[string(aresv1.RolePS)]
		sparseClusterSpec.Worker[index] = clusterSpec[rtype][index]
	}
	return sparseClusterSpec
}
