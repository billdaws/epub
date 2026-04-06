.PHONY: setup tag

setup:
	git config core.hooksPath .githooks

tag:
	@go run ./tools/semver
