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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"
)

type MachineFilter func(machine *clusterv1.Machine) bool

// Not returns a MachineFilter function that returns the opposite of the given MachineFilter
func Not(mf MachineFilter) MachineFilter {
	return func(machine *clusterv1.Machine) bool {
		return !mf(machine)
	}
}

// InFailureDomain returns a MachineFilter function to find all machines
// in a given failure domain
func InFailureDomain(failureDomain *string) MachineFilter {
	return func(machine *clusterv1.Machine) bool {
		if machine == nil {
			return false
		}
		if failureDomain == nil {
			return true
		}
		if machine.Spec.FailureDomain == nil {
			return false
		}
		return *machine.Spec.FailureDomain == *failureDomain
	}
}

// OwnedControlPlaneMachines returns a MachineFilter function to find all owned control plane machines.
// Usage: managementCluster.GetMachinesForCluster(ctx, cluster, OwnedControlPlaneMachines(controlPlane.Name))
func OwnedControlPlaneMachines(controlPlaneName string) MachineFilter {
	return func(machine *clusterv1.Machine) bool {
		if machine == nil {
			return false
		}
		controllerRef := metav1.GetControllerOf(machine)
		if controllerRef == nil {
			return false
		}
		return controllerRef.Kind == "KubeadmControlPlane" && controllerRef.Name == controlPlaneName
	}
}

// HasDeletionTimestamp returns a MachineFilter function to find all machines
// that have a deletion timestamp.
func HasDeletionTimestamp() MachineFilter {
	return func(machine *clusterv1.Machine) bool {
		if machine == nil {
			return false
		}
		return !machine.DeletionTimestamp.IsZero()
	}
}

// MatchesConfigurationHash returns a MachineFilter function to find all machines
// that match a given KubeadmControlPlane configuration hash.
func MatchesConfigurationHash(configHash string) MachineFilter {
	return func(machine *clusterv1.Machine) bool {
		if machine == nil {
			return false
		}
		if hash, ok := machine.Labels[controlplanev1.KubeadmControlPlaneHashLabelKey]; ok {
			return hash == configHash
		}
		return false
	}
}

// OlderThan returns a MachineFilter function to find all machines
// that have a CreationTimestamp earlier than the given time.
func OlderThan(t *metav1.Time) MachineFilter {
	return func(machine *clusterv1.Machine) bool {
		if machine == nil {
			return false
		}
		return machine.CreationTimestamp.Before(t)
	}
}

// SelectedForUpgrade returns a MachineFilter function to find all machines that have the
// controlplanev1.SelectedForUpgradeLabel set.
func SelectedForUpgrade() MachineFilter {
	return func(machine *clusterv1.Machine) bool {
		if machine == nil || machine.Labels == nil {
			return false
		}
		if _, ok := machine.Labels[controlplanev1.SelectedForUpgradeLabel]; ok {
			return true
		}
		return false
	}
}

// UnionFilterMachines returns a filtered list of machines that matches the union of the applied filters
func UnionFilterMachines(machines []*clusterv1.Machine, filters ...MachineFilter) []*clusterv1.Machine {
	if len(filters) == 0 {
		return machines
	}
	filteredMachines := make([]*clusterv1.Machine, 0, len(machines))

Machine:
	for _, machine := range machines {
		for _, filter := range filters {
			if filter(machine) {
				filteredMachines = append(filteredMachines, machine)
				break Machine
			}
		}
	}

	return filteredMachines
}

// FilterMachines returns a filtered list of machines that matches the intersection of the applied filters
func FilterMachines(machines []*clusterv1.Machine, filters ...MachineFilter) []*clusterv1.Machine {
	if len(filters) == 0 {
		return machines
	}

	filteredMachines := make([]*clusterv1.Machine, 0, len(machines))
	for _, machine := range machines {
		add := true
		for _, filter := range filters {
			if !filter(machine) {
				add = false
				break
			}
		}
		if add {
			filteredMachines = append(filteredMachines, machine)
		}
	}
	return filteredMachines
}
