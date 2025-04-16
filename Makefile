ARTIFACT_ID=ces-importer
MAKEFILES_VERSION=9.9.1
VERSION=0.0.1

GOTAG=1.24.2
.DEFAULT_GOAL:=help

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
