.DEFAULT_GOAL := help

SHELL := /usr/bin/env bash

# NB: many of these up-front vars need to use := to ensure that we expand them once (immediately)
# rather than re-running these (marginally expensive) commands each time the var is referenced

_GO_META = .go-version go.mod go.sum
_GO_SRCS := $(shell find . -type f -name "*.go" ) $(_GO_META)
# ignore 3rd party files (CocoaPods, Node modules, etc) and vendoreds (such as: tools/vendored-homebrew-install.sh)
_SH_SRCS := $(shell find . -type f -name "*.sh" ! -iname "vendored-*.sh" | grep -v Pods | grep -v node_modules | grep -v \.venv | grep -v .terraform/modules)
_LOCAL_PLATFORM := $(shell uname | tr '[:upper:]' '[:lower:]')

# all files recursively under <uiproject>/{src, public} (recursive) and directly under <uiproject>/ are edited by us, though 'sharedui' doesn't have public (yet).
_SHAREDUI_REACT_SRCS := $(shell find sharedui/src) $(shell find sharedui -maxdepth 1 -not -type d)
_UILIB_REACT_SRCS := $(shell find ui-component-lib/src) $(shell find ui-component-lib/public -maxdepth 1 -not -type d)
_CONSOLEUI_REACT_SRCS := $(shell find console/consoleui/src) $(shell find console/consoleui/public) $(shell find console/consoleui -maxdepth 1 -not -type d)
_PLEXUI_REACT_SRCS := $(shell find plex/plexui/src) $(shell find plex/plexui/public) $(shell find plex/plexui -maxdepth 1 -not -type d)

SERVICE_BINARIES = bin/console bin/plex bin/idp bin/authz bin/checkattribute bin/logserver bin/dataprocessor bin/worker
CODEGEN_BINARIES = bin/parallelgen bin/genconstant bin/gendbjson bin/genvalidate bin/genstringconstenum bin/genorm bin/genschemas bin/genevents bin/genrouting bin/genhandler bin/genopenapi bin/genpageable

TF_PATH = $(if $(TG_TF_PATH),$(TG_TF_PATH),"terraform")

.PHONY: help
help: ## List user-facing Make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: grafana-start
grafana-start: ## Start running Prometheus and Grafana locally
	docker compose -f docker/grafana-dev/docker-compose.yaml up --detach

.PHONY: grafana-stop
grafana-stop: ## Stop running Prometheus and Grafana
	docker compose -f docker/grafana-dev/docker-compose.yaml down

# This is the main target for developers. It starts all the services and the UIs.
# services-dev exists so that we don't rebuild all the react stuff in a ui-dev invocation
.PHONY:
dev: console/consoleui/build plex/plexui/build ## Run all userclouds services locally
	make services-dev

services-dev: $(SERVICE_BINARIES) bin/devbox bin/devlb
	@UC_REGION=themoon AWS_ACCESS_KEY_ID="${AWS_DEV_CREDS_AWS_KEY_ID}" AWS_SECRET_ACCESS_KEY="${AWS_DEV_CREDS_AWS_KEY_SECRET}"  bin/devbox

.PHONY: dbshell-dev
dbshell-dev: bin/tenantdbshell ## Start and connect to local db
	@tools/db-shell.sh dev

.PHONY: dbshell-prod
dbshell-prod: check-deps bin/tenantdbshell ## Connect to the production databases
	@UC_UNIVERSE=prod tools/db-shell.sh prod

.PHONY: dbshell-staging
dbshell-staging: check-deps bin/tenantdbshell ## Connect to the staging databases
	@UC_UNIVERSE=staging tools/db-shell.sh staging

dbshell-debug: check-deps bin/tenantdbshell
	@UC_UNIVERSE=debug tools/db-shell.sh debug

# NOTE: call `make migrate-dev` explicitly inside single DEVDB START/STOP scope because there is a race
# condition shutting down & restarting CDB if we invoke migrate-dev and provision-dev serially via
# make target dependencies.
.PHONY: provision-dev
provision-dev: bin/migrate bin/provision ## Provision dev to latest
	make migrate-dev
	@tools/provision.sh dev

.PHONY: provision-prod
provision-prod: bin/provision check-deps ## Provision prod env to latest
	@tools/provision-events.sh prod

.PHONY: provision-staging
provision-staging: bin/provision check-deps ## Provision staging env to latest
	@tools/provision-events.sh staging

provision-debug: bin/provision check-deps
	@tools/provision.sh debug

.PHONY: migrate-dev
migrate-dev: FLAGS ?=--noPrompt
migrate-dev: bin/migrate ## Migrate your dev databases up to latest
	@tools/db-migrate.sh dev $(FLAGS)

.PHONY: migrate-prod
migrate-prod: bin/migrate check-deps ## Migrate production databases up to latest
	@tools/db-migrate.sh prod

.PHONY: migrate-staging
migrate-staging: bin/migrate check-deps ## Migrate staging databases up to latest
	@tools/db-migrate.sh staging

migrate-debug: bin/migrate check-deps
	@tools/db-migrate.sh debug

.PHONY: deploy-prod
deploy-prod: bin/migrate check-deps ## Deploy your current HEAD to production
	@tools/deploy.sh prod

.PHONY: deploy-staging
deploy-staging: bin/migrate check-deps ## Deploy your current HEAD to staging
	@tools/deploy.sh staging

deploy-debug: bin/migrate check-deps
	@tools/deploy.sh debug


# NB: we no longer build all the codegen binaries themselves here because they are
# now run in go routines in parallelgen to speed up package loading
.PHONY: codegen
codegen: bin/parallelgen ## Run codegen to update generated files
	go install github.com/userclouds/easyjson/...@v0.9.0-uc6
	parallelgen

# if you need to run codegen serially, you can use this target (but it's much slower)
# you can also run an individual codegen operation by running the command after
# //go:generate in a specific file, you just need to ensure you've build that binary
# this is useful if you're debugging codegen, or if you are adding a ton (hundreds?)
# of new logging event codes and hitting conflicts because genevents runs in parallel
# across services.
codegen-serial: $(CODEGEN_BINARIES) ## Run codegen to update generated files
	go generate ./...
	make gen-openapi-spec

TOOL_DEPS=jq yq direnv bash curl git-lfs hub awscli n yarn postgresql@14 tmux \
	restack python3 terraform tflint terragrunt redis gh \
	helm kubernetes-cli kubeconform argocd docker \
	coreutils # for timeout (used in redis-shell.sh)
check-deps:
	@tools/check-deps.sh $(TOOL_DEPS)

.PHONY: clean ## Clean up rebuildable binaries
clean:
    # our service binaries
	-rm -f $(SERVICE_BINARIES)

    # tools
	-rm -f bin/goimports
	-rm -f bin/staticcheck
	-rm -f bin/errcheck
	-rm -f bin/shellcheck
	-rm -f bin/shfmt
	-rm -f bin/uclint
	-rm -f cover.out
	-rm -rf heap_profiler

    # generated
	-rm -f $(CODEGEN_BINARIES)

######################## test runner ######################

# Test DB should create a new DB / store per test run, so we can parallelize
#   The store itself is in memory for perf, but the dir is still useful for interacting with it
# Store the connection URL in a file so we can link it in
# Note that all the DB setup happens in the test target so we don't
# create random empty dirs during `make dev` etc :)
# We also source tools/devenv.sh in order to get AWS creds for secret manager, which is
#  a bit awkward given the way Make creates a shell environment per line and doesn't let them export
# TODO: we actually shouldn't source devenv.sh in CI, but since CI runs sh (not bash) it doesn't actually hurt
# TODO: should UI tests get pulled out to a separate target at some point? We could have `make servertest`, `make uitest`,
#  and `make test` can just depend on both (so then TESTARGS would only apply to `make servertest`).
# Use TESTARGS to run to eg. a specific test / package tests, `TESTARGS=./idp/internal/authn make test`
#  or `TESTARGS="./plex/internal -run TestCaseFoo"` make test for a single test
.PHONY: test
test: TESTARGS ?= ./...
test: TESTENV ?= test   # CI uses this to override UC_UNIVERSE
test: _TEMPDIR := $(shell mktemp -d)
test: _TEMPFILE := $(_TEMPDIR)/testdb
test: _TESTDB_STOP = docker rm -f testdb
test: ## Build project and run test suite
	@tools/setup-test-db.sh $(_TEMPFILE)
	@if [ "$(strip $(TESTENV))" == "test" ]; then\
		tools/start-redis.sh;\
	else\
		echo "skipping redis because TESTENV was specified ($(TESTENV))";\
	fi
	UC_UNIVERSE=$(TESTENV) UC_REGION=mars UC_TESTDB_POINTER_PATH=$(_TEMPFILE) go test \
	         -race \
			 -coverprofile=cover.out \
			 -vet=off \
			 $(TESTARGS) || ( $(_TESTDB_STOP) && exit 1)
	@$(_TESTDB_STOP)
	@if [ "$(strip $(TESTARGS))" == './...' ]; then\
		make consoleui-test;\
	else\
		echo "skipping UI tests because TESTARGS was specified ($(TESTARGS))";\
	fi

# Very similar target to the "test" target above that we will use in CI
# few chnages from the regular test target:
# * Runs all tests (no TESTARGS support)
# * No cleanup (stopping test DB after tests)
# * Will not try to start redis (CI runs them as services already)
# * Will only run backend (golang) test and not the UI tests
.PHONY: test-backend-ci
test-backend-ci: TESTARGS ?= ./...
test-backend-ci: _TEMPFILE := $(shell mktemp -d)/testdb
test-backend-ci: ## Build project and run test suite
	@tools/setup-test-db.sh $(_TEMPFILE)
	UC_UNIVERSE=ci UC_REGION=mars UC_TESTDB_POINTER_PATH=$(_TEMPFILE) go test \
	         -timeout 20m -parallel 4 -race -coverprofile=cover.out -vet=off $(TESTARGS)

.PHONY: test-provisioning
test-provisioning: bin/provision ## Test tenant & db provisioning
	tools/provision-test.sh

test-helm:
	./helm/test-charts.sh

test-fixme:
	tools/check-fixme.sh

test-codegen:
	UC_CONFIG_DIR=./config tools/check-codegen.sh

check-go-modules:
	tools/check-go-modules.sh

######################### linters ##########################

.PHONY: lint
lint: ## Lint code and config
	@tools/lint.sh

lint-golang: .go-version bin/goimports bin/staticcheck bin/errcheck bin/revive bin/uclint bin/modernize
	@tools/lint-golang.sh

lint-frontend: sharedui/build ui-lib/build # TODO: this is a required dep because the build generates *.d.ts files needed to lint downstream modules. We may want to check these in to git in the future?
	@tools/lint-frontend.sh

lint-shell: bin/shfmt bin/shellcheck
	@tools/lint-shell.sh "$(_SH_SRCS)"

lint-python:
	@tools/lint-python.sh

lint-sql:
	@tools/lint-sql.sh

.PHONY: lintfix
lintfix: bin/goimports bin/shfmt ## Automatically fix some lint problems
lintfix: sharedui/build ui-lib/build # TODO: this is a required dep because the build generates *.d.ts files needed to lint downstream modules. We may want to check these in to git in the future?
	@tools/lintfix.sh "$(_SH_SRCS)"

.PHONY: lint-terraform
lint-terraform: ## Run terraform linters
	@make tgfmt
	@make tffmt
	@make tflint

tgfmt:
	@echo "Checking terragrunt HCL formatting..."
	@terragrunt hcl fmt --check --exclude-dir .terraform && echo "Formatting okay" \
		|| (echo "Run \"terragrunt hcl fmt --exclude-dir .terraform\" to fix the above files" >&2; exit 1)

tg-generate:
	@echo "Running terragrunt to generate terraform files"
# Terragrunt's "get" command is used here as a reliable method to trigger file generation.
# While "version" command was previously used, "get" is now preferred as it consistently
# generates files across all Terragrunt versions.
# Note: The command may fail in CI or local dev due to AWS credentials, but this is
# acceptable since file generation occurs before the command fails.
	@find terraform/configurations -name .terraform.lock.hcl -execdir terragrunt run -- get > /dev/null \;

tffmt: tg-generate
	@echo "Checking terraform formatting..."
	@$(TF_PATH) fmt -recursive -check && echo "Formatting okay" \
		|| (echo "Run \"terragrunt fmt -recursive\" to fix the above files" >&2; exit 1)

tflint:
	@echo Running tflint...
	@cd terraform; tflint --recursive

######################### logging ##########################
.PHONY: log-prod
log-prod: bin/uclog
	UC_UNIVERSE=prod tools/ensure-aws-auth.sh
	bin/uclog --time 5 --streamname prod --live --verbose --outputpref sh --interactive --summary --ignorehttpcode 401,409 listlog

.PHONY: log-staging
log-staging: bin/uclog
	UC_UNIVERSE=staging tools/ensure-aws-auth.sh
	bin/uclog --time 5 --streamname staging --live  --verbose --outputpref sh --interactive --summary listlog

.PHONY: log-debug
log-debug: bin/uclog
	UC_UNIVERSE=debug tools/ensure-aws-auth.sh
	bin/uclog --time 5 --streamname debug --live --verbose --outputpref sh --interactive --summary listlog

######################### service binaries ##########################
$(SERVICE_BINARIES): $(_GO_SRCS)
	go build -o $@ \
		-ldflags \
			"-X userclouds.com/infra/service.buildHash=$(shell git rev-parse HEAD) \
			 -X userclouds.com/infra/service.buildTime=$(shell TZ=UTC git show -s --format=%cd --date=iso-strict-local HEAD)" \
		./$(notdir $@)/cmd

bin/ucconfig: $(_GO_SRCS)
	go build -o bin/ucconfig ./cmd/ucconfig

bin/opensearch: $(_GO_SRCS)
	go build -o bin/opensearch ./cmd/opensearch

bin/devbox: $(_GO_SRCS)
	go build -o bin/devbox ./cmd/devbox

bin/devlb: $(_GO_SRCS)
	go build -o bin/devlb ./cmd/devlb

bin/azcli: $(_GO_SRCS)
	go build -o bin/azcli ./cmd/azcli

bin/cachelookup: $(_GO_SRCS)
	go build -o bin/cachelookup ./cmd/cachelookup

bin/cachetool: $(_GO_SRCS)
	go build -o bin/cachetool ./cmd/cachetool

bin/provisiontenanturls: $(_GO_SRCS)
	go build -o bin/provisiontenanturls ./cmd/provisiontenanturls

bin/cleanplextokens: $(_GO_SRCS)
	go build -o bin/cleanplextokens ./cmd/cleanplextokens

bin/cleanuserstoredata: $(_GO_SRCS)
	go build -o bin/cleanuserstoredata ./cmd/cleanuserstoredata

bin/setcompanytype: $(_GO_SRCS)
	go build -o bin/setcompanytype ./cmd/setcompanytype


######################### code gen binaries #########################

$(CODEGEN_BINARIES): $(_GO_SRCS)
	@echo "building $@"
	go build -o $@ ./cmd/$(notdir $@)

######################### react ui stuff #########################

# Install/update dependencies (node modules) in development environments. This creates/updates the `node_modules`
# directories wherever there are `package.json` files in our tree, and creates/updates the `yarn.lock` file
# which tracks metadata for all installed modules. `package.json` is the source of truth for which dependencies/modules
# to fetch and what versions to use, but re-running this may alter the `yarn.lock` file as dependencies change as
# we don't always pin specific versions.
.PHONY: ui-yarn-install
ui-yarn-install: check_venv
ui-yarn-install: ## Install/update dependencies for our React UI projects (needed if adding new deps)
	python3 -mpip install --prefix $(VENV_PATH) setuptools # needed for gyp
	yarn install
	@tools/install-playwright.sh

# Install dependencies and reqs needed to build UI bundles and run UI tests (playwright) in CI.
.PHONY: ui-yarn-ci
ui-yarn-ci:
	time yarn install --immutable
	time tools/install-playwright.sh

# Install/update dependencies (node modules) in CI / Build pipelines for UI apps & libraries.
# It is very similar to `ui-yarn-install` except it treats the `package.json` and `yarn.lock` files as read-only
# and ensures that all modules are reinstalled from scratch. Hence its suitability for CI/Build jobs but why it
# isn't used in normal development.
# https://stackoverflow.com/questions/52499617/what-is-the-difference-between-npm-install-and-npm-ci
# and https://stackoverflow.com/questions/58482655/what-is-the-closest-to-npm-ci-in-yarn
ui-yarn-build-only-ci:
	time yarn install --immutable

sharedui/build: $(_SHAREDUI_REACT_SRCS)
	@rm -rf sharedui/build
	yarn sharedui:build

ui-lib/build: $(_UILIB_REACT_SRCS)
	yarn ui-lib:build # this does the rm -rf part itself

console/consoleui/build: $(_CONSOLEUI_REACT_SRCS) sharedui/build ui-lib/build
	@rm -rf console/consoleui/build
	yarn consoleui:build
	UC_CONFIG_DIR=config/ go run cmd/consoleuiinitdata/main.go

plex/plexui/build: $(_PLEXUI_REACT_SRCS) sharedui/build ui-lib/build
	@rm -rf plex/plexui/build
	yarn plexui:build

.PHONY: ui-build
ui-build: sharedui/build ui-lib/build console/consoleui/build plex/plexui/build ## Build static asset bundles for all React UI projects

.PHONY: ui-clean
ui-clean: ## Clean the output (build) dirs of our React UI projects
	@rm -rf sharedui/build ui-component-lib/build console/consoleui/build plex/plexui/build

.PHONY: ui-yarn-clean
ui-yarn-clean: ui-clean ## Clean yarn generated/downloaded files. Must re-run `make ui-yarn-install` after
	@rm -rf node_modules sharedui/node_modules ui-component-lib/node_modules console/consoleui/node_modules plex/plexui/node_modules
	@rm -rf yarn.lock sharedui/yarn.lock ui-component-lib/yarn.lock console/consoleui/yarn.lock plex/plexui/yarn.lock

.PHONY: sharedui-dev
sharedui-dev: ## Run the React rollup server in watch mode for 'sharedui'
	yarn sharedui:dev

.PHONY: ui-lib-watch
ui-lib-watch: ## Run the React rollup server in watch mode for 'ui-component-lib'
	yarn ui-lib:watch

.PHONY: ui-lib-dev
ui-lib-dev: ## Run the React rollup server in watch mode for 'ui-component-lib'
	yarn ui-lib:dev

.PHONY: consoleui-dev
consoleui-dev: sharedui/build ## Run the React development server for 'consoleui'
	yarn consoleui:dev

.PHONY: plexui-dev
plexui-dev: sharedui/build ## Run the React development server for 'plexui'
	BROWSER=none yarn plexui:dev

.PHONY: consoleui-test
consoleui-test: ## Run the tests for 'consoleui'
	make sharedui/build ui-lib/build
	CI=1 yarn consoleui:test

.PHONY: ui-dev
ui-dev: ## Run the dev backend + react dev server for plex & console
	tmux new-session "tmux source-file tools/tmux-uidev.cmd"

######################### tool build rules ##########################

bin/envtest: $(_GO_SRCS)
	go build -o $@ ./cmd/envtest

bin/containerrunner: $(_GO_SRCS)
	go build -o $@ ./cmd/containerrunner

tools: bin/goimports
bin/goimports: .go-version go.mod
	go install -mod=readonly golang.org/x/tools/cmd/goimports

tools: bin/modernize
bin/modernize: .go-version go.mod
	go install -mod=readonly golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@v0.18.0

tools: bin/shfmt
# When upgrading shfmt, make sure to update the version the cache key in .github/workflows/lint-shell.yml
bin/shfmt: .go-version go.mod
	go install -mod=readonly mvdan.cc/sh/v3/cmd/shfmt@v3.11.0

# Note that in CI, we untar shellcheck directly into bin/ so we don't polute our git porcelain status :)
tools: bin/shellcheck
bin/shellcheck:
ifeq ($(_LOCAL_PLATFORM), darwin)
	shellcheck --version || brew install shellcheck
	ln -s $$(which shellcheck) bin/shellcheck
else ifeq ($(_LOCAL_PLATFORM), linux)
	tools/install-shellcheck-linux.sh
else
	$(error "Don't know how to download shellcheck on $(_LOCAL_PLATFORM)")
endif

tools: bin/staticcheck
bin/staticcheck: .go-version go.mod
	go install -mod=readonly honnef.co/go/tools/cmd/staticcheck@2025.1.1

tools: bin/revive
bin/revive: .go-version go.mod
	go install -mod=readonly github.com/mgechev/revive

tools: bin/errcheck
bin/errcheck: .go-version go.mod
	go install -mod=readonly github.com/kisielk/errcheck

tools: bin/uclint
bin/uclint: $(_GO_SRCS)
	go build -o $@ userclouds.com/cmd/uclint

tools:bin/migrate
bin/migrate: $(_GO_SRCS)
	go build -o bin/migrate ./cmd/migrate

tools: bin/provision
bin/provision: $(_GO_SRCS)
	go build -o bin/provision ./cmd/provision

tools: bin/tenantdbshell
bin/tenantdbshell: $(_GO_SRCS)
	go build -o bin/tenantdbshell ./cmd/tenantdbshell

tools: bin/queryrunner
bin/queryrunner: $(_GO_SRCS)
	go build -o bin/queryrunner ./cmd/queryrunner

tools: bin/dataimport
bin/dataimport: $(_GO_SRCS)
	go build -o bin/dataimport ./cmd/dataimport

tools: bin/cleanupusercolumns
bin/cleanupusercolumns: $(_GO_SRCS)
	go build -o bin/cleanupusercolumns ./cmd/cleanupusercolumns

tools: bin/uclog
bin/uclog: $(_GO_SRCS)
	go build -o bin/uclog ./cmd/uclog

tools: bin/testdevcert
bin/testdevcert: $(_GO_SRCS)
	go build -o bin/testdevcert ./cmd/testdevcert

tools: bin/consoleuiinitdata
bin/consoleuiinitdata: $(_GO_SRCS)
	go build -o bin/consoleuiinitdata ./cmd/consoleuiinitdata

tools: bin/runaccessors
bin/runaccessors: $(_GO_SRCS)
	go build -o bin/runaccessors ./cmd/runaccessors

tools: bin/envtestecs
bin/envtestecs: $(_GO_SRCS)
	go build -o bin/envtestecs ./cmd/envtestecs

bin/auditlogview: $(_GO_SRCS)
	go build -o bin/auditlogview ./cmd/auditlogview

bin/remoteuserregionconfig: $(_GO_SRCS)
	go build -o bin/remoteuserregionconfig ./cmd/remoteuserregionconfig

install-tools:
	brew install $(TOOL_DEPS)
############################ initial dev setup ###########################
# this stuff should be idempotent...in theory

# NB: we actually run brew install here (rather than just a check-deps dependency) so that
# we're keeping our dev environments up to date-ish :)
# We also set up userclouds_dev_root to keep dev closer to matching our config in AWS,
# rather than relying on the local root account to have all permissions
# NBB: we create and then drop a "fake" table in defaultdb to ensure GRANT ... on defaultdb.*
# works even if you don't have any tables there (new install). We keep this GRANT around
# in case you have existing tables (specifically migrations) that our userclouds subaccount
# needs access to (this shouldn't happen after we transitioned to rootdb but maybe?)
# We use dockerized redis here for Devin since for some reason local redis doesn't work there
.PHONY: devsetup
devsetup: bin/testdevcert check_venv install-tools
devsetup: ## Initial setup when you clone the repo
	brew services restart redis || (docker run -dp 6379:6379 redis && echo "Using dockerized redis")
	mkdir -p ~/.n
	n $(shell cat .node-version | cut -c2-)
	tools/git/install.sh
	echo "CREATE USER userclouds_dev_root WITH CREATEDB CREATEROLE;" | psql postgres
	echo "CREATE DATABASE defaultdb;" | psql postgres
	echo "GRANT ALL PRIVILEGES ON DATABASE defaultdb TO userclouds_dev_root;" | psql postgres
	echo "CREATE TABLE tmp (id UUID);" | psql postgres
	echo "DROP TABLE tmp;" | psql postgres
	make provision-dev
	make ui-yarn-install ensure-secrets-dev
	python3 -mpip install --prefix $(VENV_PATH) -r requirements.txt
	cd terraform; tflint --init
	bin/testdevcert || (echo "ERROR: please see README for instructions on installing the dev HTTPS certificate"; exit 1)
	@echo "!!!!!!!!!!"
	@echo "IMPORTANT: to make yourself a UserClouds company admin, run \`tools/make-company-admin.sh dev '<youruserid>'\`"
	@echo "after creating an account on console (your user ID can be found in the upper right profile menu)"
	@echo "!!!!!!!!!!"

VENV_BASE_DIR ?=
VENV_PATH = $(VENV_BASE_DIR).venv
check_venv:
	@tools/ensure-python-venv.sh

.PHONY: devsetup-samples
devsetup-samples: bin/provision ## Setup your environment to run our sample projects
	brew services start postgresql@14 || true # ignore error if postgres is already running
	echo "create database sample_events; create user userclouds_events with password 'samples'; GRANT ALL PRIVILEGES ON DATABASE sample_events TO userclouds_events;" | psql postgres
	bin/provision provision company config/provisioning/samples/company_contoso.json
	bin/provision provision tenant config/provisioning/samples/tenant_contoso_dev.json
	bin/provision provision tenant config/provisioning/samples/tenant_contoso_prod.json
	bin/provision provision company config/provisioning/samples/company_allbirds.json
	bin/provision provision tenant config/provisioning/samples/tenant_allbirds_dev.json
	bin/provision provision tenant config/provisioning/samples/tenant_allbirds_prod.json

############################ samples ###########################

# run devsetup-samples first; we don't depend on it explicitly because it's sloooow
.PHONY:
run-events: ## Run the events sample app
	@cd samples/events; go run main.go

.PHONY:
build-deploy-binaries: $(SERVICE_BINARIES) ui-build bin/consoleuiinitdata bin/optionsfilegenerator
	echo "Built binaries for deployment"

.PHONY:
gen-openapi-spec: ## Generate the consolidated OpenAPI spec for our APIs
	go run cmd/genopenapi/main.go

configure-aws-cli:  ## configures AWS CLI for use w/ SSO
	./tools/configure-aws-cli.sh

.PHONY:
ensure-secrets-dev: ## Make sure the secrets needed to run local dev environment are present
	UC_UNIVERSE=debug tools/ensure-aws-auth.sh
	go run cmd/ensuresecrets/main.go
	direnv reload # need to reload direnv to pick up new secrets
	aws sso logout # Don't stay logged into AWS SSO


.PHONY: tf-provider-build
tf-provider-build: ## Build the Terraform provider
	$(MAKE) -C ./public-repos/terraform-provider-userclouds build


upgrade-terraform-providers:
	find terraform -type f -name .terraform.lock.hcl -exec ./terraform/upgrade-terraform-providers.sh {} \;
