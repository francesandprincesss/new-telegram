MODULE := github.com/francesandprincesss/new-telegram
PROTO_FILES := $(wildcard proto/chat/v1/*.proto)
PROTOC_GEN_GO := .tools/protoc-gen-go
PROTOC_GEN_GO_GRPC := .tools/protoc-gen-go-grpc

.PHONY: proto build test test-race vet migrate-up migrate-down docker-up docker-down

proto: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_GRPC)
	protoc --proto_path=proto \
		--plugin=protoc-gen-go=$(CURDIR)/$(PROTOC_GEN_GO) \
		--plugin=protoc-gen-go-grpc=$(CURDIR)/$(PROTOC_GEN_GO_GRPC) \
		--go_out=. --go_opt=module=$(MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
		$(PROTO_FILES)

$(PROTOC_GEN_GO):
	mkdir -p .tools
	go build -o $@ google.golang.org/protobuf/cmd/protoc-gen-go

$(PROTOC_GEN_GO_GRPC):
	mkdir -p .tools
	go build -o $@ google.golang.org/grpc/cmd/protoc-gen-go-grpc

build:
	go build ./...

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

migrate-up:
	docker compose --env-file .env run --rm migrate

migrate-down:
	docker compose --env-file .env --profile tools run --rm migrate-down

docker-up:
	docker compose --env-file .env up --build

docker-down:
	docker compose --env-file .env down
