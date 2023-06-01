package command

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

func UnInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "UnInstall go versions",
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
			gvmUnInstall(args[0])
		},
	}

	return cmd
}

func gvmUnInstall(version string) {
	currentVersion := strings.TrimPrefix(getCurrentGoVersion(), "go")
	if currentVersion == version {
		fmt.Fprintln(os.Stderr, "Couldn't uninstall go version. because the version is used")
		return
	}

	goRootPath := path.Join(gvmRootPath, fmt.Sprintf("go%s", version))
	if err := os.RemoveAll(goRootPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to uninstall go version. err: %v", err)
		return
	}

	goPkgPath := path.Join(gvmPkgPath, goPkgName(version))
	if err := os.Remove(goPkgPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to uninstall go version. err: %v", err)
		return
	}

	fmt.Fprintln(os.Stdout, "Success")
}
