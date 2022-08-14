# Permission Operator (perms)

[![Build Status](https://github.com/infra-mgmt-io/perms/actions/workflows/docker-build-and-publish.yml/badge.svg)](https://github.com/infra-mgmt-io/perms/actions/workflows/docker-build-and-publish.yml)
[![License](http://img.shields.io/:license-apache-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0.html)

----

## Overview

Permission Operator is an open source project for managing k8s bindings. It provides an custom configuration format on top of the rolebindings and clusterrolebindings. It is build to bring some extra information and a easy custom way of defining bindings.

----

## License

Operator SDK is under Apache 2.0 license. See the [LICENSE][license_file] file for details.

----

## To start developing Permission Operator

The [community repository] hosts all information about
building the operator from source, how to contribute code
and documentation, who to contact about what, etc.

If you want to build the Permission Operator right away there are two options:

##### You have a working [Go environment].

```
git clone https://github.com/infra-mgmt-io/perms.git
cd perms
make install run
```

##### You have a working [k8s environment].

```
git clone https://github.com/infra-mgmt-io/perms.git
cd perms
./hack.sh 0.0.1
```

----

## Usefull links
- [sdk-docs][https://sdk.operatorframework.io]
- [operator-framework-community][https://github.com/operator-framework/community]
- [operator-framework-communication][https://github.com/operator-framework/community#get-involved]
- [operator-framework-meetings][https://github.com/operator-framework/community#meetings]
- [operator-framework-quickstart][https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/]

----

## Project setup
Create the project directory and initialize the main project
````
operator-sdk init --domain infra-mgmt.io --repo github.com/infra-mgmt-io/perms
````

### Create APIs

#### PermsRoleBinding
````
operator-sdk create api --group perms --version v1beta1 --kind PermsRoleBinding --resource --controller
````

#### PermsClusterRoleBinding
````
operator-sdk create api --group perms --version v1beta1 --kind PermsClusterRoleBinding --resource --controller --namespaced=false
````

#### Generate objects
````
make generate
make manifests
````

#### Build
````
make docker-build docker-push IMG="docker.io/chrisautomit/operator-perms:v0.0.2"
````

#### Run
````
make deploy IMG="docker.io/chrisautomit/operator-perms:v0.0.2"
````

#### Configure k8s Namespace
````
k config set-context --current --namespace permissions-operator
````

#### Create samples
````
k apply -f config/samples/perms_v1beta1_permsclusterrolebinding.yaml
k apply -f config/samples/perms_v1beta1_permsclusterrolebinding_demo1.yaml
k apply -f config/samples/perms_v1beta1_permsrolebinding.yaml
k apply -f config/samples/perms_v1beta1_permsrolebinding_demo1.yaml
````

#### Check status
````
k get prb
NAME   USERS   GROUPS   SERVICEACCOUNTS   AVAILABLE   PROGRESSING   DEGRADED
edit   3       4        2                 True        False         False

k get pcrb
NAME                             USERS   GROUPS   SERVICEACCOUNTS   AVAILABLE   PROGRESSING   DEGRADED
permsclusterrolebinding-sample   3       6        2                 True        False         False
````
