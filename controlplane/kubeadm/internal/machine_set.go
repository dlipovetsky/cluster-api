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

// Modified copy of k8s.io/apimachinery/pkg/util/sets/int64.go
// Modifications
//   - int64 became *clusterv1.Machine
//   - Empty type is removed
//   - Sortable data type is removed in favor of util.MachinesByCreationTimestamp
//   - nil checks added to account for the pointer
//   - Added Filter, AnyFilter, and Oldest methods
//   - Added NewMachineSetFromMachineList initializer
//   - Updated Has to also check for equality of Machines

package internal

import (
	"reflect"
	"sort"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
)

// MachineSet is a set of Machines
type MachineSet map[string]*clusterv1.Machine

// NewMachineSet creates a MachineSet from a list of values.
func NewMachineSet(machines ...*clusterv1.Machine) MachineSet {
	ss := MachineSet{}
	ss.Insert(machines...)
	return ss
}

// NewMachineSetFromMachineList creates a MachineSet from the given MachineList
func NewMachineSetFromMachineList(machineList *clusterv1.MachineList) MachineSet {
	ss := MachineSet{}
	if machineList != nil {
		for i := range machineList.Items {
			ss.Insert(&machineList.Items[i])
		}
	}
	return ss
}

// Insert adds items to the set.
func (s MachineSet) Insert(machines ...*clusterv1.Machine) MachineSet {
	for i := range machines {
		if machines[i] != nil {
			m := machines[i]
			s[m.Name] = m
		}
	}
	return s
}

// Delete removes all items from the set.
func (s MachineSet) Delete(machines ...*clusterv1.Machine) MachineSet {
	for i := range machines {
		if machines[i] != nil {
			m := machines[i]
			delete(s, m.Name)
		}
	}
	return s
}

// Has returns true if and only if item is contained in the set.
func (s MachineSet) Has(item *clusterv1.Machine) bool {
	if item == nil {
		return false
	}
	_, contained := s[item.Name]
	return contained && reflect.DeepEqual(item, s[item.Name])
}

// HasAll returns true if and only if all items are contained in the set.
func (s MachineSet) HasAll(items ...*clusterv1.Machine) bool {
	for _, item := range items {
		if !s.Has(item) {
			return false
		}
	}
	return true
}

// HasAny returns true if any items are contained in the set.
func (s MachineSet) HasAny(items ...*clusterv1.Machine) bool {
	for _, item := range items {
		if s.Has(item) {
			return true
		}
	}
	return false
}

// Difference returns a set of objects that are not in s2
// For example:
// s1 = {a1, a2, a3}
// s2 = {a1, a2, a4, a5}
// s1.Difference(s2) = {a3}
// s2.Difference(s1) = {a4, a5}
func (s MachineSet) Difference(s2 MachineSet) MachineSet {
	result := NewMachineSet()
	for _, value := range s {
		if !s2.Has(value) {
			result.Insert(value)
		}
	}
	return result
}

// Union returns a new set which includes items in either s1 or s2.
// For example:
// s1 = {a1, a2}
// s2 = {a3, a4}
// s1.Union(s2) = {a1, a2, a3, a4}
// s2.Union(s1) = {a1, a2, a3, a4}
func (s MachineSet) Union(s2 MachineSet) MachineSet {
	s1 := s
	result := NewMachineSet()
	for _, value := range s1 {
		result.Insert(value)
	}
	for _, value := range s2 {
		result.Insert(value)
	}
	return result
}

// Intersection returns a new set which includes the item in BOTH s1 and s2
// For example:
// s1 = {a1, a2}
// s2 = {a2, a3}
// s1.Intersection(s2) = {a2}
func (s MachineSet) Intersection(s2 MachineSet) MachineSet {
	s1 := s
	var walk, other MachineSet
	result := NewMachineSet()
	if s1.Len() < s2.Len() {
		walk = s1
		other = s2
	} else {
		walk = s2
		other = s1
	}
	for _, value := range walk {
		if other.Has(value) {
			result.Insert(value)
		}
	}
	return result
}

// IsSuperset returns true if and only if s1 is a superset of s2.
func (s MachineSet) IsSuperset(s2 MachineSet) bool {
	s1 := s
	for _, value := range s2 {
		if !s1.Has(value) {
			return false
		}
	}
	return true
}

// Equal returns true if and only if s1 is equal (as a set) to s2.
// Two sets are equal if their membership is identical.
// (In practice, this means same elements, order doesn't matter)
func (s MachineSet) Equal(s2 MachineSet) bool {
	s1 := s
	return len(s1) == len(s2) && s1.IsSuperset(s2)
}

// List returns the contents as a sorted Machine slice.
func (s MachineSet) List() []*clusterv1.Machine {
	res := make(util.MachinesByCreationTimestamp, 0, len(s))
	for _, value := range s {
		res = append(res, value)
	}
	sort.Sort(res)
	return res
}

// UnsortedList returns the slice with contents in random order.
func (s MachineSet) UnsortedList() []*clusterv1.Machine {
	res := make([]*clusterv1.Machine, 0, len(s))
	for _, value := range s {
		res = append(res, value)
	}
	return res
}

// PopAny returns a single element from the set.
func (s MachineSet) PopAny() (*clusterv1.Machine, bool) {
	for _, value := range s {
		s.Delete(value)
		return value, true
	}
	return nil, false
}

// Len returns the size of the set.
func (s MachineSet) Len() int {
	return len(s)
}

func newFilteredMachineSet(filter MachineFilter, machines ...*clusterv1.Machine) MachineSet {
	ss := MachineSet{}
	for i := range machines {
		m := machines[i]
		if filter(m) {
			ss.Insert(m)
		}
	}
	return ss
}

// Filter returns a MachineSet containing only the Machines that match all of the given MachineFilters
func (s MachineSet) Filter(filters ...MachineFilter) MachineSet {
	return newFilteredMachineSet(And(filters...), s.UnsortedList()...)
}

// AnyFilter returns a MachineSet containing only the Machines that match any of the given MachineFilters
func (s MachineSet) AnyFilter(filters ...MachineFilter) MachineSet {
	return newFilteredMachineSet(Or(filters...), s.UnsortedList()...)
}

// Oldest returns the Machine with the oldest CreationTimestamp
func (s MachineSet) Oldest() *clusterv1.Machine {
	if len(s) == 0 {
		return nil
	}
	return s.List()[len(s)-1]
}
