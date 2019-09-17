# Control Plane Management

The purpose of this document is to help ensure we have a common understanding of the control plane management problem when we discuss how Cluster API can address it. Most of the document explains the Kubernetes control plane in theory and in practice. It concludes with a view of control plane management in Cluster API today.

- [What is the Kubernetes control plane?](#what-is-the-kubernetes-control-plane)
  - [Where is the control plane state?](#where-is-the-control-plane-state)
- [What is a highly available (HA) control plane?](#what-is-a-highly-available-ha-control-plane)
  - [Stacked etcd](#stacked-etcd)
  - [External etcd](#external-etcd)
  - [Common topologies used in production](#common-topologies-used-in-production)
    - [A. Multiple replicas, stacked etcd, with dynamic identities and local storage](#a-multiple-replicas-stacked-etcd-with-dynamic-identities-and-local-storage)
    - [B. Multiple replicas, stacked etcd, fixed identities, networked storage](#b-multiple-replicas-stacked-etcd-fixed-identities-networked-storage)
    - [C. Multiple replicas, external etcd, with dynamic identities](#c-multiple-replicas-external-etcd-with-dynamic-identities)
    - [D. Single replica, stacked or external etcd, with fixed identity and networked storage](#d-single-replica-stacked-or-external-etcd-with-fixed-identity-and-networked-storage)
  - [Common features of HA control planes](#common-features-of-ha-control-planes)
    - [Fixed API Endpoint](#fixed-api-endpoint)
- [How are Kubernetes control planes deployed?](#how-are-kubernetes-control-planes-deployed)
  - [Machine-based](#machine-based)
  - [Pod-based](#pod-based)
  - [Managed](#managed)
- [What is the lifecycle of a control plane?](#what-is-the-lifecycle-of-a-control-plane)
  - [Create](#create)
  - [Scale Up](#scale-up)
  - [Scale Down](#scale-down)
  - [Upgrade](#upgrade)
  - [Delete](#delete)
  - [Repair partial outage](#repair-partial-outage)
  - [Take backup of etcd state](#take-backup-of-etcd-state)
  - [Restore control plane after complete outage](#restore-control-plane-after-complete-outage)
- [What is the state of control plane management in Cluster API?](#what-is-the-state-of-control-plane-management-in-cluster-api)
- [What control plane management events can Cluster API support?](#what-control-plane-management-events-can-cluster-api-support)

## What is the Kubernetes control plane?

The Kubernetes control plane is, at its core, kube-apiserver and etcd. If either of these are unavailable, no API requests can be handled. This impacts not only core Kubernetes APIs, but APIs implemented with CRDs. Other components, like kube-scheduler and kube-controller-manager, are also important, but do not have the same impact on availability.

### Where is the control plane state?

All control plane state is stored in etcd. Other components are stateless.

If the etcd data on disk is lost or corrupted, the etcd state must be recovered from an etcd backup. The backup is likely to be outdated. Nevertheless, the control plane will reconcile this outdated state. This is one reason to avoid having to recover from a backup.

A common way to hedge against corruption of the on-disk state is to run an etcd cluster, where each member has its own on-disk state. On the other hand, running an etcd cluster requires great care: parameters like the leader election timeout may need to be tuned, and membership may need to be reconfigured.

## What is a highly available (HA) control plane?

An HA control plane is one with a topology that minimizes the time to recovery from a control plane failure, e.g., a kube-apiserver or etcd process crashing, or a network partition. Many HA topologies are designed to have a recovery time on the order of seconds. There are many possible topologies, but most fall under these categories:

### Stacked etcd

This topology co-locates the stateless (kube-apiserver, etc.) and stateful (etcd) components.

Advantages include simplicity, lower overhead for smaller clusters, and a fixed etcd endpoint (localhost) for kube-apiserver. With [etcd learners](https://etcd.io/docs/v3.3.12/learning/learner/), replicas need not be etcd members, but can become etcd members quickly, improving the time to recover from replica failure.

### External etcd

This topology separates the stateless and stateful components.

Advantages include the ability to scale kube-apiserver to handle more requests without increasing the etcd cluster size, or to have an even number of kube-apiserver replicas. These advantages become less relevant when using [etcd learners](https://etcd.io/docs/v3.3.12/learning/learner/).

For more information, see the [kubeadm HA topology docs](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/ha-topology/).

### Common topologies used in production

#### A. Multiple replicas, stacked etcd, with dynamic identities and local storage

 Each replica acquires a dynamic network identity and uses local etcd storage. This topology requires the etcd cluster to be reconfigured as replicas are added, deleted, or fail. The [Cluster API AWS Provider](sigs.k8s.io/cluster-api-provider-aws/) uses this topology, as do many other Cluster API providers.

#### B. Multiple replicas, stacked etcd, fixed identities, networked storage

Each replica acquires one in set of fixed network identities and networked volumes for etcd state. This topology assumes networked storage and an API to reassign fixed network identities to IPs (e.g. using DNS SRV records), and a common API endpoint. The [kops](https://github.com/kubernetes/kops) project uses this topology.

#### C. Multiple replicas, external etcd, with dynamic identities

Each replica acquires a dynamic network identity. Storage is not relevant, because the etcd cluster is external. The [CoreOS Tectonic](https://coreos.com/tectonic/docs/latest/troubleshooting/etcd-nodes.html) product used this topology, and there is at least [one example](https://banzaicloud.com/blog/etcd-multi/) of a product sharing one external etcd cluster among multiple Kubernetes control planes.

#### D. Single replica, stacked or external etcd, with fixed identity and networked storage

The replica acquires a fixed network identity and a networked volume for etcd storage. This topology assumes networked storage and the ability to detect a failure, start a new replica, and mount a volume, all within a few seconds. It is only practical for Pod-based (or equivalent) control planes. Note that the single copy of etcd state--even with replicated storage--can be destroyed by corruption at the application or filesystem level. This is is the topology used by GKE "Zonal clusters" (i.e. clusters in a single availability zone), and the [Gardener](https://gardener.cloud) project.

### Common features of HA control planes

#### Fixed API Endpoint

Fixed API endpoint. This ensures that clients can reach a replacement kube-apiserver replica. In practice, the endpoint might be a load balancer, or virtual IP, or domain name.

An alternative to the fixed API endpoint is a fixed endpoint for a source of truth that clients can use. For an example of this, see the [kube-provision](https://github.com/moshloop/kube-provision) project.

Either way, there needs to be some fixed endpoint.

For a more detailed discussion, see the [Load Balancer Provider CAEP](https://docs.google.com/document/d/17Z_F_lmv4WgXaG9TaayOwwpCGRRoBxLbY070TSXDhvs/edit) draft.

## How are Kubernetes control planes deployed?

The Cluster API project maintainers have noted three common deployments.

### Machine-based

Every kube-apiserver replica (often co-located with etcd, kube-scheduler, and kube-controller-manager replicas) runs on one or more machines. The kops project and most Cluster API Providers use this deployment.

### Pod-based

Every kube-apiserver, etcd, kube-scheduler, and kube-controller-manager replica runs in one or more Pods. The [Gardener](https://gardener.cloud) project uses this deployment.

### Managed

The details of how the control plane is deployed are obscured from the user, and the user does not have access to the underlying resources. Services like AKS, EKS, and GKE use this deployment.

## What is the lifecycle of a control plane?

Here's an end-to-end example that covers the possible lifecycle events:

1. The control plane is created.
1. To handle an increase in API requests or tolerate more replica failures, the control plane is scaled up.
1. To save costs, the control plane is scaled down.
1. To handle a failed control plane replica, the control plane is reconfigured.
1. To support a newer version of Kubernetes, each control plane replica is replaced, in a rolling sequence, by a control plane replica with newer packages. (In practice, replicas can be upgraded in place, but Cluster API has decided to support replace upgrades only.)
1. The control plane is deleted.

Let's look in more detail at each event. For simplicity, the following assumes topology A.

### Create

1. Create a fixed API endpoint.
2. For each replica:
    1. Create the replica.
    1. Add the replica to the fixed API endpoint metadata.

### Scale Up

1. For each new replica:
    1. Create the replica.
    1. Add the replica to the fixed API endpoint metadata.

### Scale Down

1. For each replica to be deleted:
    1. Remove the replica from the fixed API endpoint metadata.
    2. Delete the replica.

### Upgrade

1. For each replica to be upgraded:
   1. Scale up with a "new" replica
   2. Scale down with an "old" replica

### Delete

1. For each replica:
   1. Remove the replica from the fixed API endpoint metadata.
   2. Delete the replica.
2. Delete the fixed API endpoint.

### Repair partial outage

1. For each failed replica:
    1. Using the etcd membership API, remove the etcd member corresponding to the replica.
    1. Remove the replica from the fixed API endpoint metadata.
    2. Delete the replica, if necessary.

### Take backup of etcd state

1. For one replica:
    1. Using the etcd snapshot API, take a snapshot
    2. Store the snapshot in a well-known location

### Restore control plane after complete outage

1. Obtain etcd state from an etcd snapshot, or from the disk of an existing replica.
1. For each existing replica:
    1. Remove the replicas from the fixed API endpoint metadata.
    2. Delete the replica.
1. For the first new replica:
    1. Create the replica.
    1. Initialize etcd with etcd state
    2. Add the replica to the fixed API endpoint metadata.
1. For each additional new replica:
    1. Create the replica.
    1. Add the replica to the fixed API endpoint metadata.

## What is the state of control plane management in Cluster API?

Initially, all providers deployed a single control plane replica. Some providers now deploy multiple control plane replicas. From the point of view of Cluster API controllers, each control plane replica is independent.

Providers rely on kubeadm to deploy multiple control plane replicas. The providers only need to wait for the first control plane replica to be available before deploying the others. For example, when that replica is ready, the provider annotates the Cluster object.

The Cluster API controllers have no logic for the managing control plane.

## What control plane management events can Cluster API support?

When the "management" and "workload" clusters are separate, Cluster API can support any of the control plane lifecycle events above. When they are the same, then Cluster API will be unavailable whenever the control plane is unavailable. In that case, Cluster API will be unable to restore the control plane after a complete outage, but it should be able to support everything else.