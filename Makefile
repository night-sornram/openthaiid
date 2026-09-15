.PHONY: build test race vet fmt doctor sdk sdk-test tidy package clean check

build:
	mkdir -p dist
	go build -o dist/openthaiid ./cmd/openthaiid

test:
	go test ./...

race:
	go test -race ./internal/...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l .)"

doctor: build
	./dist/openthaiid doctor

sdk:
	cd sdk/web && npm install && npm run build

sdk-test:
	cd sdk/web && npm install && npm test

tidy:
	go mod tidy

package: build
	chmod +x scripts/package.sh
	./scripts/package.sh

check: fmt-check vet test sdk-test build doctor
	@echo "All local checks passed."

clean:
	rm -rf dist sdk/web/dist
