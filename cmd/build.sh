#!/bin/bash

GOOS=windows GOARCH=amd64 go build -o 导出dxf门窗尺寸.exe -ldflags "-s -w"