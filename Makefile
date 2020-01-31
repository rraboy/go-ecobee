all: build-docker

build-docker:
	docker build -t go-ecobee .
	
.PHONY:
	all build-docker

