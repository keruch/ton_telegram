ENV := $(if $(ENV),$(ENV),local)
BINARIES_DIR := cmd
SERVICES_LIST := $(shell find $(BINARIES_DIR) -maxdepth 1 \( ! -iname "$(BINARIES_DIR)" \) -type d -exec basename {} \;)
SERVICES_RUN_TARGETS_LIST := $(addprefix run-, $(SERVICES_LIST))

mod:
	go mod tidy

vendor:
	go mod vendor

update: mod vendor

install-migrator:
	go get -v github.com/rubenv/sql-migrate/...

migrate-status:
	ENV=$(ENV) ./migrations/runner.sh status

migrate-up:
	ENV=$(ENV) ./migrations/runner.sh up

migrate-down:
	ENV=$(ENV) ./migrations/runner.sh down

migrate-reports-down:
	ENV=$(ENV) ./migrations/dealer-reports/runner.sh down

migrate-up-down-up:
	ENV=$(ENV) ./migrations/runner.sh up-down-up

env-start:
	./docker/env.sh start

env-restart:
	./docker/env.sh restart

env-stop:
	./docker/env.sh stop

setup-env: install-migrator update migrate-up env-start

$(SERVICES_RUN_TARGETS_LIST): run-%:
	./docker/env.sh start
	go run ./cmd/$*
