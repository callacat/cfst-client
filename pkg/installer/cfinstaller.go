package installer

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ... ReleaseAsset 和 ReleaseInfo 结构体保持不变 ...
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}
type ReleaseInfo struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}


// [修改] Installer 结构体增加 configDir 字段
type Installer struct {
	proxy     string
	apiURL    string
	binPath   string
	configDir string // 新增：用于存放 ip.txt 等配置文件
	cacheFile string
}

// [修改] NewInstaller 构造函数增加 configDir 参数
func NewInstaller(proxy, apiURL, binPath, configDir string) *Installer {
	return &Installer{
		proxy:     proxy,
		apiURL:    apiURL,
		binPath:   binPath,
		configDir: configDir,
		cacheFile: binPath + ".version",
	}
}

// InstallOrUpdate 方法保持不变，但会把 configDir 传递给解压函数
func (i *Installer) InstallOrUpdate() error {
    // ... 此方法前面的逻辑保持不变 ...

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

	if data, err := os.ReadFile(i.cacheFile); err == nil && string(data) == info.TagName {
		log.Println("CloudflareSpeedTest is already the latest version:", info.TagName)
		return nil
	}
	
	log.Println("New CloudflareSpeedTest version found:", info.TagName)

	targetFilename := fmt.Sprintf("cfst_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	var assetURL string
	archIdentifier := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	for _, a := range info.Assets {
		if strings.Contains(a.Name, archIdentifier) && strings.HasSuffix(a.Name, ".tar.gz") {
			assetURL = a.BrowserDownloadURL
			targetFilename = a.Name
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

	// [修改] 将 configDir 传递给解压函数
	log.Println("Unpacking archive to specified directories...")
	if err := i.unpackTarGz(tmp); err != nil {
		return fmt.Errorf("unpack: %w", err)
	}
	log.Println("Unpack successful.")

	if err := os.WriteFile(i.cacheFile, []byte(info.TagName), 0644); err != nil {
		return err
	}
	log.Println("Version cache updated.")
	return nil
}

// [核心重构] unpackTarGz 现在可以解压多个文件到不同目录
func (i *Installer) unpackTarGz(archive string) error {
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
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(i.binPath), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.configDir, 0755); err != nil {
		return err
	}

	foundFiles := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		var destPath string
		var filePerm os.FileMode

		// 根据文件名判断解压路径和权限
		switch hdr.Name {
		case "ip.txt", "ipv6.txt":
			destPath = filepath.Join(i.configDir, hdr.Name)
			filePerm = 0644 // 普通文件权限
			log.Printf("Extracting '%s' to '%s'", hdr.Name, destPath)
		case "cfst": // 假设可执行文件名是 cfst
			destPath = i.binPath
			filePerm = 0755 // 可执行文件权限
			log.Printf("Extracting executable to '%s'", destPath)
		default:
			// 如果还有其他可执行文件名（例如 Windows 下的 cfst.exe）
			if strings.HasPrefix(hdr.Name, "cfst") {
				destPath = i.binPath
				filePerm = 0755
				log.Printf("Extracting executable '%s' to '%s'", hdr.Name, destPath)
			} else {
				// 忽略其他未知文件
				continue
			}
		}

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, filePerm)
		if err != nil {
			return err
		}

		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		out.Close()
		foundFiles++
	}

	if foundFiles == 0 {
		return fmt.Errorf("no valid files (executable, ip.txt, ipv6.txt) found in archive")
	}

	return nil
}