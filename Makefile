.PHONY: build

build:
	export CGO_ENABLED=0
	sam build