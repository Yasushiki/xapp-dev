# xApp SCTP
Utiliza o protocolo SCTP para a conexão com o RIC

## Estado atual
O xApp consegue se conectar com a DU e recebe mensagens de volta. Ainda é necessário conectar uma UE para verificar se o xApp consegue receber as métricas e outras mensagens.

O xApp utiliza o wrapper de ASN.1 do kpimon-go, com pouca modificação.

Também utiliza bytes hardcoded gerados por `asn1/gerar_kpm.py` para a ActionDefitinion. `eventTriggerDefinition` indica o período entre relatórios. `actionDefBytes` indica o estilo e formato da ActionDefinition, filtra as UEs e indica quais métricas serão enviadas pela DU

---

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
