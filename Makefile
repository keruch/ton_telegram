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

migrate-bot-status:
	ENV=$(ENV) ./migrations/bot/runner.sh status

migrate-bot-up:
	ENV=$(ENV) ./migrations/bot/runner.sh up

migrate-bot-down:
	ENV=$(ENV) ./migrations/bot/runner.sh down

migrate-bot-up-down-up:
	ENV=$(ENV) ./migrations/bot/runner.sh up-down-up

migrate-market-status:
	ENV=$(ENV) ./migrations/market/runner.sh status

migrate-market-up:
	ENV=$(ENV) ./migrations/market/runner.sh up

migrate-market-down:
	ENV=$(ENV) ./migrations/market/runner.sh down

migrate-market-up-down-up:
	ENV=$(ENV) ./migrations/market/runner.sh up-down-up

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
