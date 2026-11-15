### possivelmente precisa fazer um port-forwarding
kubectl port-forward -n ricplt svc/service-ricplt-appmgr-http 8080:8080

### precisa rodar o chartmuseum:
chartmuseum --debug --port=8090 --storage=local --storage-local-rootdir=$HOME/xapp_dev/chartmuseum-storage

### empacota o chart
helm package chart

### manda o chart pro chart museum
curl -sS --data-binary "@xapp-sctp-1.0.0.tgz" http://localhost:8090/api/charts

### verifica se o chart está no museum
curl -sS http://localhost:8090/index.yaml

### verifica se o cluster alcança o chat museum
kubectl run curl --rm -it --restart=Never -n ricplt --image=curlimages/curl -- curl -sS http://200.18.102.25:8090/health

### descobre o nome do configmap 
kubectl get configmap -n ricplt | grep -i appmgr

### edita o configmap para o IP do charmuseum
kubectl get configmap configmap-ricplt-appmgr-appconfig -n ricplt -o yaml

### reinicia o appmgr
kubectl rollout restart -n ricplt deploy/deployment-ricplt-appmgr
