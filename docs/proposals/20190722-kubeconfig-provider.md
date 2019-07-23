---
title: Custom workload cluster kubeconfigs
authors:
  - "@dlipovetsky"
reviewers:
  - "TBD"
creation-date: 2019-07-22
last-updated: 2019-07-22
status: provisional
---

# Custom workload cluster kubeconfigs

## Table of Contents

* [Custom kubeconfigs for workload clusters](#custom-kubeconfigs-for-workload-clusters)
  * [Table of Contents](#table-of-contents)
  * [Glossary](#glossary)
  * [Summary](#summary)
  * [Motivation](#motivation)
    * [Goals](#goals)
    * [Non\-Goals](#non-goals)
    * [Future Work](#future-work)
  * [Proposal](#proposal)
  * [User Stories](#user-stories)


## Glossary

- Kubeconfig
- Management Cluster
- Workload Cluster
- Proxy
- Tunnel
- Infrastructure Provider
- Cluster API Controllers
- API Server

## Summary

This proposal describes how to allow the Cluster API Controllers to reach Workload Cluster API servers that are not reachable directly, but are instead reachable through a proxy or tunnel.

It introduces a new controller--the Kubeconfig Provider--that generates kubeconfigs that contain information necessary to reach the Workload Cluster API server through a proxy or tunnel.

This change will enable a Management Cluster to run in a restricted network environment, and to manage Workload Clusters in restricted network environments.

## Motivation

Cluster API controllers use the workload cluster API. A kubeconfig gives the controllers all the information needed to reach the API server and authenticate with it. An infrastructure provider generates a kubeconfig for each workload cluster.

Some workload clusters may have an API server that cannot be reached by the Cluster API controllers. One example is a workload cluster whose API server is [accessible only from a network that disallows ingress from the internet](https://github.com/kubernetes-sigs/cluster-api-provider-aws/issues/873). Another example is a workload cluster whose API server is [accessible only through a proxy](https://github.com/kubernetes/client-go/issues/351).

Although an Infrastructure Provider generates a kubeconfig for each workload cluster, the Infrastructure Provider may not know that the workload cluster API server can be reached only through a proxy or tunnel.

ALTERNATIVES

let infra provider support custom kubeconfigs on its own. but that means each provider has to implement this on its own. and sometimes the problem is not within the provider's scope, e.g., when the CAPI controllers are in a restricted network environment.

<!-- This section is for explicitly listing the motivation, goals and non-goals of this proposal.
Describe why the change is important and the benefits to users.
The motivation section can optionally provide links to [experience reports][] to demonstrate the interest in a proposal within the wider Kubernetes community. -->

### Goals

- To generate a kubeconfig that enables the Cluster API controllers to reach the Workload Cluster through an HTTP/S proxy, a SOCKS proxy, or TCP tunnel.

### Non-Goals

- To generate kubeconfigs for any purpose other than use by Cluster API controllers.
- To configure the Workload Cluster to accept requests arriving through a proxy or tunnel. For example, to add the TCP tunnel IP or DNS name to the Subject Alt Names to the Workload Cluster's API Server X509 certificate.
- To configure the proxy or tunnel.

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is.

### User Stories [optional]

Detail the things that people will be able to do if this proposal is implemented.
Include as much detail as possible so that people can understand the "how" of the system.
The goal here is to make this feel real for users without getting bogged down.

#### Story 1

#### Story 2

### Implementation Details/Notes/Constraints [optional]

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

How will security be reviewed and by whom?
How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial proposal should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in proposal

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

If applicable, how will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a proposal should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the proposal was available
- the version of Kubernetes where the proposal graduated to general availability
- when the proposal was retired or superseded

## Drawbacks [optional]

Why should this proposal _not_ be implemented.

## Alternatives [optional]

Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a proposal.
