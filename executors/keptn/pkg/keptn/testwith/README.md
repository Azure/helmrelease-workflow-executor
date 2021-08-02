# Continuous evaluation of a webserver app using keptn & `hey`

## Setup

### Deploy webserver app

#### Without errors/delays

```console
helm upgrade --install webserver https://nitishm.github.io/charts/webserver-v1.0.0.tgz -n webserver --create-namespace
```

#### With errors/delays

This should set the error rate to 50% and the delay to 500ms.

```console
helm upgrade --install webserver https://nitishm.github.io/charts/webserver-v1.0.0.tgz -n webserver --create-namespace --set "webserver.errors.enabled=true" --set "webserver.delay.enabled=true"
```

## Deploy Orkestra with keptn controlplane

Follow the installation instructions [here](https://github.com/Azure/orkestra#installation-).

## Deploy keptn job-executor-service

```console
kubectl create -f job-executor-service.yaml -n orkestra
```

## Deploy prometheus

```console
helm install prometheus prometheus-community/prometheus -n prometheus --create-namespace
```

## Configure keptn to use prometheus for SLI/SLO quality gates

```console
kubectl create -f keptn-prometheus-service.yaml -n orkestra
```

## Webserver application [Webserver](https://github.com/nitishm/k8s-webserver-app)

### Create the project

```console
keptn create project webserver --shipyard=./shipyard.yaml
```

### Onboard service to project

```console
keptn onboard service webserver --project=webserver
```

### Add resource SLI

```console
keptn add-resource --project=webserver --service=webserver --resource=prometheus/sli.yaml  --resourceUri=prometheus/sli.yaml --all-stages
```

### Add resource SLO

```console
keptn add-resource --project=webserver --service=webserver --resource=slo.yaml --resourceUri=slo.yaml --all-stages
```

### Configure prometheus sli provider

```console
keptn configure monitoring prometheus --project=webserver --service=webserver
```

### Continuous Testing & Evaluation

#### Run load tester (manually)

```console
hey -z 1h -n -1 http://$(kubectl get svc -n webserver webserver -ojsonpath='{.status.loadBalancer.ingress[0].ip}')/hello
```

#### Trigger load testing (event)

```console
keptn add-resource --project=webserver --service=webserver --resource=config.yaml --resourceUri=job/config.yaml --all-stages
```

where, `config.yaml` contains,

```yaml
apiVersion: v2
actions:
  - name: "Run hey"
    events:
      - name: "sh.keptn.event.test.triggered"
    tasks:
      - name: "Run hey load tests"
        image: "azurewebserver/hey"
        cmd: "hey -z 1m http://webserver.webserver.svc.cluster.local:80/hello"
```

#### Trigger evaluation

keptn trigger evaluation --project=webserver --service=webserver --timeframe=5m

---

### TL;DR

```console
keptn create project hey --shipyard=./shipyard.yaml                     
keptn create service webservers --project=hey      
keptn configure monitoring prometheus --project=hey --service=webservers
keptn add-resource --project=hey --service=webservers --resource=slo.yaml --resourceUri=slo.yaml --stage=dev
keptn add-resource --project=hey --service=webservers --resource=prometheus/sli.yaml  --resourceUri=prometheus/sli.yaml --stage=dev
keptn add-resource --project=hey --service=webservers --resource=job/config.yaml  --resourceUri=job/config.yaml --stage=dev

kkeptn trigger evaluation --project=hey --service=webservers --timeframe=5m --stage dev --start $(date -u +"%Y-%m-%dT%T")
```
