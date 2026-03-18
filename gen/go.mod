module github.com/weoses/memelo/gen

go 1.25.0

require (
	connectrpc.com/connect v1.19.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	google.golang.org/protobuf/cmd/protoc-gen-go
)
