.DEFAULT_GOAL := all

.PHONY: all
all: build push

.PHONY: build
build:
ifeq ($(TAG_VERSION),)
	@echo "You have to provide a TAG_VERSION to build this image.\n"
	@exit 1
endif
	docker build -t docker.io/akerouanton/swarm-tasks-exporter:$(TAG_VERSION) .

.PHONY: push
push:
ifeq ($(TAG_VERSION),)
	@echo "You have to provide a TAG_VERSION to build this image.\n"
	@exit 1
endif
	docker push docker.io/akerouanton/swarm-tasks-exporter:$(TAG_VERSION)
