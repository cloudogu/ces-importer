ARTIFACT_ID_IMPORTER=ces-importer
ARTIFACT_ID_JOB=${ARTIFACT_ID_IMPORTER}-migration-job

# set default to main application
ARTIFACT_ID=${ARTIFACT_ID_IMPORTER}

MAKEFILES_VERSION=10.5.0
VERSION=2.2.0

GOTAG=1.25.5
GO_BUILD_FLAGS?=-mod=vendor -a -tags netgo $(LDFLAGS) -installsuffix cgo -o $(BINARY) ./cmd/ces-importer
.DEFAULT_GOAL:=help

IMAGE=cloudogu/${ARTIFACT_ID}:${VERSION}

K8S_RESOURCE_DIR=${WORKDIR}/k8s
K8S_COMPONENT_SOURCE_VALUES = ${HELM_SOURCE_DIR}/values.yaml
K8S_COMPONENT_TARGET_VALUES = ${HELM_TARGET_DIR}/values.yaml
HELM_PRE_GENERATE_TARGETS = helm-values-update-image-version
HELM_POST_GENERATE_TARGETS = helm-values-replace-image-repo template-log-level template-image-pull-policy template-api-config template-migration-config template-smtp-config
CHECK_VAR_TARGETS=check-all-vars
IMAGE_IMPORT_TARGET=images-import

# docker
IMAGE=${ARTIFACT_ID}:${VERSION}


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
	@if [[ "${STAGE}" == "development" ]]; then \
		echo "Setting dev image repo in target values.yaml!" ;\
		echo "Component target values: ${IMAGE_DEV}" ;\
		REGISTRY=$$(echo "${IMAGE_DEV}" | sed 's|\([^/]*\)/.*|\1|') ;\
		MAIN_REPOSITORY=$$(echo "${IMAGE_DEV}" | sed 's|^[^/]*/||; s|:.*$$||') ;\
		JOB_IMAGE=$$(echo "${IMAGE_DEV}" | sed "s|/${ARTIFACT_ID}/|/${ARTIFACT_ID_JOB}/|") ;\
		JOB_REPOSITORY=$$(echo "$$JOB_IMAGE" | sed 's|^[^/]*/||; s|:.*$$||') ;\
		echo "Registry: $$REGISTRY" ;\
		echo "Main Repository: $$MAIN_REPOSITORY" ;\
		echo "Job Image: $$JOB_IMAGE" ;\
		echo "Job Repository: $$JOB_REPOSITORY" ;\
		$(BINARY_YQ) -i e ".main.image.registry=\"$$REGISTRY\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
		$(BINARY_YQ) -i e ".job.image.registry=\"$$REGISTRY\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
		$(BINARY_YQ) -i e ".main.image.repository=\"$$MAIN_REPOSITORY\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
		$(BINARY_YQ) -i e ".job.image.repository=\"$$JOB_REPOSITORY\"" ${K8S_COMPONENT_TARGET_VALUES} ;\
	fi

.PHONY: template-log-level
template-log-level: ${BINARY_YQ}
	@if [[ "${STAGE}" == "development" ]]; then \
		echo "Setting LOG_LEVEL env in deployment to ${LOG_LEVEL}!" ; \
		$(BINARY_YQ) -i e ".config.logging.level=\"${LOG_LEVEL}\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
	fi

.PHONY: template-image-pull-policy
template-image-pull-policy: $(BINARY_YQ)
	@if [[ "${STAGE}" == "development" ]]; then \
		echo "Setting pull policy to always!" ; \
		$(BINARY_YQ) -i e ".main.imagePullPolicy=\"Always\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
		$(BINARY_YQ) -i e ".job.imagePullPolicy=\"Always\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
	fi

.PHONY: template-api-config
template-api-config: $(BINARY_YQ)
	@if [[ "${STAGE}" == "development" ]]; then \
		echo "Setting api.host from environment-variable 'EXPORTER_HOST'" ; \
		$(BINARY_YQ) -i e ".config.api.host=\"${EXPORTER_HOST}\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
		echo "Setting api.skipTLSVerify from environment-variable 'EXPORTER_SKIP_VERIFY_TLS'" ; \
        $(BINARY_YQ) -i e ".config.api.skipTLSVerify=${EXPORTER_SKIP_VERIFY_TLS}" "${K8S_COMPONENT_TARGET_VALUES}" ; \
	fi

.PHONY: template-migration-config
template-migration-config: $(BINARY_YQ)
	@if [[ "${STAGE}" == "development" ]]; then \
		echo "Setting migration.regularSchedule from environment-variable 'MIGRATION_REGULAR_SCHEDULE'" ; \
		$(BINARY_YQ) -i e ".config.migration.regularSchedule=\"${MIGRATION_REGULAR_SCHEDULE}\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
		echo "Setting migration.finalSchedule from environment-variable 'MIGRATION_FINAL_TIMESTAMP'" ; \
        $(BINARY_YQ) -i e ".config.migration.finalSchedule=\"${MIGRATION_FINAL_TIMESTAMP}\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
	fi

.PHONY: template-smtp-config
template-smtp-config: $(BINARY_YQ)
	@if [[ "${STAGE}" == "development" ]]; then \
		echo "Setting smtp.server from environment-variable 'SMTP_SERVER'" ; \
		$(BINARY_YQ) -i e ".config.smtp.server=\"${SMTP_SERVER}\"" "${K8S_COMPONENT_TARGET_VALUES}" ; \
		echo "Setting smtp.port from environment-variable 'SMTP_PORT'" ; \
        $(BINARY_YQ) -i e ".config.smtp.port=${SMTP_PORT}" "${K8S_COMPONENT_TARGET_VALUES}" ; \
        echo "Setting smtp.to from environment-variable 'SMTP_TO'" ; \
        $(BINARY_YQ) -i e ".config.smtp.to=[\"${SMTP_TO}\"]" "${K8S_COMPONENT_TARGET_VALUES}" ; \
	fi

.PHONY: apikey-secret
apikey-secret: $(BINARY_YQ) ## generates a K8s secret for the API key from an environment variable
	@kubectl delete secret ces-importer-secret || true
	@kubectl create secret generic ces-importer-secret --from-literal=apiKey=${EXPORTER_API_KEY} --from-file=privateKey=${IMPORTER_SSH_KEY_FILE} --from-literal=mailPassword=${IMPORTER_MAIL_PASSWORD} --namespace="${NAMESPACE}" --context="${KUBE_CONTEXT_NAME}"

.PHONY: helm-apply-dev
helm-apply-dev:
	@sed -i -E "s/(^VERSION=[[:digit:]].[[:digit:]].[[:digit:]])/\1-$$(date +%s)/g" Makefile
	@make helm-apply
	@sed -i -E "s/(^VERSION=[[:digit:]].[[:digit:]].[[:digit:]])-.*/\1/g" Makefile
	@sed -i -E "s/(tag: [[:digit:]].[[:digit:]].[[:digit:]])-.*/\1/g" k8s/helm/values.yaml

.PHONY: docker-build
docker-build: check-docker-credentials check-k8s-image-env-var ${BINARY_YQ} ## Overwrite docker-build from k8s.mk to include build arguments
	@echo "Building docker image $(IMAGE)..."
	@echo "Build Arguments: $(BUILD_ARGS)"
	@DOCKER_BUILDKIT=1 docker build  . -t $(IMAGE) $(BUILD_ARGS)

.PHONY: images-import
images-import: ## import images from ces-importer and
	@echo "Import ces-importer image"
	@make image-import
	@echo "Import migration-job image"
	@make image-import \
		IMAGE=${ARTIFACT_ID_JOB}:${VERSION} \
		IMAGE_DEV_VERSION=$(CES_REGISTRY_HOST)$(CES_REGISTRY_NAMESPACE)/$(ARTIFACT_ID_JOB)/$(GIT_BRANCH):${VERSION} \
		BUILD_ARGS="--build-arg BINARY=import-job --build-arg UID=0 --build-arg GID=0"

.PHONY: ces-importer-release
ces-importer-release: ${BINARY_YQ} ## Interactively starts the release workflow for the ces-importer
	@echo "Starting git flow release..."
	@build/make/release.sh ces-importer