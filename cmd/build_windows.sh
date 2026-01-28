#!/bin/bash

name="导出dxf门窗尺寸v1.0.1.exe"

echo "Building..."

export GOOS=windows
export GOARCH=amd64

# 把png转成ico：https://www.aconvert.com/cn/icon/png-to-ico
# 生成go编译时可嵌入的图标文件
# go get github.com/akavel/rsrc
rsrc -manifest ico.manifest -o ico.syso -ico icon.ico

# 编译成gui程序并隐藏启动
go build -o "$name" -ldflags "-s -w -H windowsgui"

rm ico.syso

# 压缩程序
#upx --brute "$name"