/*
Copyright 2022 KML Ares-Operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package reconciler

import (
	"context"
	"encoding/json"

	aresmeta "ares-operator/metadata"

	"github.com/kubeflow/common/pkg/util"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// diff: diff两个string map
func diff(current, expected map[string]string, filter func(string) bool) map[string]interface{} {
	if current == nil {
		current = map[string]string{}
	}
	if expected == nil {
		expected = map[string]string{}
	}
	result := map[string]interface{}{}
	// add
	for key, val := range expected {
		if !filter(key) {
			continue
		}
		if oldVal, exists := current[key]; !exists || oldVal != val {
			result[key] = val
		}
	}
	// delete
	for key := range current {
		if !filter(key) {
			continue
		}
		if _, exists := expected[key]; !exists {
			result[key] = nil // delete
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// patchMetadata: 修订Metadata
func patchMetadata(cli client.Client, expected, object client.Object) error {
	annotations := diff(object.GetAnnotations(), expected.GetAnnotations(), aresmeta.IsJobLevelKey)
	labels := diff(object.GetLabels(), expected.GetAnnotations(), aresmeta.IsJobLevelKey) // NOTE: 以job的Annotations为准
	if len(annotations) == 0 && len(labels) == 0 {
		return nil
	}
	patch := map[string]map[string]interface{}{
		"metadata": {},
	}
	if len(annotations) > 0 {
		patch["metadata"]["annotations"] = annotations
	}
	if len(labels) > 0 {
		patch["metadata"]["labels"] = labels
	}
	content, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	err = cli.Patch(context.TODO(), object, client.RawPatch(types.MergePatchType, content))
	if err != nil {
		return err
	}
	util.LoggerForJob(expected).Infof("succeeded to patch to %s <%s/%s>: patch=%v",
		object.GetObjectKind().GroupVersionKind().Kind, object.GetNamespace(), object.GetName(), patch,
	)
	return nil
}
