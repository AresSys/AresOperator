module ares-operator

go 1.15

require (
	github.com/bombsimon/logrusr v1.1.0
	github.com/gin-gonic/gin v1.7.2
	github.com/go-logr/logr v0.3.0
	github.com/kubeflow/common v0.3.4
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.19.9
	k8s.io/apimachinery v0.19.9
	k8s.io/client-go v0.19.9
	sigs.k8s.io/controller-runtime v0.7.2
	volcano.sh/apis v1.2.0-k8s1.19.6
)

replace (
	k8s.io/api => k8s.io/api v0.19.9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.9
	k8s.io/apiserver => k8s.io/apiserver v0.19.9
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.9
	k8s.io/client-go => k8s.io/client-go v0.19.9
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.9
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.19.9
	k8s.io/code-generator => k8s.io/code-generator v0.19.9
	k8s.io/component-base => k8s.io/component-base v0.19.9
	k8s.io/cri-api => k8s.io/cri-api v0.16.10-beta.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.19.9
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.19.9
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.19.9
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.19.9
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.19.9
	k8s.io/kubectl => k8s.io/kubectl v0.19.9
	k8s.io/kubelet => k8s.io/kubelet v0.19.9
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.19.9
	k8s.io/metrics => k8s.io/metrics v0.19.9
	k8s.io/node-api => k8s.io/node-api v0.19.9
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.19.9
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.19.9
	k8s.io/sample-controller => k8s.io/sample-controller v0.19.9
)
