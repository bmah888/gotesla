Tesla Powerwall Device Vitals
=============================

One endpoint in the Tesla Powerwall API is /api/device/vitals that
returns lots of interesting data about the devices in the Powerwall
system.  The return value is base64-encoded binary data packed in a
protocol buffers format.  The specific format was reverse engineered
by @brianhealy and @jasonacox into the tesla.proto file released in
https://github.com/jasonacox/pypowerwall.  That file has been adapted
here for use with [golang](https://golang.org).

Building tesla.pb.go
--------------------

To build the tesla.pb.go library from the tesla.proto file requires
installation of the protocol buffers compiler `protoc` plus the
`protoc-gen-go` plugin to generate Go-specific code.

One way to install the `protoc` compiler is to build it from sources
at https://github.com/protocolbuffers/protobuf.  It may also be
available from package managers.

The `protoc-gen-go` plugin is obtained from
https://github.com/protocolbuffers/protobuf-go and can be installed
using the command
> go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

This will install the plugin in $GOBIN (e.g., $HOME/go/bin) which must
be added to $PATH for the compiler to find it.

Finally, with the compiler installed, build the tesla.pb.go library here:
> protoc --proto_path=.. --go_out=. --go_opt=paths=source_relative tesla.proto
