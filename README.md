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

## Testing the Inventory and Connection CRs  

Note: you need Application ID and Application Secret, for creating a secret. See more, [API Reference](https://docs.crunchybridge.com/api/getting_started)

1. Create a Secret :
```
kubectl create secret generic crunchy-bridge-api-key  --from-literal="publicApiKey=<Application ID>"   --from-literal="privateApiSecret=<Application Secret>"   -n crunchy-bridge-operator-system
```
2. Create a `CrunchyBridgeInventory` Custom Resource
```
kubectl apply -f config/samples/dbaas.redhat.com_v1alpha1_crunchybridgeinventory.yaml
```
The CRs status will be updated with list of clusters.

example: 
```
Instances:
    Extra Info:
      Cpu:            1
      created_at:     2021-04-01 19:36:19.937782 +0000 UTC
      is_ha:          false
      major_version:  13
      Memory:         2
      provider_id:    aws
      region_id:      us-east-1
      Storage:        100
      team_id:        vp6hlxjcl5g73furjiztcrr2vi
      updated_at:     2021-06-04 21:30:57.53937 +0000 UTC
    Instance ID:      475ow3natngrhaffymv7fbxmha
    Name:             sampledatabasse

```
3. Get cluster instance ID from Step 2 CR status and specify a cluster instance ID in `CrunchyBridgeConnection` Custom Resource and run the below command.
  ```
  kubectl apply -f config/samples/dbaas.redhat.com_v1alpha1_crunchybridgeconnection.yaml
 ``` 
The CRs status will be updated with connection details of specified instance ID. 

example:
```
  connectionInfoRef:
   name: crunchy-bridge-db-conn-cm-k8rkv // name of configmap contains connection info like host, port, datbase name
  credentialsRef:
   name: crunchy-bridge-db-credentials-vmml2 // name of secret contains username and password
```
## Links

* [Operator SDK](https://github.com/operator-framework/operator-sdk)
* [DBaaS Operator](https://github.com/RHEcosystemAppEng/dbaas-operator)
* [API Reference](https://docs.crunchybridge.com/api/getting_started)