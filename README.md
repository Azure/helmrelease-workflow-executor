# orkestra-workflow-executor

[![Build Status](https://dev.azure.com/azure/Orkestra/_apis/build/status/nitishm.orkestra-workflow-executor?branchName=main)](https://dev.azure.com/azure/Orkestra/_build/latest?definitionId=100&branchName=main)

Azure Orkestra's default workflow executor

## Functionality

The default executor is responsible for deploying the `HelmRelease` object passed in as an input parameter to the docker container. The `HelmRelease` is represented by a `base64` encoded YAML string. The executor deploys, watches and polls for the status of the deployed `HelmRelease` until it either succeeds/fails or it times out. The timeout interval is configurable with the default set to 5m.

## CLI options

```shell
  $ make build
  $ ./bin/executor --help
Usage of /var/folders/g1/9bc3h33n3rb70fbxp_t1v4dc0000gn/T/go-build274703125/b001/e
  -spec string
        Spec of the helmrelease object to apply
  -timeout string
        Timeout for the execution of the argo workflow task (default "5m")
```
