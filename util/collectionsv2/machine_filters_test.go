/*
Copyright 2020 The Kubernetes Authors.

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

package collections_test

import (
	"testing"

	collections "sigs.k8s.io/cluster-api/util/collectionsv2"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
)

func TestExample(t *testing.T) {
	a := &clusterv1.Machine{}
	filter := collections.And(collections.Not(collections.HasControllerRef), collections.InFailureDomains(nil))
	result := filter(a)
	t.Log(result)
}
