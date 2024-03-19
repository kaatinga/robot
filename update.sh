#!/bin/bash

go list -m -u -f '{{if not (or .Indirect .Main)}}{{.Path}}@latest{{end}}' all | xargs go get
go mod tidy -compat=1.18