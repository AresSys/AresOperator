package reconciler

import (
	"strings"
	"testing"

	aresv1 "ares-operator/api/v1"
	aresmeta "ares-operator/metadata"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
)

func Test_diff(t *testing.T) {
	tests := []struct {
		current  map[string]string
		newmap   map[string]string
		expected map[string]interface{}
	}{
		{
			current: map[string]string{
				"foo":                          "bar",     // 保留
				aresmeta.GroupName + "/update": "bar",     // 修改
				aresmeta.GroupName + "/delete": "deleted", // 删除
			},
			newmap: map[string]string{
				aresmeta.GroupName + "/add":    "added",
				aresmeta.GroupName + "/update": "changed",
			},
			expected: map[string]interface{}{
				aresmeta.GroupName + "/add":    "added",   // 新增
				aresmeta.GroupName + "/update": "changed", // 修改
				aresmeta.GroupName + "/delete": nil,       // 删除
			},
		},
		{
			current: map[string]string{
				"foo":                          "bar",     // 保留
				aresmeta.GroupName + "/add":    "added",   // 新增
				aresmeta.GroupName + "/update": "changed", // 修改
			},
			newmap: map[string]string{
				"foo":                          "baz",
				aresmeta.GroupName + "/add":    "added",
				aresmeta.GroupName + "/update": "changed",
			},
			expected: map[string]interface{}(nil),
		},
	}
	for _, test := range tests {
		result := diff(test.current, test.newmap, func(key string) bool {
			return strings.HasPrefix(key, aresmeta.GroupName)
		})
		assert.Equal(t, test.expected, result)
		t.Logf("expected: %#v", test.expected)
	}
}

func Test_patchMetadata(t *testing.T) {
	pg := &scheduling.PodGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-test",
			Namespace: "default",
			Labels: map[string]string{
				"foo": "label",
			},
			Annotations: map[string]string{
				"foo":                             "annotation", // 保留
				aresmeta.JobPriorityAnnotationKey: "bar",        // 修改
				//aresmeta.PreemptableAnnotationKey: "deleted",    // 删除 // fakeClient暂时不支持删除
			},
		},
	}
	job := &aresv1.AresJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-test",
			Namespace: "default",
			Annotations: map[string]string{
				"foo":                             "job-annotation",
				aresmeta.AresQueueAnnotationKey:   "added",
				aresmeta.JobPriorityAnnotationKey: "changed",
			},
		},
	}
	job.Default()
	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(pg, job).
		Build()

	err := patchMetadata(cli, job, pg)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
		"foo":                             "annotation",
		aresmeta.AresQueueAnnotationKey:   "added",
		aresmeta.JobPriorityAnnotationKey: "changed",
	}, pg.Annotations)
	t.Logf("annotations: %#v", pg.Annotations)
	assert.Equal(t, map[string]string{
		"foo":                             "label",
		aresmeta.AresQueueAnnotationKey:   "added",
		aresmeta.JobPriorityAnnotationKey: "changed",
	}, pg.Labels)
	t.Logf("labels: %#v", pg.Labels)
}
