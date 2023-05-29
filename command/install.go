package command

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

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
		Short: "Install Go from specified version.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.Version == "" {
				return errors.New("version must be not empty")
			}
			if runtime.GOOS == "windows" {
				return errors.New("unsupported os: windows")
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

			gvmUse(&UseOptions{Version: opts.Version})
		},
	}

	cmd.Flags().StringVarP(&opts.Source, "source", "s", opts.Source, "Install Go from specified source.")
	cmd.Flags().StringVarP(&opts.Version, "version", "v", opts.Version, "Go version")

	return cmd
}

func goPkgName(options *InstallOptions) string {
	ext := "tar.gz"

	return fmt.Sprintf("go%s.%s-%s.%s", options.Version, runtime.GOOS, runtime.GOARCH, ext)
}

func setFileDescriptors() {

}

// download 下载golang安装包
func download(options *InstallOptions) error {
	filename := goPkgName(options)
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

	filepath := path.Join(gvmPkgPath, goPkgName(options))
	tarFile, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gr, err := gzip.NewReader(tarFile)
	if err != nil {
		return err
	}
	defer gr.Close()

	tarReader := tar.NewReader(gr)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
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
			os.Chmod(destFileName, header.FileInfo().Mode())

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
	Index int    //文件分片的序号
	From  int    //开始byte
	To    int    //结束byte
	Data  []byte //http下载得到的文件内容
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

// hasAcceptRanges 获取要下载的文件的基本信息(header) 使用HTTP Method Head
func (d *FileDownloader) hasAcceptRanges() (int, bool, error) {
	r, err := d.getNewRequest("HEAD")
	if err != nil {
		return 0, false, err
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return 0, false, err
	}
	if resp.StatusCode > 299 {
		return 0, false, fmt.Errorf("can't process, response is %v", resp.StatusCode)
	}

	//检查是否支持 断点续传
	//https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Accept-Ranges
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return 0, false, nil
	}

	//https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Length
	contentLength, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	return contentLength, true, err
}

// Run 开始下载任务
func (d *FileDownloader) Run() error {
	if _, err := os.Stat(path.Join(d.outputDir, d.outputFileName)); err == nil {
		if err := os.Remove(path.Join(d.outputDir, d.outputFileName)); err != nil {
			return err
		}
	}
	fileTotalSize, ok, err := d.hasAcceptRanges()
	if err != nil {
		return err
	}
	d.fileSize = fileTotalSize
	if !ok {
		d.totalPart = 1
	}

	jobs := make([]filePart, d.totalPart)
	eachSize := fileTotalSize / d.totalPart

	for i := range jobs {
		jobs[i].Index = i
		if i == 0 {
			jobs[i].From = 0
		} else {
			jobs[i].From = jobs[i-1].To + 1
		}
		if i < d.totalPart-1 {
			jobs[i].To = jobs[i].From + eachSize
		} else {
			//the last filePart
			jobs[i].To = fileTotalSize - 1
		}
	}

	var wg sync.WaitGroup
	fmt.Fprintf(os.Stdout, "\033[32m开始下载%s...\033[0m\n", d.outputFileName)
	errs := make(chan error, d.totalPart)
	for _, j := range jobs {
		wg.Add(1)
		go func(job filePart, errs chan error) {
			defer wg.Done()
			err := d.downloadPart(job)
			if err != nil {
				errs <- err
			}
		}(j, errs)

	}
	wg.Wait()
	close(errs)

	err = <-errs
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m下载文件失败\033[0m: err: %v\n", err)
	}

	return d.mergeFileParts()
}

// 下载分片
func (d FileDownloader) downloadPart(c filePart) error {
	r, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}
	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", c.From, c.To))
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}}
	resp, err := client.Do(r)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		return fmt.Errorf("服务器错误状态码: %v", resp.StatusCode)
	}
	defer resp.Body.Close()
	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(bs) != (c.To - c.From + 1) {
		return errors.New("下载文件分片长度错误")
	}
	c.Data = bs
	d.doneFilePart[c.Index] = c
	return nil

}

// getNewRequest 创建一个request
func (d FileDownloader) getNewRequest(method string) (*http.Request, error) {
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
func (d FileDownloader) mergeFileParts() error {
	// 目录outputDir不存在则创建
	_, err := os.Stat(d.outputDir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(d.outputDir, os.ModePerm)
	}
	if err != nil {
		return err
	}
	path := filepath.Join(d.outputDir, d.outputFileName)
	mergedFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer mergedFile.Close()
	hash := sha256.New()
	totalSize := 0
	for _, s := range d.doneFilePart {

		mergedFile.Write(s.Data)
		hash.Write(s.Data)
		totalSize += len(s.Data)
	}
	if totalSize != d.fileSize {
		return errors.New("文件不完整")
	}
	fmt.Fprintln(os.Stdout, "\033[32m下载完成\033[0m")

	return nil
}
