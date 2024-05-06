# Run in k8s

## Create Image

Create image with `docker` or `podman` and make it available in some
registry. (My build can be retrieved from `ghcr.io/mnlipp/fritzdocsis:0.4.0`.)

## Create ConfigMap

```
kubectl -n monitoring create configmap fritz-docsis-config --from-literal=url=...
```

## Create Secret

```
kubectl -n monitoring create secret generic fritz-docsis-secret --from-literal=username=... --from-literal=password='...'
```

## Deploy fritzDocsis

Using this [descriptor](fritz-docsis-deployment.yaml).

## Deploy the Service

Using this [descriptor](fritz-docsis-service.yaml). Due to
the annotation `prometheus.io/scrape: "true"` the service should
be picked up by your default prometheus instance automatically.
