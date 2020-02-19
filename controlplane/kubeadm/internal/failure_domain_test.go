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

package internal

import (
	"testing"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

func TestNewFailureDomainPicker(t *testing.T) {
	a := "us-west-1a"
	b := "us-west-1b"

	fds := clusterv1.FailureDomains{
		a: clusterv1.FailureDomainSpec{},
		b: clusterv1.FailureDomainSpec{},
	}
	machinea := &clusterv1.Machine{Spec: clusterv1.MachineSpec{FailureDomain: &a}}
	machineb := &clusterv1.Machine{Spec: clusterv1.MachineSpec{FailureDomain: &b}}
	machinenil := &clusterv1.Machine{Spec: clusterv1.MachineSpec{FailureDomain: nil}}

	testcases := []struct {
		name     string
		fds      clusterv1.FailureDomains
		machines MachineSet
		expected []string
	}{
		{
			name:     "simple",
			expected: []string{""},
		},
		{
			name: "no machines",
			fds: clusterv1.FailureDomains{
				a: clusterv1.FailureDomainSpec{},
			},
			expected: []string{a},
		},
		{
			name:     "one machine in a failure domain",
			fds:      fds,
			machines: NewMachineSet(machinea.DeepCopy()),
			expected: []string{b},
		},
		{
			name: "no failure domain specified on machine",
			fds: clusterv1.FailureDomains{
				a: clusterv1.FailureDomainSpec{},
			},
			machines: NewMachineSet(machinenil.DeepCopy()),
			expected: []string{a, b},
		},
		{
			name: "mismatched failure domain on machine",
			fds: clusterv1.FailureDomains{
				a: clusterv1.FailureDomainSpec{},
			},
			machines: NewMachineSet(machineb.DeepCopy()),
			expected: []string{a},
		},
		{
			name:     "failure domains and no machines should return a valid failure domain",
			fds:      fds,
			expected: []string{a, b},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fd := PickFewest(tc.fds, tc.machines)

			found := false
			for _, expectation := range tc.expected {
				if fd == expectation {
					found = true
				}
			}
			if !found {
				t.Fatal("did not find expected value")
			}
		})
	}
}
