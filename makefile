run: build
	@./bin/miniredis

build:
	@go build -o bin/miniredis .
