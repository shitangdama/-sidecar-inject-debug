# README

Sidecar-Inject is a simple server to inject containers into Pod By `CRD`
It's easy to use and also offer container selector ability.

## Prerequisites

Kubernetes 1.9.0 or above with the `admissionregistration.k8s.io/v1beta1` API enabled. Verify that by the following command:
```
kubectl api-versions | grep admissionregistration.k8s.io/v1beta1
```
The result should be:
```
admissionregistration.k8s.io/v1beta1
```

In addition, the `MutatingAdmissionWebhook` admission controllers should be added and listed in the correct order in the admission-control flag of kube-apiserver.

## Build Image

`Debug-Sidecar-inject` is managed by `go mod`

run `build.sh` to build docker image

## Deploy

1. Create a signed cert/key pair and store it in a Kubernetes `secret` that will be consumed by sidecar deployment

```
./deployment/webhook-create-signed-cert.sh \
    --service perf-sidecar-injector-webhook-svc \
    --secret perf-sidecar-injector-webhook-certs \
    --namespace default
```

2. Patch the `MutatingWebhookConfiguration` by setting `caBundle` with correct value from Kubernetes cluster
```
cat deployment/mutatingwebhook.yaml | \
    deployment/webhook-patch-ca-bundle.sh > \
    deployment/mutatingwebhook-ca-bundle.yaml
```

3. Deploy resources
```
kubectl create -f deployment/crd.yaml
kubectl create -f deployment/deployment.yaml
kubectl create -f deployment/service.yaml
kubectl create -f deployment/rbac.yaml
kubectl create -f deployment/mutatingwebhook-ca-bundle.yaml
```

## Verify

1. The sidecar inject webhook should be running
```
$ kubectl get pods
NAME                                                  READY     STATUS    RESTARTS   AGE
sidecar-injector-webhook-deployment-76ddb9f7f6-72pp8   1/1       Running   0          5m

$ kubectl get deployment
NAME                                  DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
sidecar-injector-webhook-deployment   1         1         1            1           5m
```

2. Label the default namespace with `sidecar-injector=enabled`
```
$ kubectl label namespace default sidecar-injector=enabled
$ kubectl get namespace -L sidecar-injector
NAME          STATUS    AGE       SIDECAR-INJECTOR
default       Active    18h       enabled
kube-public   Active    18h
kube-system   Active    18h
```

3. Deploy an app in Kubernetes cluster, take `nginx` app as an example

```
kubectl create -f example/nginx.yml
```

NAME                                                   READY   STATUS        RESTARTS   AGE
nginx-deployment-54f57cf6bf-g7q59                      1/1     Running       0          18m
sidecar-injector-webhook-deployment-76ddb9f7f6-72pp8   1/1     Running       0          20s

3. Deploy an app in Kubernetes cluster, take `debuger` crd as an example

```
kubectl create -f example/sidecar.yml


NAME                                                   READY   STATUS    RESTARTS   AGE
nginx-deployment-66fb44b8fd-jr64b                      2/2     Running   0          49s
sidecar-injector-webhook-deployment-76ddb9f7f6-72pp8   1/1     Running   0          78s
```

get debuger crd
```
kubectl get debugers

NAME              AGE
test-sidecarset   18m
```