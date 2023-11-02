package command

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

type InstallOptions struct {
	Source  string
	Version string
}

func Install() *cobra.Command {
	opts := &InstallOptions{
		Source: "https://storage.googleapis.com/golang",
	}
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install go version",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.Version = strings.TrimPrefix(opts.Version, "go")
			if opts.Version == "" {
				return errors.New("version must be not empty")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := download(opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31m下载golang %s失败.\033[0m err: %v\n", opts.Version, err)
				return
			}

			err = extract(opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31m文件解压失败.\033[0m err: %v\n", err)
				return
			}

			gvmUse(opts.Version)
		},
	}

	cmd.Flags().StringVarP(&opts.Source, "source", "s", opts.Source, "Install Go from specified source.")
	cmd.Flags().StringVarP(&opts.Version, "version", "v", opts.Version, "Go version")

	return cmd
}

func goPkgName(version string) string {
	ext := "tar.gz"

	return fmt.Sprintf("go%s.%s-%s.%s", version, runtime.GOOS, runtime.GOARCH, ext)
}

// download 下载golang安装包
func download(options *InstallOptions) error {
	filename := goPkgName(options.Version)
	downloader := NewFileDownloader(fmt.Sprintf("%s/%s", options.Source, filename), gvmPkgPath, filename, 10)
	if err := downloader.Run(); err != nil {
		return err
	}
	return nil
}

// extract 解压缩文件
func extract(options *InstallOptions) error {
	goRoot := path.Join(gvmRootPath, fmt.Sprintf("go%s", options.Version))
	_, err := os.Stat(goRoot)
	if err == nil {
		if err := os.RemoveAll(goRoot); err != nil {
			return err
		}
	}
	if os.IsNotExist(err) {
		if err := os.MkdirAll(goRoot, os.ModePerm); err != nil {
			return err
		}
	}

	pkgAbsPath := path.Join(gvmPkgPath, goPkgName(options.Version))
	tarFile, err := os.Open(pkgAbsPath)
	if err != nil {
		return err
	}
	defer func() { _ = tarFile.Close() }()

	gr, err := gzip.NewReader(tarFile)
	if err != nil {
		return err
	}
	defer func() { _ = gr.Close() }()

	tarReader := tar.NewReader(gr)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.Contains(header.Name, "..") {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(path.Join(goRoot, strings.TrimPrefix(header.Name, "go/")), header.FileInfo().Mode())
			if err != nil {
				return err
			}
		case tar.TypeReg:
			destFileName := path.Join(goRoot, strings.TrimPrefix(header.Name, "go/"))
			dest, err := os.Create(destFileName)
			if err != nil {
				return err
			}
			_ = os.Chmod(destFileName, header.FileInfo().Mode())

			_, err = io.Copy(dest, tarReader)
			if err != nil {
				return err
			}

			_ = dest.Close()
		default:
			return fmt.Errorf("unable to extract %s", header.Name)
		}
	}

	return nil
}

// FileDownloader 文件下载器
type FileDownloader struct {
	fileSize       int
	url            string
	outputFileName string
	totalPart      int //下载线程
	outputDir      string
	doneFilePart   []filePart
}

// filePart 文件分片
type filePart struct {
	Index int           //文件分片的序号
	From  int           //开始byte
	To    int           //结束byte
	Body  io.ReadCloser // 文件内容
}

// NewFileDownloader .
func NewFileDownloader(url, outputDir, outputFileName string, totalPart int) *FileDownloader {
	return &FileDownloader{
		fileSize:       0,
		url:            url,
		outputDir:      outputDir,
		outputFileName: outputFileName,
		totalPart:      totalPart,
		doneFilePart:   make([]filePart, totalPart),
	}
}

// Run 开始下载任务
func (d *FileDownloader) Run() error {
	if _, err := os.Stat(path.Join(d.outputDir, d.outputFileName)); err == nil {
		if err := os.Remove(path.Join(d.outputDir, d.outputFileName)); err != nil {
			return err
		}
	}

	req, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return errors.New("invalid version")
	}

	err = d.writer(resp)
	if err != nil {
		return err
	}

	return nil
}

// getNewRequest 创建一个request
func (d *FileDownloader) getNewRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(
		method,
		d.url,
		nil,
	)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "go-version-manager")
	return r, nil
}

// mergeFileParts 合并下载的文件
func (d *FileDownloader) writer(resp *http.Response) error {
	// 目录outputDir不存在则创建
	_, err := os.Stat(d.outputDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(d.outputDir, os.ModePerm)
	}
	if err != nil {
		return err
	}

	name := filepath.Join(d.outputDir, d.outputFileName)
	f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()


	bar := progressbar.NewOptions(int(resp.ContentLength),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription(fmt.Sprintf("正在下载[\033[32m%s\033[0m]", d.outputFileName)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	if err != nil {
		return err
	}

	return nil
}
