run: build
	@./bin/miniredis --listenAddr :6379

build:
	@go build -o bin/miniredis .
