package command

import (
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
				fmt.Fprintf(os.Stderr, "\033[31m请指定版本号\033[0m\n")
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
			fmt.Fprintf(os.Stderr, "\033[31m%s is not installed.\033[0m Install it by running 'gvm install %s'\n", options.Version, options.Version)
			return
		}
		fmt.Fprintf(os.Stderr, "\033[31m获取GOROOT信息失败.\033[0m err: %v\n", err)
		return
	}

	// 配置GOROOT
	if err := forceSymlink(goRoot, envGoROOT); err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m设置GOROOT目录失败.\033[0m err: %v\n", err)
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
