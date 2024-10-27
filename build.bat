@echo off
setlocal
pushd "%~dp0"
set GOAMD64=v3
go build -trimpath -asmflags -trimpath -gcflags="all=-C -dwarf=false" -ldflags="-s -w"
popd
