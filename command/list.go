package command

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	ListRemote bool
}

func List() *cobra.Command {
	opts := &ListOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list installed go versions",
		Run: func(cmd *cobra.Command, args []string) {
			err := listPgks(opts.ListRemote)

			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31m获取Golang版本列表失败.\033[0m err: %v\n", err)
			}
		},
	}

	cmd.Flags().BoolVarP(&opts.ListRemote, "remote", "", opts.ListRemote, "if list remote available versions. default is false")
	return cmd
}

func listPgks(isRemotePkgs bool) error {
	if isRemotePkgs {
		return listRemotePkgs()
	}

	return listLocalPkgs()
}

// listLocalPkgs 获取已安装的Go版本
func listLocalPkgs() error {
	fmt.Println("\ngvm gos (installed)")
	fmt.Println()

	pkgs, err := os.ReadDir(gvmRootPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	currentGoVersion := getCurrentGoVersion()
	for _, pkg := range pkgs {
		if !pkg.IsDir() {
			continue
		}

		if pkg.Name() == currentGoVersion {
			fmt.Printf("\033[32m=> %s\033[0m\n", pkg.Name())
			continue
		}
		fmt.Printf("   %s\n", pkg.Name())
	}
	return nil
}

// listRemotePkgs 获取已发布的Go版本
func listRemotePkgs() error {
	customClient := &http.Client{
		// 10 second timeout
		Timeout: 10 * time.Second,
		// 不允许重定向
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
	}
	client.InstallProtocol("https", githttp.NewClient(customClient))

	rem := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin", URLs: []string{"https://github.com/golang/go"},
	})
	refs, err := rem.List(&git.ListOptions{PeelingOption: git.AppendPeeled})
	if err != nil {
		return err
	}

	// 版本号正则表达式
	versionReg := regexp.MustCompile(`^go(.*)[a-zA-Z]`)

	// 获取Tag列表
	var tags Tags
	for _, ref := range refs {
		if !ref.Name().IsTag() {
			continue
		}

		shortName := ref.Name().Short()
		if strings.HasPrefix(shortName, "go1") && !versionReg.MatchString(shortName) {
			tags = append(tags, shortName)
		}
	}
	sort.Sort(tags)

	fmt.Println("\ngvm gos (available)")
	fmt.Println()
	for _, tag := range tags {
		fmt.Printf("   %s\n", tag)
	}
	return nil
}

// getCurrentGoVersion 当前Go版本
func getCurrentGoVersion() string {
	data, err := os.Readlink(envGoROOT)
	if err != nil {
		return ""
	}
	parts := strings.Split(data, "/")
	return parts[len(parts)-1]
}

type Tags []string

func (tag Tags) Len() int {
	return len(tag)
}

func (tag Tags) Less(i, j int) bool {
	tagVi := tag.strToSlice(i)
	tagVj := tag.strToSlice(j)
	for index := 0; index < len(tagVi) || index < len(tagVj); index++ {
		if index == len(tagVi) {
			return true
		}
		if index == len(tagVj) {
			return false
		}

		v1, _ := strconv.Atoi(tagVi[index])
		v2, _ := strconv.Atoi(tagVj[index])
		if v1 == v2 {
			continue
		}
		if v1 < v2 {
			return true
		}

		return false
	}
	return false
}

func (tag Tags) Swap(i, j int) {
	tag[i], tag[j] = tag[j], tag[i]
}

func (tag Tags) strToSlice(i int) []string {
	return strings.Split(strings.TrimPrefix(tag[i], "go"), ".")
}
