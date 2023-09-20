.DEFAULT_GOAL := test

# Run tests and generates html coverage file
cover: test
	@go tool cover -html=./cover.text -o ./cover.html
	@test -f ./cover.out && rm ./cover.out;
.PHONY: cover

# Run linters
lint:
	@golangci-lint run ./...
.PHONY: lint

# Run test
test:
	@go test ./...
	@go test ./... -short -race
	@go test ./... -run=NONE -bench=. -benchmem
.PHONY: test

# Run test-coverage
test-coverage:
	@go test -cpu=4 -race -coverprofile=coverage.txt -covermode=atomic
.PHONY: test-coverage
