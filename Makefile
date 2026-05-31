APP=quickshare
VERSION=0.1.2

.PHONY: build build-windows build-all clean run test

build:
	go build -ldflags="$(LDFLAGS) -X main.version=$(VERSION)" -o $(APP)$(SUFFIX) .

build-windows:
	go build -ldflags="-H windowsgui -X main.version=$(VERSION)" -o $(APP).exe .

build-all:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(APP)-darwin-x86_64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(APP)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(APP)-linux-x86_64 .
	GOOS=linux GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION)" -o dist/$(APP)-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -X main.version=$(VERSION)" -o dist/$(APP)-windows-x86_64.exe .
	@echo "Binaries built in dist/"

clean:
	rm -rf dist/ $(APP) $(APP).exe

run:
	go run .

test:
	@echo "No tests yet"
