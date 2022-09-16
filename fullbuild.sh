export VERSION=0.0.2
export CHANNEL=develop
export BUNDLE_CHANNELS=--channels=${CHANNEL}
export BUNDLE_DEFAULT_CHANNEL=--default-channel=${CHANNEL}
export IMAGE_TAG_BASE=ghcr.io/infra-mgmt-io/perms
export BUNDLE_METADATA_OPTS="${BUNDLE_CHANNELS} ${BUNDLE_DEFAULT_CHANNEL}"
export BUNDLE_IMG=${IMAGE_TAG_BASE}-bundle:v${VERSION}
export BUNDLE_GEN_FLAGS="-q --overwrite --version ${VERSION} ${BUNDLE_METADATA_OPTS}"
export IMG=ghcr.io/infra-mgmt-io/perms:v${VERSION}
export CATALOG_IMG="${IMAGE_TAG_BASE}-catalog:v${VERSION}"
export FROM_INDEX_OPT="--from-index ${CATALOG_IMG}"
export WORKINGDIR=`pwd`
export LOCALBIN=${WORKINGDIR}/bin/
export KUSTOMIZE_INSTALL_SCRIPT="https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
export KUSTOMIZE_VERSION=4.5.7
export CONTROLLER_TOOLS_VERSION=v0.9.2

rm -f ${LOCALBIN}/kustomize
curl -s ${KUSTOMIZE_INSTALL_SCRIPT} | bash -s -- ${KUSTOMIZE_VERSION} ${LOCALBIN}
rm -f ${LOCALBIN}/controller-gen
GOBIN=${LOCALBIN} go install sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_TOOLS_VERSION}

env | grep BUNDLE
env | grep IMG

${LOCALBIN}/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
${LOCALBIN}/controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases 
go build -o bin/manager main.go
docker build -t ${IMG} .
docker push ${IMG}
operator-sdk generate kustomize manifests -q
cd config/manager && ${LOCALBIN}kustomize edit set image controller=${IMG}
cd ../../
${LOCALBIN}kustomize build config/manifests | operator-sdk generate bundle ${BUNDLE_GEN_FLAGS}
operator-sdk bundle validate ./bundle
docker build -f bundle.Dockerfile -t ${BUNDLE_IMG} .
docker push ${IMAGE_TAG_BASE}-bundle:v${VERSION}
${LOCALBIN}opm index add --container-tool docker --mode semver --tag ${CATALOG_IMG} --bundles ${BUNDLE_IMG} 
docker push ${CATALOG_IMG}