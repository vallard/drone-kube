# DRONE Kubernetes
This drone kubernetes plugin does the equivalent of: 

```
kubectl set image deploy/nginx-deployment nginx=nginx:sometag
```

```yaml
pipeline:
  kube:
    image: vallard/drone-kube
    namespace: default
    name: hottub
    containers: 
    	- name: hottub
    	  image: jamesbrown/hottub:{{BUILD_ID}}
    	- name: healthz
    	  image: some/example:{{BUILD_ID}}
   
```
## Secrets
The kube
	