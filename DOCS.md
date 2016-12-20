# DRONE Kubernetes
This drone kubernetes plugin does the equivalent of: 

```
kubectl set image deploy/nginx-deployment nginx=nginx:sometag
```

```yaml
pipeline:
  kube:
    template: deployment.yaml
```
## Secrets
The kube
	