#!/bin/bash
set -eux

# go files
find services -iname '*.proto' -exec protoc --proto_path=./ --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. {} \;

# grpc-gateway for all services
find services -iname '*.proto' -exec protoc --proto_path=./ --grpc-gateway_out . --grpc-gateway_opt logtostderr=true --grpc-gateway_opt generate_unbound_methods=true --grpc-gateway_opt paths=source_relative {} \;

# swagger for all services
find services  -iname '*.proto' -exec protoc --proto_path=./ --openapiv2_out . --openapiv2_opt logtostderr=true --openapiv2_opt generate_unbound_methods=true {} \;

#echo "Please git add any files you want to keep now, pressing any key to continue will run a git clean -f" && read
#git clean -f .
