module github.com/m-this/tf2-archipelago

go 1.27.3

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.12-20260825204119-511051f7f437.2
	connectrpc.com/connect v1.21.0
	connectrpc.com/grpchealth v1.5.0
	connectrpc.com/validate v0.9.0
	fyne.io/systray v1.12.2
	github.com/coder/websocket v1.8.15
	github.com/m-this/tf2-mvm-bots-go v0.19.0
	go.yaml.in/yaml/v3 v3.0.5
	golang.org/x/sys v0.49.0
	google.golang.org/protobuf v1.36.12
)

require (
	buf.build/go/protovalidate v1.4.0 // indirect
	cel.dev/cel-go v0.32.0 // indirect
	cel.dev/expr v0.25.3 // indirect
	connectrpc.com/connect/v2 v2.0.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/godbus/dbus/v5 v5.2.4 // indirect
	golang.org/x/exp v0.0.0-20261013213113-2ab7f3cbc5fb // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20261012220534-4341edf3f941 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261012220534-4341edf3f941 // indirect
)

tool github.com/m-this/tf2-mvm-bots-go/cmd/gen
