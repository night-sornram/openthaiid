.PHONY: build test race vet fmt doctor sdk sdk-test tidy package clean check acceptance test-repeat

VERSION ?= 0.0.1-dev

build:
	mkdir -p dist
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/openthaiid ./cmd/openthaiid

test:
	go test ./... -count=1 -timeout 8m

race:
	go test -race ./internal/... -count=1 -timeout 10m

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l .)"

doctor: build
	./dist/openthaiid doctor

sdk:
	cd sdk/web && npm ci && npm run build
	cd sdk/react && npm ci && npm run build

sdk-test:
	cd sdk/web && npm ci && npm test && npm run build
	cd sdk/react && npm ci && npm test

tidy:
	go mod tidy

package: build
	chmod +x scripts/package.sh
	VERSION=$(VERSION) ./scripts/package.sh

check: fmt-check vet test sdk-test build
	@echo "All local checks passed."

# Repeat formerly flaky integration cases many times (local + full-CI gate).
# GitHub PR CI uses -count=5; push to main/dev and workflow_dispatch use 50.
test-repeat:
	go test ./internal/bridge/ ./internal/reader/ -count=50 -timeout 20m

acceptance: build sdk
	cd acceptance && npm ci && npm run test:all

clean:
	rm -rf dist sdk/web/dist sdk/react/dist acceptance/node_modules
