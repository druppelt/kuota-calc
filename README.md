![ci](https://github.com/druppelt/kuota-calc/workflows/ci/badge.svg)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/druppelt/kuota-calc)
[![Go Report Card](https://goreportcard.com/badge/github.com/druppelt/kuota-calc)](https://goreportcard.com/report/github.com/druppelt/kuota-calc)
![License](https://img.shields.io/github/license/druppelt/kuota-calc)

> [!NOTE]
> This is a fork of [postfinance/kuota-calc](https://github.com/postfinance/kuota-calc) that adds some features as the original doesn't seem to be maintained.
> But neither will this. After I added what I need, I will stop development as well. Feel free to fork as well :)

# kuota-calc
Simple utility to calculate the maximum needed resource quota for deployment(s). kuota-calc takes the
deployment strategy, replicas and all containers into account, see [supported-resources](https://github.com/druppelt/kuota-calc#supported-k8s-resources) for a list of kubernetes resources which are currently supported by kuota-calc.

## Motivation
In shared environments such as kubernetes it is always a good idea to isolate/constrain different workloads to prevent them from infering each other. Kubernetes provides [Resource Quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/) to limit compute, storage and object resources of namespaces.

Calculating the needed compute resources can be a bit challenging (especially with large and complex deployments) because we must respect certain settings/defaults like the deployment strategy, number of replicas and so on. This is where kuota-calc can help you, it calculates the maximum needed resource quota in order to be able to start a deployment of all resources at the same time by respecting deployment strategies, replicas and so on.

## Example
```bash
$ cat examples/deployment.yaml | kuota-calc -detailed
Version    Kind           Name     Replicas    Strategy         MaxReplicas    CPURequest    CPULimit    MemoryRequest    MemoryLimit
apps/v1    Deployment     myapp    10          RollingUpdate    11             2750m         5500m       704Mi            2816Mi
apps/v1    StatefulSet    myapp    3           RollingUpdate    3              750m          3           6Gi              12Gi

Total
CPU Request: 3500m
CPU Limit: 8500m
Memory Request: 6848Mi
Memory Limit: 15104Mi
```

## Installation
Pre-compiled statically linked binaries are available on the [releases page](https://github.com/druppelt/kuota-calc/releases).

kuota-calc can either be used as a kubectl plugin or invoked directly. If you intend to use kuota-calc as
a kubectl plugin, simply place the binary anywhere in `$PATH` named `kubectl-kuota_calc` with execute permissions.
For further information, see the offical documentation on kubectl plugins [here](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/).

## supported k8s resources
**kuota-calc is still a work-in progress**, there are plans to support more k8s resources (see [#5](https://github.com/druppelt/kuota-calc/issues/5) for more info). 

Currently supported:

- apps/v1 Deployment
- apps/v1 StatefulSet
- apps/v1 DaemonSet
- batch/v1 CronJob
- batch/v1 Job
- v1 Pod
