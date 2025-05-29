.PHONY: build
build:
	go build -o dist/my_go_cli_example ./cmd/my_go_cli_example

.PHONY: install
install:
	go install github.com/takaishi/my_go_cli_example/cmd/my_go_cli_example

.PHONY: test
test:
	go test -race ./...
