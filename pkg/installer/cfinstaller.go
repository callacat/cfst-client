package installer

import (
    "archive/tar"
    "compress/gzip"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "runtime"
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
        return nil
    }

    // 寻找对应平台的资产
    target := fmt.Sprintf("%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
    var assetURL string
    for _, a := range info.Assets {
        if a.Name == target {
            assetURL = a.BrowserDownloadURL
            break
        }
    }
    if assetURL == "" {
        return fmt.Errorf("asset %s not found", target)
    }

    dlURL := assetURL
    if i.proxy != "" {
        dlURL = i.proxy + assetURL
    }

    // 下载到临时
    tmp := filepath.Join(os.TempDir(), "cfspeed.tar.gz")
    out, err := os.Create(tmp)
    if err != nil {
        return err
    }
    if resp2, err := http.Get(dlURL); err != nil {
        out.Close()
        return fmt.Errorf("download asset: %w", err)
    } else {
        _, err = io.Copy(out, resp2.Body)
        resp2.Body.Close()
        out.Close()
        if err != nil {
            return fmt.Errorf("save asset: %w", err)
        }
    }

    // 解压到 binPath
    if err := unpackTarGz(tmp, i.binPath); err != nil {
        return fmt.Errorf("unpack: %w", err)
    }

    // 写版本缓存
    if err := os.WriteFile(i.cacheFile, []byte(info.TagName), 0644); err != nil {
        return err
    }
    return nil
}

func unpackTarGz(archive, dest string) error {
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
        if hdr.Typeflag != tar.TypeReg {
            continue
        }
        out, err := os.OpenFile(dest, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
        if err != nil {
            return err
        }
        if _, err := io.Copy(out, tr); err != nil {
            out.Close()
            return err
        }
        out.Close()
        return nil
    }
    return fmt.Errorf("no file in archive")
}
