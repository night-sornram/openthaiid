.PHONY: build test race vet fmt doctor sdk sdk-test tidy package clean check acceptance

VERSION ?= 0.0.1-dev

build:
	mkdir -p dist
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/openthaiid ./cmd/openthaiid

test:
	go test ./... -count=1

race:
	go test -race ./internal/... -count=1

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

sdk-test:
	cd sdk/web && npm ci && npm test

tidy:
	go mod tidy

package: build
	chmod +x scripts/package.sh
	VERSION=$(VERSION) ./scripts/package.sh

check: fmt-check vet test sdk-test build
	@echo "All local checks passed."

acceptance: build sdk
	cd acceptance && npm ci && npm test

clean:
	rm -rf dist sdk/web/dist acceptance/node_modules
