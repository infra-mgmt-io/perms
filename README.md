# Installation
- k3d
- operator-sdk (brew)

# Doc
https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/

# Project init
````
operator-sdk init --domain infra-mgmt.io --repo github.com/infra-mgmt-io/perms
````

# Create API
````
operator-sdk create api --group perms --version v1beta1 --kind PermsRoleBinding --resource --controller
````

## Generate objects
````
make generate
make manifests
````

## Build
````
make docker-build docker-push IMG="docker.io/chrisautomit/operator-perms:v0.0.2"
````

## Run
````
make deploy IMG="docker.io/chrisautomit/operator-perms:v0.0.2"
GOBIN=/Users/chris/Documents/git/bm/operator/go/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0
/Users/chris/Documents/git/bm/operator/go/bin/controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
cd config/manager && /Users/chris/Documents/git/bm/operator/go/bin/kustomize edit set image controller=controller:latest
/Users/chris/Documents/git/bm/operator/go/bin/kustomize build config/default | kubectl apply -f -
namespace/go-system created
customresourcedefinition.apiextensions.k8s.io/permissions.cache.automit.de configured
serviceaccount/go-controller-manager created
role.rbac.authorization.k8s.io/go-leader-election-role created
clusterrole.rbac.authorization.k8s.io/go-manager-role created
clusterrole.rbac.authorization.k8s.io/go-metrics-reader created
clusterrole.rbac.authorization.k8s.io/go-proxy-role created
rolebinding.rbac.authorization.k8s.io/go-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/go-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/go-proxy-rolebinding created
configmap/go-manager-config created
service/go-controller-manager-metrics-service created
deployment.apps/go-controller-manager created
````

k config set-context --current --namespace permissions-operator

k apply -f config/samples/perms_v1beta1_permsrolebinding.yaml 

