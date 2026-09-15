.PHONY: build test race vet fmt doctor sdk sdk-test tidy

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

doctor: build
	./dist/openthaiid doctor

sdk:
	cd sdk/web && npm install && npm run build

sdk-test:
	cd sdk/web && npm install && npm test

tidy:
	go mod tidy
