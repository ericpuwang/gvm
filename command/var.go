package command

import (
	"os"
	"path"
)

var (
	// gvm基础目录
	gvmBasePath = path.Join(os.Getenv("HOME"), ".gvm")
	// golang安装包目录
	gvmPkgPath = path.Join(gvmBasePath, "archive")
	// golang版本root目录
	gvmRootPath = path.Join(gvmBasePath, "goroots")
	// 当前go版本目录的软连接
	envGoROOT = path.Join(gvmBasePath, "go")
)

// 环境变量名称
const (
	gvmGoName = "GVM_GO_NAME"
)
