package command

import (
	"bufio"
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
)

type UseOptions struct {
	Version string
}

func Use() *cobra.Command {
	opts := &UseOptions{}
	cmd := &cobra.Command{
		Use:   "use",
		Short: "Select a go version to use",
		Run: func(cmd *cobra.Command, args []string) {
			if opts.Version == "" {
				fmt.Fprintf(os.Stderr, "\033[31m请指定版本号\033[0m")
				return
			}

			gvmUse(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Version, "version", "v", opts.Version, "Go version")

	return cmd
}

func gvmUse(options *UseOptions) {
	goRoot := path.Join(gvmRootPath, fmt.Sprintf("go%s", options.Version))
	if _, err := os.Stat(goRoot); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "\033[31m%s is not installed.\033[0m Install it by running 'gvm install %s'", options.Version, options.Version)
			return
		}
		fmt.Fprintf(os.Stderr, "获取GOROOT信息失败. err: %v", err)
		return
	}

	// 配置GOROOT
	if err := forceSymlink(goRoot, envGoROOT); err != nil {
		fmt.Fprintf(os.Stderr, "设置GOROOT目录失败. err: %v", err)
	}
	// 环境变量
	if err := writeEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "配置环境变量失败. err: %v\n\texport GOROOT=$HOME/.gvm/go", err)
	}
}

func forceSymlink(oldname, newname string) error {
	if _, err := os.Lstat(newname); err == nil {
		os.Remove(newname)
	}
	err := os.Symlink(oldname, newname)
	if err != nil {
		return err
	}

	return nil
}

func writeEnv() error {
	v, exist := os.LookupEnv("GOROOT")
	if exist {
		fmt.Fprintf(os.Stdout, "\033[31mWARN:\033[0m环境变量GOROOT已经存在.GOROOT=%s", v)
		return nil
	}

	filepath := path.Join(os.Getenv("HOME"), ".bashrc")
	if os.Getenv("SHELL") == "/bin/zsh" {
		filepath = path.Join(os.Getenv("HOME"), ".zshrc")
	}

	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	write := bufio.NewWriter(file)
	write.WriteString(`
# Go Version Manager
# github.com/periky/gvm
export GOROOT=$HOME/.gvm/go
export PATH=$PATH:$GOROOT/bin:$HOME/go/bin
`)
	err = write.Flush()
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "Installed Go Version Manager")
	fmt.Fprintln(os.Stdout, "Please restart your terminal session or to get started right away run")
	fmt.Fprintf(os.Stdout, "   \033[32msource %s\033[0m\n", filepath)
	return nil
}
