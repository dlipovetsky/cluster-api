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

package collections

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha4"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha4"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FilterResult struct {
	accepted bool
	reason   string
}

// Func is the functon definition for a filter.
type Func func(machine *clusterv1.Machine) FilterResult

// And returns a filter that returns true if all of the given filters returns true.
func And(filters ...Func) Func {
	return func(machine *clusterv1.Machine) FilterResult {
		reasons := []string{}
		for _, f := range filters {
			result := f(machine)
			if !result.accepted {
				return result
			}
			reasons = append(reasons, result.reason)
		}
		return FilterResult{false, strings.Join(reasons, ", and ")}
	}
}

// Or returns a filter that returns true if any of the given filters returns true.
func Or(filters ...Func) Func {
	return func(machine *clusterv1.Machine) FilterResult {
		reasons := []string{}
		for _, f := range filters {
			result := f(machine)
			if result.accepted {
				return result
			}
			reasons = append(reasons, result.reason)
		}
		return FilterResult{false, fmt.Sprintf("not %s", strings.Join(reasons, ", or "))}
	}
}

// Not returns a filter that returns the opposite of the given filter.
func Not(mf Func) Func {
	return func(machine *clusterv1.Machine) FilterResult {
		result := mf(machine)
		reason := fmt.Sprintf("not %s", result.reason)
		if !result.accepted {
			return FilterResult{true, reason}
		}
		return FilterResult{false, reason}
	}
}

// HasControllerRef is a filter that returns true if the machine has a controller ref.
func HasControllerRef(machine *clusterv1.Machine) FilterResult {
	if machine == nil {
		return FilterResult{false, "is nil"}
	}
	return FilterResult{metav1.GetControllerOf(machine) != nil, "has controller ref"}
}

// InFailureDomains returns a filter to find all machines
// in any of the given failure domains.
func InFailureDomains(failureDomains ...*string) Func {
	return func(machine *clusterv1.Machine) FilterResult {
		if machine == nil {
			return FilterResult{false, "is nil"}
		}
		for i := range failureDomains {
			fd := failureDomains[i]
			if fd == nil {
				if fd == machine.Spec.FailureDomain {
					return FilterResult{true, "is in failure domain nil"}
				}
				continue
			}
			if machine.Spec.FailureDomain == nil {
				continue
			}
			if *fd == *machine.Spec.FailureDomain {
				return FilterResult{true, fmt.Sprintf("is in failure domain %s", *fd)}
			}
		}
		return FilterResult{false, "is not in any failure domain"}
	}
}

// OwnedMachines returns a filter to find all machines owned by specified owner.
// Usage: GetFilteredMachinesForCluster(ctx, client, cluster, OwnedMachines(controlPlane)).
func OwnedMachines(owner client.Object) func(machine *clusterv1.Machine) FilterResult {
	return func(machine *clusterv1.Machine) FilterResult {
		if machine == nil {
			return FilterResult{false, "is nil"}
		}
		if util.IsOwnedByObject(machine, owner) {
			return FilterResult{true, fmt.Sprintf("is owned by object %s", owner.GetName())}
		}
		return FilterResult{false, fmt.Sprintf("is not owned by object %s", owner.GetName())}
	}
}

// ControlPlaneMachines returns a filter to find all control plane machines for a cluster, regardless of ownership.
// Usage: GetFilteredMachinesForCluster(ctx, client, cluster, ControlPlaneMachines(cluster.Name)).
func ControlPlaneMachines(clusterName string) func(machine *clusterv1.Machine) FilterResult {
	selector := ControlPlaneSelectorForCluster(clusterName)
	return func(machine *clusterv1.Machine) FilterResult {
		if machine == nil {
			return FilterResult{false, "is nil"}
		}
		if selector.Matches(labels.Set(machine.Labels)) {
			return FilterResult{true, fmt.Sprintf("selector %s matches", selector)}
		}
		return FilterResult{false, fmt.Sprintf("selector %s does not match", selector)}
	}
}

// AdoptableControlPlaneMachines returns a filter to find all un-controlled control plane machines.
// Usage: GetFilteredMachinesForCluster(ctx, client, cluster, AdoptableControlPlaneMachines(cluster.Name, controlPlane)).
func AdoptableControlPlaneMachines(clusterName string) func(machine *clusterv1.Machine) FilterResult {
	return And(
		ControlPlaneMachines(clusterName),
		Not(HasControllerRef),
	)
}

// ActiveMachines returns a filter to find all active machines.
// Usage: GetFilteredMachinesForCluster(ctx, client, cluster, ActiveMachines).
func ActiveMachines(machine *clusterv1.Machine) FilterResult {
	result := HasDeletionTimestamp(machine)
	if result.accepted {
		return FilterResult{false, fmt.Sprintf("is not active (%s)", result.reason)}
	}
	return FilterResult{true, fmt.Sprintf("is active (%s)", result.reason)}
}

// HasDeletionTimestamp returns a filter to find all machines that have a deletion timestamp.
func HasDeletionTimestamp(machine *clusterv1.Machine) FilterResult {
	if machine == nil {
		return FilterResult{false, "is nil"}
	}
	if !machine.DeletionTimestamp.IsZero() {
		return FilterResult{true, "has deletion timestamp"}
	}
	return FilterResult{false, "does not have deletion timestamp"}
}

// HasUnhealthyCondition returns a filter to find all machines that have a MachineHealthCheckSucceeded condition set to False,
// indicating a problem was detected on the machine, and the MachineOwnerRemediated condition set, indicating that KCP is
// responsible of performing remediation as owner of the machine.
func HasUnhealthyCondition(machine *clusterv1.Machine) FilterResult {
	if machine == nil {
		return FilterResult{false, "is nil"}
	}
	if conditions.IsTrue(machine, clusterv1.MachineHealthCheckSuccededCondition) {
		return FilterResult{false, fmt.Sprintf("condition %s is true", clusterv1.MachineHealthCheckSuccededCondition)}
	}
	if conditions.IsTrue(machine, clusterv1.MachineOwnerRemediatedCondition) {
		return FilterResult{false, fmt.Sprintf("condition %s is true", clusterv1.MachineOwnerRemediatedCondition)}
	}
	return FilterResult{true, fmt.Sprintf("conditions %s and %s are false",
		clusterv1.MachineHealthCheckSuccededCondition, clusterv1.MachineOwnerRemediatedCondition)}
}

// IsReady returns a filter to find all machines with the ReadyCondition equals to True.
func IsReady() Func {
	return func(machine *clusterv1.Machine) FilterResult {
		if machine == nil {
			return FilterResult{false, "is nil"}
		}
		accepted := conditions.IsTrue(machine, clusterv1.ReadyCondition)
		return FilterResult{accepted, fmt.Sprintf("condition %s is %t", clusterv1.ReadyCondition, accepted)}
	}
}

// ShouldRolloutAfter returns a filter to find all machines where
// CreationTimestamp < rolloutAfter < reconciliationTIme.
func ShouldRolloutAfter(reconciliationTime, rolloutAfter *metav1.Time) Func {
	return func(machine *clusterv1.Machine) FilterResult {
		if machine == nil {
			return FilterResult{false, "is nil"}
		}
		if !machine.CreationTimestamp.Before(rolloutAfter) {
			return FilterResult{false, "creationTimestamp is not before rolloutAfter"}
		}
		if !rolloutAfter.Before(reconciliationTime) {
			return FilterResult{false, "rolloutAfter is not before reconciliationTime"}
		}
		return FilterResult{true, "creationTimestamp is before rolloutAfter, and rolloutAfter is before reconciliationTime"}
	}
}

// HasAnnotationKey returns a filter to find all machines that have the
// specified Annotation key present.
func HasAnnotationKey(key string) Func {
	return func(machine *clusterv1.Machine) FilterResult {
		if machine == nil {
			return FilterResult{false, "is nil"}
		}
		if machine.Annotations == nil {
			return FilterResult{false, "annotations are nil"}
		}
		if _, ok := machine.Annotations[key]; ok {
			return FilterResult{true, fmt.Sprintf("has annotation %s", key)}
		}
		return FilterResult{false, fmt.Sprintf("does not have annotation %s", key)}
	}
}

// ControlPlaneSelectorForCluster returns the label selector necessary to get control plane machines for a given cluster.
func ControlPlaneSelectorForCluster(clusterName string) labels.Selector {
	must := func(r *labels.Requirement, err error) labels.Requirement {
		if err != nil {
			panic(err)
		}
		return *r
	}
	return labels.NewSelector().Add(
		must(labels.NewRequirement(clusterv1.ClusterLabelName, selection.Equals, []string{clusterName})),
		must(labels.NewRequirement(clusterv1.MachineControlPlaneLabelName, selection.Exists, []string{})),
	)
}

// MatchesKubernetesVersion returns a filter to find all machines that match a given Kubernetes version.
func MatchesKubernetesVersion(kubernetesVersion string) Func {
	return func(machine *clusterv1.Machine) FilterResult {
		if machine == nil {
			return FilterResult{false, "is nil"}
		}
		if machine.Spec.Version == nil {
			return FilterResult{false, "spec.version is nil"}
		}
		if *machine.Spec.Version == kubernetesVersion {
			return FilterResult{true, fmt.Sprintf("spec.version equals %s", kubernetesVersion)}
		}
		return FilterResult{false, fmt.Sprintf("spec.version does not equal %s", kubernetesVersion)}
	}
}

// WithVersion returns a filter to find machine that have a non empty and valid version.
func WithVersion() Func {
	return func(machine *clusterv1.Machine) FilterResult {
		if machine == nil {
			return FilterResult{false, "is nil"}
		}
		if machine.Spec.Version == nil {
			return FilterResult{false, "spec.version is nil"}
		}
		if _, err := semver.ParseTolerant(*machine.Spec.Version); err != nil {
			return FilterResult{false, fmt.Sprintf("spec.version %s is not a valid semver (%s)", *machine.Spec.Version, err)}
		}
		return FilterResult{true, fmt.Sprintf("spec.version %s is a valid semver", *machine.Spec.Version)}
	}
}

// HealthyAPIServer returns a filter to find all machines that have a MachineAPIServerPodHealthyCondition
// set to true.
func HealthyAPIServer() Func {
	return func(machine *clusterv1.Machine) FilterResult {
		if machine == nil {
			return FilterResult{false, "is nil"}
		}
		accepted := conditions.IsTrue(machine, controlplanev1.MachineAPIServerPodHealthyCondition)
		return FilterResult{accepted, fmt.Sprintf("condition %s is %t", controlplanev1.MachineAPIServerPodHealthyCondition, accepted)}
	}
}
