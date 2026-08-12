module github.com/protosio/protos/network/modules/ipc

go 1.26.4

toolchain go1.26.5

require (
	github.com/protosio/protos v0.0.0
	google.golang.org/grpc v1.83.0
)

require (
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/protosio/protos => ../../..
