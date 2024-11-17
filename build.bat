@echo off
setlocal
pushd "%~dp0"
set GOTELEMETRY=off
set GOAMD64=v3
go build -trimpath -gcflags="all=-C -dwarf=false" -ldflags="-s -w -buildid=" -buildvcs=false
popd
