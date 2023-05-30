package command

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

func Use() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <version>",
		Short: "Select a go version to use",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("%q only support arguments <version>, got %q", cmd.CommandPath(), args)
			}
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 1 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			pkgs, err := os.ReadDir(gvmRootPath)
			if err != nil && !os.IsNotExist(err) {
				return nil, cobra.ShellCompDirectiveDefault
			}

			var comps []string
			for _, pkg := range pkgs {
				if !pkg.IsDir() {
					continue
				}
				comps = append(comps, strings.TrimPrefix(pkg.Name(), "go"))
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(cmd *cobra.Command, args []string) {
			gvmUse(args[0])
		},
	}

	return cmd
}

func gvmUse(version string) {
	goRoot := path.Join(gvmRootPath, fmt.Sprintf("go%s", version))
	if _, err := os.Stat(goRoot); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "\033[31m%s is not installed.\033[0m Install it by running 'gvm install %s'\n", version, version)
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
