run: build
	@./bin/miniredis --listenAddr :5001

build:
	@go build -o bin/miniredis .
