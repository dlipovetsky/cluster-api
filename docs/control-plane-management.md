# Control Plane Management

## What is the Kubernetes control plane?

The Kubernetes control plane is, at its core, kube-apiserver and etcd. If either of these are unavailable, no API requests can be handled. This impacts not only core Kubernetes APIs, but APIs implemented with CRDs. Other components, like kube-scheduler and kube-controller-manager, are also important, but do not have the same impact on availability.

### Where is the control plane state?

All control plane state is stored in etcd. Other components are stateless.

If the etcd data on disk is lost or corrupted, the etcd state must be recovered from an etcd backup. The backup is likely to be outdated. Nevertheless, the control plane will reconcile this outdated state. This is one reason to avoid having to recover from a backup.

A common way to hedge against corruption of the on-disk state is to run an etcd cluster, where each member has its own on-disk state. On the other hand, running an etcd cluster requires great care: parameters like the leader election timeout may need to be tuned, and membership may need to be reconfigured.

## What is a highly available (HA) control plane?

A control plane deployed in such a way as to minimize the time to recovery from a control plane failure, e.g., a kube-apiserver or etcd process crashing, or a network partition. Many HA deployments have a recovery time on the order of seconds. There are many types of highly available control plane deployments. Here are a few common ones:

1. kops-style

2. Dynamically reconfigured.

3. Deploy a one replica of kube-apiserver, and one replica of etcd. Ensure both have fixed network identities. Write etcd data to networked, replicated storage. When a replica fails, restart it elsewhere. This deployment assumes you can detect a failure, start a replica, and mount a volume, all within a few seconds. On the other hand, it does not require an etcd cluster. Though it is replicated, there is still one copy of the etcd data, and if corrupted, etcd state must be recovered from a backup. I believe that early versions of GKE used this deployment.

### Common features of highly available control planes

1. Multiple kube-apiserver replicas. In practice, each replica is co-located with kube-scheduler and kube-controller-manager.
1. Well-defined API endpoint. This ensures that clients can reach a replacement kube-apiserver replica. In practice, the endpoint might be a load balancer, or virtual IP, or domain name.
1. Etcd cluster. This ensures that etcd is not a single point of failure. In practice, etcd replicas are either co-located with kube-apiserver replicas (sometimes referred to as "stacked etcd"), or are located outside of the Kubernetes cluster ("external etcd").

## How are Kubernetes control plane deployed?

In the wild, we have observed three ways of deploying control planes:

1. Machine-based.
2. Pod-based.
3. Black box.

## What is the lifecycle of a control plane?

<!--
Operations:

Deploy
Scale (symmetric... kubeadm reset)
Repair (asymmetric... no kubeadm reset possible)
Upgrade
-->

Cluster API is in a position to deploy, scale, repair, and upgrade machine-based and pod-pased control planes. It is also in a position to perform a subset of these for black box control planes.