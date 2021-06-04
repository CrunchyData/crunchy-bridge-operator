# Crunchy Bridge Operator

## Building the Operator
- Build Operator `make build`
- For build and push the operator image, operator-bundle image and operator-catalog image to a registry run:  `make release`**make sure change the registry under -  `Makefile` and set `ORG` with your own Quay.io Org!**
- `make release` is commented out while the release process is being considered
- For more make commands run `make help`
- see [operator-sdk documentation](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/) for further info

## Running the Operator

### Prerequisite Tools

* golang 1.15+
* operator-sdk v1.7 or later
* (OCP 4.6.1) or later 
* [OpenShift command line tool](https://developers.redhat.com/openshift/command-line-tools)

**Run as a local instance**:

- `make install run WATCH_NAMESPACE=<your_target_namespace>`

**Deploy & run on a cluster:**
- `oc project <your_target_namespace>`
- `make deploy`
- When finished, remove deployment via:
    - `make  undeploy`

**Deploy via OLM on cluster:**
- **Make sure to edit `Makefile` and set `ORG` with your own Quay.io Org!**
- **Next edit the [catalog-source.yaml](config/samples/catalog-source.yaml) template to indicate your new Quay.io org image**  
- `make release catalog-update`
- search the Crunchy Bridge Operator in OperatorHub, click on install.

## Enable DBaaS Integration

Follow the steps below to enable integration with DBaaS.

Note: you need  Application ID and Application Secret, for creating a secret. See more, [API Reference](https://docs.crunchybridge.com/api/getting_started)

1. Create a Secret :
```
kubectl create secret generic crunchy-bridge-api-key  --from-literal="publicApiKey=<Application ID>"   --from-literal="privateApiSecret=<Application Secret>"   -n crunchy-bridge-operator-system
```
2. Create a `CrunchyBridgeInventory` Custom Resource
```
kubectl apply -f config/samples/dbaas.redhat.com_v1alpha1_crunchybridgeinventory.yaml
```


