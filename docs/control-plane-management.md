# Control Plane Management

## What is the Kubernetes control plane?

The Kubernetes control plane is, at its core, kube-apiserver and etcd. If either of these are unavailable, no API requests can be handled. This impacts not only core Kubernetes APIs, but APIs implemented with CRDs. Other components, like kube-scheduler and kube-controller-manager, are also important, but do not have the same impact on availability.

### Where is the control plane state?

All control plane state is stored in etcd. Other components are stateless.

If the etcd data on disk is lost or corrupted, the etcd state must be recovered from an etcd backup. The backup is likely to be outdated. Nevertheless, the control plane will reconcile this outdated state. This is one reason to avoid having to recover from a backup.

A common way to hedge against corruption of the on-disk state is to run an etcd cluster, where each member has its own on-disk state. On the other hand, running an etcd cluster requires great care: parameters like the leader election timeout may need to be tuned, and membership may need to be reconfigured.

## What is a highly available (HA) control plane?

An HA control plane is one with a topology that minimizes the time to recovery from a control plane failure, e.g., a kube-apiserver or etcd process crashing, or a network partition. Many HA topologies are designed to have a recovery time on the order of seconds. There are many possible topologies, but here are a few common ones:

### Multiple replicas, fixed identities and networked storage

Each replica acquires one in set of fixed network identities and networked volumes for etcd state. This topology assumes networked storage and an API to reassign fixed network identities to IPs (e.g. using DNS SRV records), and a common API endpoint. The [kops](https://github.com/kubernetes/kops) project uses this topology.

### Multiple replicas, with dynamic identities and local storage

 Each replica acquires a dynamic network identity and uses local etcd storage. This topology assumes a common API endpoint. The [Cluster API AWS Provider](sigs.k8s.io/cluster-api-provider-aws/) uses this topology, as do many other providers.

### Single replica, with fixed identity and networked storage

The replica acquires a fixed network identity and a networked volume for etcd storage. This topology assumes networked storage and the ability to detect a failure, start a new replica, and mount a volume, all within a few seconds. It is only practical for Pod-based (or equivalent) control planes. To handle more API requests, it's possible to use multiple kube-apiserver replicas while using a single etcd replica. Note that the single copy of etcd state--even with replicated storage--can be destroyed by corruption at the application or filesystem level. This is is the topology used by the [Gardener](https://gardener.cloud) project.

### Common features of HA control planes

#### Fixed API Endpoint

Fixed API endpoint. This ensures that clients can reach a replacement kube-apiserver replica. In practice, the endpoint might be a load balancer, or virtual IP, or domain name.

An alternative to the fixed API endpoint is a fixed endpoint for a source of truth that clients can use. For an example of this, see the [kube-provision](https://github.com/moshloop/kube-provision) project.

Either way, there needs to be some fixed endpoint.

Related: The [Load Balancer Provider CAEP](https://docs.google.com/document/d/17Z_F_lmv4WgXaG9TaayOwwpCGRRoBxLbY070TSXDhvs/edit) draft.

<!-- 2. Etcd cluster. This ensures that etcd is not a single point of failure. In practice, etcd replicas are either co-located with kube-apiserver replicas (sometimes referred to as "stacked etcd"), or are located outside of the Kubernetes cluster ("external etcd"). -->

## How are Kubernetes control planes deployed?

The Cluster API project maintainers have noted three common deployments.

### Machine-based

Every kube-apiserver replica (often co-located with etcd, kube-scheduler, and kube-controller-manager replicas) runs on one or more machines. The kops project and most Cluster API Providers use this deloyment.

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

### Repair

1.

### Upgrade

1. For each replica to be upgraded:
   1. Scale up with a "new" replica
   2. Scale down with an "old" replica

### Delete

1. For each replica:
   1. Remove the replica from the fixed API endpoint metadata.
   2. Delete the replica.
2. Delete the fixed API endpoint.
