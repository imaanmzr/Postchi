# Frontend shortcuts (run from frontend/ or repo root via: make -C frontend <target>)
ROOT_DIR := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)

.PHONY: dev install build
dev:
	$(MAKE) -C $(ROOT_DIR) frontend

install:
	cd $(ROOT_DIR)/frontend && npm install

build:
	cd $(ROOT_DIR)/frontend && npm run build
