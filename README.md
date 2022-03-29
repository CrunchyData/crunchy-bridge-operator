# Crunchy Bridge Operator

## Building the Operator
**if you are using podman instead of docker set CONTAINER_ENGINE as podman** `export CONTAINER_ENGINE=podman`
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

- `make install run INSTALL_NAMESPACE=<your_target_namespace> `

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

## Test Database as a Service (DBaaS) on OpenShift  

The Crunchy Bridge Operator is integrated with the [Red Hat Database-as-a-Service (DBaaS) Operator](https://github.com/RHEcosystemAppEng/dbaas-operator) which allows application developers to import database instances and connect to the databases through the [Service Binding Operator](https://github.com/redhat-developer/service-binding-operator). More information can be found [here](https://github.com/RHEcosystemAppEng/dbaas-operator#readme).

Note that both the DBaaS Operator and Crunchy Bridge Operator should be installed through the [Operator Lifecyle Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager).


**1.** Check DBaaS Registration

If the DBaaS Operator has been deployed in the OpenShift Cluster, the Crunchy Bridge Operator automatically creates a cluster level [DBaaSProvider](https://github.com/RHEcosystemAppEng/dbaas-operator/blob/main/config/crd/bases/dbaas.redhat.com_dbaasproviders.yaml) custom resource (CR) object `crunchy-bridge-registration` to automatically register itself with the DBaaS Operator.

```
apiVersion: dbaas.redhat.com/v1alpha1
kind: DBaaSProvider
metadata:
  name: crunchy-bridge-registration
  labels:
    related-to: dbaas-operator
    type: dbaas-provider-registration
spec:
  provider:
    name: Red Hat DBaaS / Crunchy Bridge
    displayName: Crunchy Bridge managed PostgreSQL
    displayDescription: The Crunchy Bridge Fully Managed Postgres as a Service.
    icon:
      base64data: <>
      mediatype: image/png
  inventoryKind: CrunchyBridgeInventory
  connectionKind: CrunchyBridgeConnection
  credentialFields:
    - key: publicApiKey
      displayName: Public API Key
      type: string
      required: true
    - key: privateApiSecret
      displayName: Private API Secret
      type: maskedstring
      required: true
```
If the crunchy bridge Operator is undeployed with the OLM, the above registration CR gets cleaned up automatically.

**2.** Creating a Secret 

Administrator will first create the secret with Application ID and Application Secret, for creating a secret. See more, [API Reference](https://docs.crunchybridge.com/api/getting_started)

```
kubectl create secret generic crunchy-bridge-api-key  --from-literal="publicApiKey=<Application ID>"   --from-literal="privateApiSecret=<Application Secret>"   -n crunchy-bridge-operator-system
```
**3.** Creating  `CrunchyBridgeInventory` Custom Resource

Administrator will creates a [DBaaSInventory](https://github.com/RHEcosystemAppEng/dbaas-operator/blob/main/config/crd/bases/dbaas.redhat.com_dbaasinventories.yaml) CR for CrunchyBrige. 
The DBaaS Operator automatically creates a CrunchyBridgeInventory CR, and the crunchy-bridge Operator discovers the clusters and  instances, and sets the result in the CR status.

The CRs status will be updated with list of clusters, as seen below

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
**3.** Creating `CrunchyBridgeConnection` Custom resource

Now the application developer can create a [DBaaSConnection](https://github.com/RHEcosystemAppEng/dbaas-operator/blob/main/config/crd/bases/dbaas.redhat.com_dbaasconnections.yaml) CR
for connection to the Crunchy database instance using from the list of instances, the DBaaS Operator automatically creates CrunchyBridgeConnection 
CR. The crunchy Operator stores the db user credentials in a kubernetes secret, and the remaining connection information in a configmap, and then updates the CrunchyBridgeConnection CR status.

The CRs status will be updated with connection details of specified instance ID as seen example:
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