.PHONY: build test race vet fmt doctor sdk sdk-test tidy package package-macos clean check acceptance test-repeat smoke-helper qualify-smoke smoke-package friday-dry-run bind-stress-smoke

VERSION ?= 0.1.0-rc.7

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

# Darwin builds need a Mac host (ebfe/scard + Cocoa tray cgo).
# Does not sign or notarize.
package-macos:
	@host=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	if [ "$$host" != "darwin" ]; then \
		echo "make package-macos must run on a Mac (cgo + PCSC.framework + Cocoa)."; \
		echo "Linux/Windows cannot cross-compile darwin. See docs/installation.md."; \
		echo "This target does not produce a signed or notarized build."; \
		exit 1; \
	fi
	chmod +x scripts/package.sh
	VERSION=$(VERSION) OS=darwin ./scripts/package.sh

smoke-helper: build
	chmod +x scripts/smoke-clinic-helper.sh
	./scripts/smoke-clinic-helper.sh

qualify-smoke: build
	chmod +x scripts/smoke-qualify.sh scripts/hardware-qualify.sh
	./scripts/smoke-qualify.sh

# Synthetic PC/SC bind-stress (no ACR39U). Does not close P2–P10.
bind-stress-smoke: build
	chmod +x scripts/smoke-bind-stress.sh scripts/bind-stress.sh
	./scripts/smoke-bind-stress.sh

# Synthetic Friday-morning rehearsal (no ACR39U). Not a hardware pass.
friday-dry-run: build
	chmod +x scripts/friday-dry-run.sh scripts/smoke-qualify.sh scripts/hardware-qualify.sh
	./scripts/friday-dry-run.sh $(ARGS)

smoke-package:
	chmod +x scripts/package.sh scripts/smoke-package.sh
	./scripts/smoke-package.sh

check: fmt-check vet test sdk-test build
	@echo "All local checks passed."

# Repeat formerly flaky integration cases many times (local + full-CI gate).
# GitHub PR CI uses -count=5; push to main/dev and workflow_dispatch use 50.
test-repeat:
	go test ./internal/bridge/ ./internal/reader/ -count=50 -timeout 20m

acceptance: build sdk
	cd acceptance && npm ci && npm run test:all

# Linux / this cloud: Playwright Firefox (first-class). Safari/WebKit is macOS-only.
acceptance-firefox: build sdk
	cd acceptance && npm ci && npx playwright install --with-deps firefox && OPENTHAIID_BROWSER=firefox npm run test:all

acceptance-webkit: build sdk
	cd acceptance && npm ci && npx playwright install --with-deps webkit && OPENTHAIID_BROWSER=webkit npm run test:all

clean:
	rm -rf dist sdk/web/dist sdk/react/dist acceptance/node_modules
