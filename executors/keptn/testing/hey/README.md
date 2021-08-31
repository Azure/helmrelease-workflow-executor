# Webserver quality gates

## Create the project

keptn create project webserver --shipyard=./shipyard.yaml

## Onboard service to project

keptn onboard service webserver --project=webserver

## Add resource SLI

keptn add-resource --project=webserver --service=webserver --resource=prometheus/sli.yaml  --resourceUri=prometheus/sli.yaml --all-stages

## Add resource SLO

keptn add-resource --project=webserver --service=webserver --resource=slo.yaml --resourceUri=slo.yaml --all-stages

## Configure prometheus sli provider

keptn configure monitoring prometheus --project=webserver --service=webserver

## Start testing

### Run load tester (manually)

hey -z 1h -n -1 http://$(kubectl get svc -n orkestra webserver -ojsonpath='{.status.loadBalancer.ingress[0].ip}')/hello

### Trigger load testing (event)

keptn add-resource --project=webserver --service=webserver --resource=config.yaml --resourceUri=job/config.yaml --all-stages

where, `config.yaml` contains,

```yaml
apiVersion: v2
actions:
  - name: "Run hey"
    events:
      - name: "sh.keptn.event.test.triggered"
    tasks:
      - name: "Run hey load tests"
        image: "azureorkestra/hey"
        cmd: "hey -z 1m http://webserver.orkestra.svc.cluster.local:80/hello"
```

### Trigger evaluation

keptn trigger evaluation --project=webserver --service=webserver --timeframe=5m

---

```console
keptn create project hey --shipyard=./shipyard.yaml                     
keptn create service webservers --project=hey      
keptn configure monitoring prometheus --project=hey --service=webservers
keptn add-resource --project=hey --service=webservers --resource=slo.yaml --resourceUri=slo.yaml --stage=dev
keptn add-resource --project=hey --service=webservers --resource=prometheus/sli.yaml  --resourceUri=prometheus/sli.yaml --stage=dev
keptn add-resource --project=hey --service=webservers --resource=job/config.yaml  --resourceUri=job/config.yaml --stage=dev

keptn send event -f hey/event.json 
```

