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

// ... (ReleaseAsset 和 ReleaseInfo 结构体保持不变) ...
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}
type ReleaseInfo struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}

type Installer struct {
	proxy     string
	apiURL    string
	binPath   string
	configDir string
	cacheFile string
}

func NewInstaller(proxy, apiURL, binPath, configDir string) *Installer {
	return &Installer{
		proxy:     proxy,
		apiURL:    apiURL,
		binPath:   binPath,
		configDir: configDir,
		cacheFile: binPath + ".version",
	}
}

// ... (InstallOrUpdate 方法保持不变) ...
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


// [核心修正] unpackTarGz 现在精确查找名为 'cfst' 的二进制文件
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
	if err := os.MkdirAll(filepath.Dir(i.binPath), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(i.configDir, 0755); err != nil {
		return err
	}

	executableFound := false
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

		switch hdr.Name {
		case "ip.txt", "ipv6.txt":
			destPath = filepath.Join(i.configDir, hdr.Name)
			filePerm = 0644
			log.Printf("Extracting '%s' to '%s'", hdr.Name, destPath)
		// [核心修正] 我们只接受名为 "cfst" 的文件作为可执行程序
		case "cfst":
			destPath = i.binPath
			filePerm = 0755
			log.Printf("Extracting executable '%s' to '%s'", hdr.Name, destPath)
			executableFound = true
		default:
			// 忽略所有其他文件，特别是那个同名的脚本
			continue
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
	}

	if !executableFound {
		return fmt.Errorf("executable 'cfst' not found in archive")
	}

	return nil
}