#ARCH
ARCH="`uname -s`"
LINUX="Linux"
Darwin="Darwin"
tag="latest"

build:
	@if [ $(ARCH) = $(LINUX) ]; \
    	then \
    		go build -o gvm -tags 'netgo osusergo' -ldflags '-extldflags "-static"' main.go; \
    	elif [ $(ARCH) = $(Darwin) ]; \
    	then \
    		GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o gvm -ldflags '-s -extldflags "-sectcreate __TEXT __info_plist Info.plist"' main.go; \
    	else \
    		echo "ARCH unknow"; \
    	fi
