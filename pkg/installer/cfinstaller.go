package installer

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log" // [新增] 导入 log 包，用于输出更友好的信息
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings" // [新增] 导入 strings 包
)

// ReleaseAsset 关键信息
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}
type ReleaseInfo struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}

// Installer 检测并安装 CloudflareSpeedTest
type Installer struct {
	proxy     string
	apiURL    string
	binPath   string
	cacheFile string
}

// NewInstaller 构造
func NewInstaller(proxy, apiURL, binPath string) *Installer {
	return &Installer{
		proxy:     proxy,
		apiURL:    apiURL,
		binPath:   binPath,
		cacheFile: binPath + ".version",
	}
}

// InstallOrUpdate 若版本变更则下载并安装
func (i *Installer) InstallOrUpdate() error {
	api := i.apiURL
	if i.proxy != "" {
		api = i.proxy + api
	}
	resp, err := http.Get(api)
	if err != nil {
		return fmt.Errorf("fetch release info: %w", err)
	}
	defer resp.Body.Close()

	var info ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return fmt.Errorf("decode release: %w", err)
	}

	// 版本未变则跳过
	if data, err := os.ReadFile(i.cacheFile); err == nil && string(data) == info.TagName {
		log.Println("CloudflareSpeedTest is already the latest version:", info.TagName)
		return nil
	}
	
	log.Println("New CloudflareSpeedTest version found:", info.TagName)

	// [核心修改] 修正目标文件名的拼接格式
	// 原始格式: linux_amd64.tar.gz
	// 新格式: cfst_linux_amd64.tar.gz
	targetFilename := fmt.Sprintf("cfst_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	var assetURL string
	
	// 为了更强的兼容性，我们不再做完全匹配，而是检查文件名是否包含我们需要的架构信息
	archIdentifier := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	for _, a := range info.Assets {
		// 使用 strings.Contains 进行模糊匹配，以防未来命名再次变化
		if strings.Contains(a.Name, archIdentifier) && strings.HasSuffix(a.Name, ".tar.gz") {
			assetURL = a.BrowserDownloadURL
			targetFilename = a.Name // 直接使用 API 返回的正确文件名
			log.Println("Found matching asset:", targetFilename)
			break
		}
	}

	if assetURL == "" {
		return fmt.Errorf("asset for %s not found in release assets", archIdentifier)
	}

	dlURL := assetURL
	if i.proxy != "" {
		dlURL = i.proxy + assetURL
	}
	
	log.Println("Downloading from:", dlURL)

	// 下载到临时目录
	tmp := filepath.Join(os.TempDir(), targetFilename)
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	
	resp2, err := http.Get(dlURL)
	if err != nil {
		out.Close()
		return fmt.Errorf("download asset: %w", err)
	}
	defer resp2.Body.Close()

	_, err = io.Copy(out, resp2.Body)
	out.Close()
	if err != nil {
		return fmt.Errorf("save asset: %w", err)
	}

	// 解压到 binPath
	log.Println("Unpacking", tmp, "to", i.binPath)
	if err := unpackTarGz(tmp, i.binPath); err != nil {
		return fmt.Errorf("unpack: %w", err)
	}
	log.Println("Unpack successful.")

	// 写版本缓存
	if err := os.WriteFile(i.cacheFile, []byte(info.TagName), 0644); err != nil {
		return err
	}
	log.Println("Version cache updated.")
	return nil
}

// unpackTarGz 解压文件
func unpackTarGz(archive, dest string) error {
    // ... (此函数保持不变) ...
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// 我们只解压普通文件，并假设压缩包内第一个文件就是我们要的可执行程序
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// 确保目标目录存在
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		// 创建并写入文件，赋予可执行权限
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		out.Close()
		// 成功解压第一个文件后即可返回
		return nil
	}
	return fmt.Errorf("no valid file found in archive")
}