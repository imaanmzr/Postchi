# Backend shortcuts (run from backend/ or repo root via: make -C backend <target>)
ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)

.PHONY: run test build migrate-up migrate-down migrate-status
run:
	$(MAKE) -C $(ROOT_DIR) backend

test:
	cd $(ROOT_DIR)/backend && go test ./...

build:
	$(MAKE) -C $(ROOT_DIR) build-backend

migrate-up migrate-down migrate-status migrate migrate-create migrate-force:
	$(MAKE) -C $(ROOT_DIR) $@
