ARTIFACT_ID=ces-importer
MAKEFILES_VERSION=9.9.1
VERSION=0.0.1

GOTAG=1.24.2
.DEFAULT_GOAL:=help

IMAGE=cloudogu/${ARTIFACT_ID}:${VERSION}

K8S_RESOURCE_DIR=${WORKDIR}/k8s
K8S_COMPONENT_SOURCE_VALUES = ${HELM_SOURCE_DIR}/values.yaml
K8S_COMPONENT_TARGET_VALUES = ${HELM_TARGET_DIR}/values.yaml
HELM_PRE_GENERATE_TARGETS = helm-values-update-image-version
HELM_POST_GENERATE_TARGETS = helm-values-replace-image-repo template-stage template-log-level template-image-pull-policy template-importer-public-key
CHECK_VAR_TARGETS=check-all-vars
IMAGE_IMPORT_TARGET=image-import
IMAGE=ces-exporter:${VERSION}

include build/make/variables.mk
include build/make/dependencies-gomod.mk
include build/make/build.mk
include build/make/test-common.mk
include build/make/test-unit.mk
include build/make/static-analysis.mk
include build/make/clean.mk
include build/make/mocks.mk
include build/make/release.mk
include build/make/self-update.mk
include build/make/k8s-component.mk

.PHONY: mocks
mocks: ${MOCKERY_BIN} ${MOCKERY_YAML} ## target is used to generate mocks for all interfaces in a project.
	${MOCKERY_BIN}
	@echo "Mocks successfully created."

.PHONY: helm-values-update-image-version
helm-values-update-image-version: $(BINARY_YQ)
	@echo "Updating the image version in source values.yaml to ${VERSION}..."
	@$(BINARY_YQ) -i e ".main.image.tag = \"${VERSION}\"" ${K8S_COMPONENT_SOURCE_VALUES}
	@$(BINARY_YQ) -i e ".job.image.tag = \"${VERSION}\"" ${K8S_COMPONENT_SOURCE_VALUES}

.PHONY: helm-values-replace-image-repo
helm-values-replace-image-repo: $(BINARY_YQ)
	@if [[ ${STAGE} == "development" ]]; then \
      		echo "Setting dev image repo in target values.yaml!" ;\
    		$(BINARY_YQ) -i e ".main.image.registry=\"$(shell echo '${IMAGE_DEV}' | sed 's/\([^\/]*\)\/\(.*\)/\1/')\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
    		$(BINARY_YQ) -i e ".job.image.registry=\"$(shell echo '${IMAGE_DEV}' | sed 's/\([^\/]*\)\/\(.*\)/\1/')\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
    		$(BINARY_YQ) -i e ".main.image.repository=\"$(shell echo '${IMAGE_DEV}' | sed 's/\([^\/]*\)\/\(.*\)/\2/')\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
    		$(BINARY_YQ) -i e ".job.image.repository=\"$(shell echo '${IMAGE_DEV}' | sed 's/\([^\/]*\)\/\(.*\)/\2/')\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
    	fi

.PHONY: template-stage
template-stage: $(BINARY_YQ)
	@if [[ ${STAGE} == "development" ]]; then \
  		echo "Setting STAGE env in deployment to ${STAGE}!" ;\
		$(BINARY_YQ) -i e ".env.stage=\"${STAGE}\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
	fi

.PHONY: template-log-level
template-log-level: ${BINARY_YQ}
	@if [[ "${STAGE}" == "development" ]]; then \
      echo "Setting LOG_LEVEL env in deployment to ${LOG_LEVEL}!" ; \
      $(BINARY_YQ) -i e ".env.logLevel=\"${LOG_LEVEL}\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
    fi

.PHONY: template-image-pull-policy
template-image-pull-policy: $(BINARY_YQ)
	@if [[ "${STAGE}" == "development" ]]; then \
          echo "Setting pull policy to always!" ; \
          $(BINARY_YQ) -i e ".main.imagePullPolicy=\"Always\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
          $(BINARY_YQ) -i e ".job.imagePullPolicy=\"Always\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
    fi

.PHONY: template-importer-public-key
template-importer-public-key: $(BINARY_YQ)
	@if [[ "${STAGE}" == "development" ]]; then \
          echo "Setting importer-public-key from environment-variable 'IMPORTER_PUBLIC_KEY'" ; \
          $(BINARY_YQ) -i e ".publicKey.data=\"${IMPORTER_PUBLIC_KEY}\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
    fi

.PHONY: apikey-secret
apikey-secret: $(BINARY_YQ) ## generates a K8s secret for the API key from an environment variable
	@kubectl delete secret ces-exporter-secret || true
	@kubectl delete secret ces-importer-secret || true
	@kubectl create secret generic ces-exporter-secret --from-literal=apiKey=${EXPORTER_API_KEY} --namespace="${NAMESPACE}" --context="${KUBE_CONTEXT_NAME}"
	@kubectl create secret generic ces-importer-secret --from-file=${IMPORTER_SSH_KEY_FILE} --namespace="${NAMESPACE}" --context="${KUBE_CONTEXT_NAME}"

.PHONY: helm-apply-dev
helm-apply-dev:
	@sed -i -E "s/(^VERSION=[[:digit:]].[[:digit:]].[[:digit:]])/\1-$$(date +%s)/g" Makefile
	@make helm-apply
	@sed -i -E "s/(^VERSION=[[:digit:]].[[:digit:]].[[:digit:]])-.*/\1/g" Makefile
	@sed -i -E "s/(tag: [[:digit:]].[[:digit:]].[[:digit:]])-.*/\1/g" k8s/helm/values.yaml