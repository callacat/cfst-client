# cfst-client

`cfst-client` 是一个自动化的 Cloudflare 速度测试客户端，旨在定期运行速度测试并将结果聚合上传到指定的 GitHub Gist。它被打包为一个易于部署的 Docker 容器，并支持丰富的自定义配置。

## ✨ 功能特性

  * **🚀 自动化测试**: 使用 Cron 表达式定义测试任务的执行周期。
  * **🌐 支持双栈**: 可同时对 IPv4 和 IPv6 进行速度测试。
  * **💾 Gist 结果聚合**: 将格式化后的测试结果自动上传并覆盖到 GitHub Gist，便于多设备结果汇总。
  * **🔄 健壮的重试机制**: 当测试结果不足时，支持多次即时重试；当所有即时重试失败后，还可启用延迟重试。
  * **📦 Docker 化部署**: 提供 `linux/amd64` 和 `linux/arm64` 架构的 Docker 镜像，方便在各种设备和 NAS 系统上运行。
  * **🤖 自动更新**: 能够自动检查并下载最新版本的 [XIU2/CloudflareSpeedTest](https://github.com/XIU2/CloudflareSpeedTest) 核心程序。
  * \*\* CI/CD\*\*: 通过 GitHub Actions 实现了全自动的版本管理、PR 合并、二进制文件构建和 Docker 镜像发布。

## 🚀 快速开始

### Docker (推荐)

本项目通过 Docker 运行。您需要准备一个配置文件，并通过环境变量传入敏感信息。

1.  **准备配置文件目录**
    在您的主机上创建一个目录，用于存放配置文件。

    ```bash
    mkdir -p /path/to/your/config
    ```

2.  **创建 `config.yml`**
    在上述目录中创建一个 `config.yml` 文件。您可以从项目中的 `config/config.yml` 文件开始。

3.  **运行 Docker 容器**
    使用以下命令启动容器。请确保将 `/path/to/your/config` 替换为您自己的配置目录，并填充您的环境变量。

    ```bash
    docker run -d \
      --name cfst-client \
      -v /path/to/your/config:/app/config \
      -e GITHUB_TOKEN="ghp_YourGitHubToken" \
      -e TELEGRAM_BOT_TOKEN="YourTelegramBotToken" \
      -e TELEGRAM_CHAT_ID="YourTelegramChatID" \
      --restart always \
      callacat/cfst-client:latest
    ```

      * `GITHUB_TOKEN`: 用于 Gist 上传的 GitHub Personal Access Token。
      * `TELEGRAM_BOT_TOKEN` (可选): Telegram Bot 的 Token。
      * `TELEGRAM_CHAT_ID` (可选): 要发送通知的 Telegram Chat ID。

### Windows (本地运行)
除了 Docker, 您也可以直接在 Windows 系统上运行预编译的 .exe 程序。

1.  **下载可执行文件**
前往本项目的 GitHub Releases 页面，下载最新的 Windows 版本，例如 cfst-client-windows-amd64.exe。

2.  **下载cfst**

请前往 CloudflareSpeedTest 的官方 [Releases](https://github.com/XIU2/CloudflareSpeedTest/releases) 页面，根据您的系统架构，下载最新的 Windows 版本压缩包（例如 cfst_windows_amd64.zip）
解压您下载的 .zip 文件，您会得到一个 cfst.exe 文件。

为了方便管理，建议您将这个 cfst.exe 文件直接放到您之前创建的 D:\app\config\ 文件夹中。

3.  **创建配置文件夹**

    > 注意: 这是关键的一步。由于程序目前将配置目录硬编码为 /app/config，您必须在当前盘根目录下创建这个文件夹结构。

    例如程序放在 D 盘任意目录下，打开文件资源管理器，在 D:\ 下创建 app 文件夹，然后在 app 文件夹内创建 config 文件夹。最终路径为 D:\app\config。

4.  **创建 config.yml 文件**
将您的 config.yml 配置文件放置在 D:\app\config 目录下。**注意**：找到 cf 和 cf6 这两个部分，将其中的 binary 字段修改为您刚刚放置的 cfst.exe 的完整 Windows 路径。
提示: 在 YAML 文件中，路径使用正斜杠 / 是最安全的方式，可以避免反斜杠 \ 的转义问题。

5.  **运行程序**
打开一个终端（PowerShell 或 CMD），进入您下载 .exe 文件的目录，然后运行程序。在运行前，需要先设置必要的环境变量。

**使用 PowerShell**:

```PowerShell

# 设置环境变量 (仅在当前窗口有效)
$env:GITHUB_TOKEN="ghp_YourGitHubToken"
$env:TELEGRAM_BOT_TOKEN="YourTelegramBotToken"
$env:TELEGRAM_CHAT_ID="YourTelegramChatID"

# 运行程序
.\cfst-client-windows-amd64.exe
```
**使用 Command Prompt (CMD)**:

```DOS

# 设置环境变量 (仅在当前窗口有效)
set GITHUB_TOKEN=ghp_YourGitHubToken
set TELEGRAM_BOT_TOKEN=YourTelegramBotToken
set TELEGRAM_CHAT_ID=YourTelegramChatID

# 运行程序
cfst-client-windows-amd64.exe
```
程序启动后会立即执行一次测试，然后根据 config.yml 中定义的 cron 表达式定时执行。

## ⚙️ 配置说明

所有配置均在挂载到容器 `/app/config` 目录下的 `config.yml` 文件中完成。

| 字段 | 描述 |
| --- | --- |
| `cron` | Cron 表达式，用于定时执行测速任务。 |
| `device_name` | 当前测试端设备的唯一名称，会用于 Gist 文件名。 |
| `line_operator` | 当前设备所属的线路运营商 (如 `ct`, `cu`, `cm`)，会用于 Gist 文件名。 |
| `test_ipv6` | 是否启用 IPv6 测试 (`true` / `false`)。 |
| `proxy_prefix` | 全局 GitHub 前置代理前缀，可使用环境变量。 |
| **`gist`** | |
| `token` | GitHub Gist 的访问 Token，建议使用 `${GITHUB_TOKEN}` 从环境变量读取。 |
| `gist_id` | 要更新的 Gist ID。 |
| **`test_options`** | |
| `min_results` | 触发即时重试的结果数量下限。 |
| `max_retries` | 即时重试的最大次数。 |
| `retry_delay` | 即时重试的间隔时间（秒）。 |
| `delayed_retry` | 当即时重试全部失败后，启用此机制。 |
| `gist_upload_limit` | 上传到 Gist 的最大 IP 数量。 |
| **`cf` / `cf6`** | |
| `binary` | `CloudflareSpeedTest` 可执行文件的路径。|
| `args` | 传递给 `CloudflareSpeedTest` 的命令行参数。**注意！** 测试用的IP列表文件固定为`config/ip.txt`和`config/ipv6.txt`，无需填写。|
| `output_file` | `CloudflareSpeedTest` 输出的 CSV 文件名，**无需修改**。 |

## 📦 Gist 输出格式

程序会向指定的 Gist ID 推送文件，每次推送会覆盖同名文件。

  * **文件名格式**: `results-运营商-设备名-v4.json` 或 `results6-运营商-设备名-v6.json`。
  * **文件内容格式**:
    ```json
    {
      "timestamp": "2025-08-26T00:56:12+08:00",
      "results": [
        {
          "ip": "104.16.41.174",
          "latency_ms": 153,
          "loss_pct": 0,
          "dl_mbps": 17.58,
          "region": "SEA"
        }
      ]
    }
    ```