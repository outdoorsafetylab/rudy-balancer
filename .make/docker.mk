CODENAME := $(notdir $(shell pwd))
IMAGE_NAME := outdoorsafetylab/$(CODENAME)
REPO_NAME ?= outdoorsafetylab/$(CODENAME)
VERSION ?= $(subst v,,$(shell git describe --tags --exact-match 2>/dev/null || echo ""))
GEOIP2_LICENSE_KEY ?=

# Build docker image.
#
# Usage:
#	make docker/build [no-cache=(no|yes)]

docker/build:
	docker build --network=host --force-rm \
		$(if $(call eq,$(no-cache),yes),--no-cache --pull,) \
		--build-arg GIT_HASH=$(GIT_HASH) \
		--build-arg GIT_TAG=$(GIT_TAG) \
		-t $(IMAGE_NAME) \
		-f .docker/Dockerfile \
		.

# Run docker image.
#
# Usage:
#	make docker/run

docker/run:
	docker run -it --rm \
		--name=$(CODENAME) \
		-p 8080:8080 \
		-e GEOIP2_LICENSE_KEY=$(GEOIP2_LICENSE_KEY) \
		$(IMAGE_NAME)

# Tag docker images.
#
# Usage:
#	make docker/tag [VERSION=<image-version>]

docker/tag:
	docker tag $(IMAGE_NAME) $(REPO_NAME):latest
ifneq ($(VERSION),)
	docker tag $(IMAGE_NAME) $(REPO_NAME):$(VERSION)
endif

# Push docker images.
#
# Usage:
#	make docker/push

docker/push:
	docker push $(REPO_NAME):latest
ifneq ($(VERSION),)
	docker push $(REPO_NAME):$(VERSION)
endif

docker: docker/build docker/tag docker/push

.PHONY: docker/build docker/run docker/tag docker/push docker
