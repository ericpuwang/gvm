# gvm
Go Version Manager


## 安装gvm

```shell
# Ensure GVM_VERSION is set as above
GVM_PLATFORM=linux_amd64 # also supported: linux_arm64, darwin_amd64, darwin_arm64
curl -L https://github.com/periky/gvm/releases/download/${GVM_VERSION}/gvm_${GVM_PLATFORMPLA} -o ./gvm
chmod +x ./gvm
```

## 设置环境变量

```shell
export GOROOT=$HOME/.gvm/go
export PATH=$PATH:$GOROOT/bin:$HOME/go/bin

# 命令行补全[可选]
## gvm completion -h
source <(gvm completion zsh)
```

## 功能

### 列出已安装的Golang版本
```shell
./gvm list
```

### 列出Golang所有版本(不包含rc和beta版)
```shell
./gvm list --remote
```

### 安装指定版本
```shell
# 安装golang 1.20.4
./gvm install -v 1.20.4
```

### 设置默认版本
```shell
# use命令支持参数补全
./gvm use 1.20.4
```