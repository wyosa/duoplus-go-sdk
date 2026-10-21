# DuoPlus Go SDK

[English](README.md) | 简体中文

基于 [DuoPlus OpenAPI](https://help.duoplus.net/docs/api-reference) 的非官方 Golang SDK。

适合这几类场景：

- 在 Go 服务中直接调用 DuoPlus OpenAPI
- 编写云手机运维脚本、批量任务和内部工具
- 将代理、云盘、自动化能力接入你自己的业务系统

## 功能概览

SDK 已覆盖的模块：

- 云手机管理
- 云号码
- 云手机分组
- 代理管理
- 包月开机管理
- 应用管理
- 云盘管理
- 自动化
- 团队

### 不在范围内

API 文档中所有 `/api/v1` 接口均已封装。以下内容不在范围内：

- **AI 控制云手机 API**：使用独立网关（`agent-gateway.duoplus.net`）、Bearer 鉴权和不同的响应格式。
- **高级命令**（`DuoPlusDumpUI`、`hideAccb`）：它们是命令字符串而不是接口，可以直接通过 `client.CloudPhones.Command(...)` 执行。

## 项目结构

```text
github.com/wyosa/duoplus-go-sdk/
├── app/                    # 应用管理模块
├── automation/             # 自动化模块
├── clouddisk/              # 云盘管理模块
├── cloudnumber/            # 云号码模块
├── cloudphone/             # 云手机管理模块
├── common/                 # 通用请求/响应类型
├── group/                  # 云手机分组模块
├── internal/clientcore/    # HTTP 核心、鉴权、限流、错误处理
├── proxy/                  # 代理管理模块
├── subscriptionstartup/    # 包月开机模块
├── team/                   # 团队模块：订单、余额、资源
├── examples/               # 可运行示例
├── client.go               # 根客户端入口
├── LICENSE
├── README.md               # English
└── README.zh-CN.md
```

## 安装

要求：

- Go 1.22+

```bash
go get github.com/wyosa/duoplus-go-sdk
```

建议在业务项目中显式安装版本：

```bash
go get github.com/wyosa/duoplus-go-sdk@latest
```

如果你要在生产环境长期使用，建议固定 tag，而不是始终跟随最新提交。

## API 约定

SDK 按 DuoPlus 当前公开文档的通用约定实现：

- 所有公开接口统一使用 `POST`
- 请求体为 `application/json`
- 认证头为 `DuoPlus-API-Key`
- 语言头为 `Lang`（`en`、`zh`、`zh-TW`、`ru`）
- 默认请求域名为 `https://openapi.duoplus.net`
- 中国大陆环境可切换为 `https://openapi.duoplus.cn`（`duoplus.DefaultCNBaseURL`）
- 文档规定**每个接口** QPS 为 1，SDK 默认对同一接口的两次调用至少间隔 1 秒

响应通常遵循这样的业务结构：

```json
{
	"code": 200,
	"data": {},
	"message": "Success"
}
```

其中：

- `code = 200` 表示业务成功
- 其他 code 会返回 `*duoplus.APIError`

## 安全

- **从不跟随重定向。** `net/http` 会把 `DuoPlus-API-Key` 头转发给任意重定向目标（包括其他域名）。SDK 会复制 HTTP 客户端
  （包括通过 `WithHTTPClient` 传入的客户端）并拒绝所有重定向，不会修改你自己的 `*http.Client`。
- **强制 HTTPS。** `WithBaseURL` 只接受 `https://`，`http://` 仅允许用于本地回环地址（`localhost`、`127.0.0.1`、`::1`）测试。
  带账号密码、查询参数或片段的 URL 会被拒绝。
- **限制响应大小**为 10 MiB，`APIError.Body` 截断为 4 KiB，避免过大或敏感的原始响应进入日志。
- 请把 API Key 放在环境变量或密钥管理系统中。`.env` 已被 git 忽略，参见 `.env.example`。

## 设计说明

- 默认请求域名为 `https://openapi.duoplus.net`
- 默认请求语言为 `en`
- 按文档限制，每个接口内置 1 秒最小请求间隔
- 所有接口统一使用 `POST + JSON`
- 业务错误返回 `*duoplus.APIError`
- 通用类型集中放在 `common` 包，模块请求/响应类型放在对应子包中
- `0` 或 `""` 有业务含义的可选请求字段使用指针类型（例如 `RenewalStatus *int`）。用 `common.Ptr(0)` 显式发送 0，保持 `nil` 则不发送

## 模块导入说明

推荐按这个模式导入：

```go
import (
	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/common"
)
```

约定如下：

- 根包 `duoplus` 用来创建客户端和传入全局选项
- 子包如 `cloudphone`、`proxy`、`automation` 负责各模块的请求与响应类型
- `common` 放通用分页、消息响应和公共结构

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	duoplus "github.com/wyosa/duoplus-go-sdk"
	"github.com/wyosa/duoplus-go-sdk/cloudphone"
	"github.com/wyosa/duoplus-go-sdk/common"
)

func main() {
	apiKey := os.Getenv("DUOPLUS_API_KEY")
	if apiKey == "" {
		log.Fatal("DUOPLUS_API_KEY is required")
	}

	client, err := duoplus.NewClient(
		apiKey,
		duoplus.WithBaseURL(duoplus.DefaultBaseURL),
		duoplus.WithLanguage("zh"),
	)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.CloudPhones.List(context.Background(), cloudphone.ListRequest{
		PaginationRequest: common.PaginationRequest{Page: 1, PageSize: 10},
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range resp.List {
		fmt.Println(item.ID, item.Name, item.Status)
	}
}
```

直接运行现成示例：

```bash
export DUOPLUS_API_KEY=your-api-key
go run ./examples/basic
```

## 如何使用

### 1. 初始化客户端

```go
client, err := duoplus.NewClient(
	apiKey,
	duoplus.WithBaseURL(duoplus.DefaultBaseURL),
	duoplus.WithLanguage("zh"),
)
```

可选项：

- `duoplus.WithBaseURL(...)`
- `duoplus.WithLanguage(...)`
- `duoplus.WithHTTPClient(...)`
- `duoplus.WithMinInterval(...)`

中国大陆环境初始化示例：

```go
client, err := duoplus.NewClient(
	apiKey,
	duoplus.WithBaseURL(duoplus.DefaultCNBaseURL),
	duoplus.WithLanguage("zh"),
)
```

### 2. 从模块入口访问功能

根客户端上挂载了按模块划分的 service：

- `client.CloudPhones`
- `client.CloudNumbers`
- `client.Groups`
- `client.Proxies`
- `client.SubscriptionStartups`
- `client.Apps`
- `client.CloudDisk`
- `client.Automation`
- `client.Team`

### 3. 从对应 package 引入请求/响应类型

例如云手机列表使用：

- `cloudphone.ListRequest`
- `cloudphone.ListResponse`

分页参数和通用返回类型则来自：

- `common.PaginationRequest`
- `common.Pagination`
- `common.MessageResponse`

### 4. 区分只读接口和副作用接口

下面这些调用会产生真实业务影响，接入业务时应单独审计：

- 开机、关机、重启
- 推送文件到云机
- 应用安装、卸载、启动、停止
- 自动化任务创建、修改、重新执行
- 购买、续费

因此仓库中的业务脚本示例默认都是 dry-run，只有设置 `DUOPLUS_EXECUTE=1` 才会执行真实变更。

## 常见工作流

### 工作流 1：列出云手机

1. 初始化客户端
2. 调用 `client.CloudPhones.List(...)`
3. 遍历 `resp.List`

### 工作流 2：开机并等待完成

1. 调用 `client.CloudPhones.PowerOn(...)`
2. 轮询 `client.CloudPhones.Status(...)`
3. 等待目标状态变为开机完成

参考示例：`examples/power-on-wait`

### 工作流 3：执行健康检查型 ADB 命令

1. 确保目标云机已开机
2. 调用 `client.CloudPhones.Command(...)`
3. 对返回内容做关键词校验

参考示例：`examples/power-on-adb-check`

### 工作流 4：推送云盘文件到云机

1. 调用 `client.CloudDisk.List(...)` 找到文件
2. 确认云机可用
3. 调用 `client.CloudDisk.PushFiles(...)`

参考示例：`examples/real-workflow`

## 功能列表

### 云手机管理

- 列表查询
- 开机、关机、重启
- 状态查询
- 详情查询
- 参数批量修改（包括 GPS、使用云号码的 SIM、基站）
- 代理初始化（`InitProxy`）、开启/关闭代理（`SetProxyEnabled`）
- 时区与语言列表（`Timezones`、`Languages`）
- 一键新机（`ResetDevice`）
- 手机型号列表
- Root 权限设置
- ADB 命令执行
- ADB 开关与 IP 白名单（`SetADBIPWhitelist` 覆盖，`AppendADBIPWhitelist` 追加）
- 分享与分享密码管理
- 标签、连接用户、资源列表、分辨率
- 直播（`Live`）、扫码（`ScanCode`）
- 购买、续费与自动续费（`SetAutoRenewal`）
- 转换安卓版本（`ConvertVersion`）

对应 package：`cloudphone`

### 云号码

- 云号码列表
- 云号码短信查询
- 写入短信到云机
- 购买套餐详情、套餐时长（`PurchasePackages`、`PackageDurations`）
- 购买（`Purchase`）
- 续费套餐详情与续费（`RenewalPackages`、`Renew`）
- 自动续费（`SetAutoRenewal`）

对应 package：`cloudnumber`

### 云手机分组

- 分组列表
- 添加到分组
- 移动到分组
- 创建分组
- 编辑分组
- 删除分组

对应 package：`group`

### 代理管理

- 代理列表
- 批量添加
- 批量删除
- 批量刷新 URL
- 修改代理
- 检测代理（`Check`；检测失败时调用本身仍然成功，请检查 `Success` 和 `Tips`）
- 自动续费（`SetAutoRenewal`）

对应 package：`proxy`

### 包月开机管理

- 列表
- 购买
- 续费
- 自动续费（`SetAutoRenewal`）

对应 package：`subscriptionstartup`

### 应用管理

- 平台应用列表
- 团队应用列表
- 批量安装
- 已安装应用列表
- 批量卸载
- 批量启动
- 批量关闭

对应 package：`app`

### 云盘管理

- 文件列表
- 推送文件到云机
- 上传本地文件（`UploadFile`、`Upload`，或单独调用 `SignedURL`）
- 删除文件（`Delete`）

对应 package：`clouddisk`

### 自动化

- 自定义模板列表与详情（`UserTemplateInfo`）
- 官方模板列表与详情（`OfficialTemplateInfo`）
- 创建定时任务
- 创建循环任务
- 循环任务列表
- 修改循环任务
- 暂停/开始执行循环任务
- 删除循环任务
- 定时任务列表
- 查看任务日志
- 修改任务发布时间
- 取消/重新执行任务

对应 package：`automation`

### 团队

- 订单列表（`Orders`）
- 余额查询（`Balance`）
- 团队资源（`Resources`）

对应 package：`team`

## 使用案例

### 案例 1：查询云手机状态

```go
status, err := client.CloudPhones.Status(ctx, []string{"image-id-1", "image-id-2"})
if err != nil {
	return err
}

for _, item := range status.List {
	fmt.Println(item.ID, item.Status)
}
```

### 案例 2：批量添加代理

```go
resp, err := client.Proxies.Add(ctx, proxy.AddRequest{
	ProxyList: []proxy.Input{
		{
			Protocol: "socks5",
			Host:     "127.0.0.1",
			Port:     3000,
			User:     "user",
			Password: "pass",
			Name:     "proxy-1",
		},
	},
	IPScanChannel: "ip2location",
})
if err != nil {
	return err
}

fmt.Println(resp.Success)
```

### 案例 3：查询云盘文件并推送到云机

```go
files, err := client.CloudDisk.List(ctx, clouddisk.ListRequest{
	Keyword:           "apk",
	PaginationRequest: common.PaginationRequest{Page: 1, PageSize: 20},
})
if err != nil {
	return err
}

if len(files.List) > 0 {
	_, err = client.CloudDisk.PushFiles(ctx, clouddisk.PushFilesRequest{
		IDs:      []string{files.List[0].ID},
		ImageIDs: []string{"image-id"},
		DestDir:  "/sdcard/Download",
	})
	if err != nil {
		return err
	}
}
```

### 案例 4：创建自动化定时任务

```go
_, err := client.Automation.CreateTask(ctx, automation.CreateTaskRequest{
	TemplateID:   "template-id",
	TemplateType: 2,
	Name:         "task-name",
	Remark:       "remark",
	Images: []automation.TaskImage{
		{
			ImageID: "image-id",
			IssueAt: "2026-04-01 10:00",
			Config: map[string]automation.ConfigValue{
				"text": {
					Key:      "text",
					Value:    "hello\nworld",
					Type:     "textarea",
					Required: true,
				},
			},
		},
	},
})
if err != nil {
	return err
}
```

### 案例 5：购买云手机（不自动续费）

```go
order, err := client.CloudPhones.Purchase(ctx, cloudphone.PurchaseRequest{
	OS:            "12A",
	Quantity:      1,
	Duration:      "30",
	RenewalStatus: common.Ptr(0), // 服务端默认为 1（开启自动续费）
})
if err != nil {
	return err
}

fmt.Println(order.OrderID, order.ProductIDs)
```

### 案例 6：上传 APK 到云盘

```go
file, err := client.CloudDisk.UploadFile(ctx, "./app.apk", clouddisk.SignedURLRequest{
	IsApp: 1,
	Pkg:   "com.example.app",
})
if err != nil {
	return err
}

fmt.Println(file.Name)
```

文件直接上传到预签名存储 URL，不携带 API Key。HTTP 客户端的超时不作用于上传，大文件请用 `ctx` 控制超时。

## 示例目录

- `examples/basic`: 基础云手机列表示例
- `examples/automation`: 自动化模板列表示例
- `examples/group-disk`: 云手机分组与云盘示例
- `examples/power-on-wait`: 开机后轮询状态直到完成的真实业务脚本示例
- `examples/power-on-adb-check`: 开机后执行 ADB 命令并校验输出的真实业务脚本示例
- `examples/real-workflow`: 基于真实资源自动选择云手机和云盘文件的完整业务流程示例

运行方式：

```bash
export DUOPLUS_API_KEY=your-api-key
go run ./examples/basic
go run ./examples/automation
go run ./examples/group-disk
```

### 示例环境变量

| 变量名 | 用途 | 适用示例 |
| --- | --- | --- |
| `DUOPLUS_API_KEY` | DuoPlus API Key | 全部示例 |
| `DUOPLUS_EXECUTE` | 设置为 `1` 时执行真实变更 | `power-on-wait` `power-on-adb-check` `real-workflow` |
| `DUOPLUS_TARGET_IMAGE_ID` | 指定目标云手机 ID | `power-on-wait` `power-on-adb-check` |
| `DUOPLUS_ADB_COMMAND` | 自定义 ADB shell 命令 | `power-on-adb-check` |
| `DUOPLUS_EXPECT_SUBSTRING` | 校验 ADB 输出包含指定文本 | `power-on-adb-check` |
| `DUOPLUS_DEST_DIR` | 自定义推送目录 | `real-workflow` |

### 真实业务脚本示例

#### 示例 1：开机后轮询直到完成

默认只做 dry-run，不会真的开机：

```bash
export DUOPLUS_API_KEY=your-api-key
go run ./examples/power-on-wait
```

执行真实开机并轮询状态：

```bash
export DUOPLUS_API_KEY=your-api-key
export DUOPLUS_EXECUTE=1
go run ./examples/power-on-wait
```

可选地指定目标云手机：

```bash
export DUOPLUS_TARGET_IMAGE_ID=your-image-id
```

#### 示例 2：开机后执行 ADB 命令并校验结果

默认也是 dry-run：

```bash
export DUOPLUS_API_KEY=your-api-key
go run ./examples/power-on-adb-check
```

执行真实流程：

```bash
export DUOPLUS_API_KEY=your-api-key
export DUOPLUS_EXECUTE=1
go run ./examples/power-on-adb-check
```

可选环境变量：

```bash
export DUOPLUS_TARGET_IMAGE_ID=your-image-id
export DUOPLUS_ADB_COMMAND='getprop ro.product.model'
export DUOPLUS_EXPECT_SUBSTRING='Pixel'
```

这个脚本会：

- 自动选择一台云手机，优先选择关机中的机器
- 如有需要先开机，并轮询到开机完成
- 执行一条 ADB shell 命令
- 对返回内容做简单校验，适合接业务前置健康检查

#### 示例 3：更贴近业务的完整流程

这个脚本会：

- 自动选择一台云手机
- 自动选择云盘中的一个文件
- 如果云手机处于关机状态，则先开机并轮询到完成
- 再把文件推送到云机的目标目录

默认也是 dry-run：

```bash
export DUOPLUS_API_KEY=your-api-key
go run ./examples/real-workflow
```

执行真实流程：

```bash
export DUOPLUS_API_KEY=your-api-key
export DUOPLUS_EXECUTE=1
go run ./examples/real-workflow
```

可选地自定义推送目录：

```bash
export DUOPLUS_DEST_DIR=/sdcard/Download
```

说明：

- 真实执行可能触发云手机开机计费
- 脚本默认 dry-run，就是为了避免误操作
- 脚本会优先选取关机的云手机和找到的第一个云盘文件

## 错误处理

当接口返回业务错误时，会返回 `*duoplus.APIError`。

```go
resp, err := client.CloudPhones.List(ctx, cloudphone.ListRequest{})
if err != nil {
	var apiErr *duoplus.APIError
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.Code, apiErr.Message)
	}
	return err
}

_ = resp
```

常见处理建议：

- 先用 `errors.As(err, &apiErr)` 判断是否为业务错误
- 查询型接口可以做有限重试，但要尊重 1 秒限流约束
- 对开机、推送、购买类接口，建议记录目标 ID 和返回结果，便于排查和审计
- 批量操作即使调用成功，也可能在 `Fail`（以及 `FailReason`）中返回单项失败

## 文档依据

SDK 以**英文** API 文档（https://help.duoplus.net/docs/api-reference）为准。与中文文档不一致时，以英文为准：

| 接口 | 英文文档（采用） | 中文文档 |
| --- | --- | --- |
| 详情 `/cloudPhone/info` | `image_ids`（string） | `image_id` |
| 批量设置 Root `/cloudPhone/batchRoot` | `pkgs` | `packages` |
| 写入短信 `/cloudNumber/imageWriteSms` | `image_id` 为数组 | `image_id` 为字符串 |

唯一的例外是云号码自动续费：英文页面误写为代理的路径 `/api/v1/proxy/autoRenewal`，SDK 使用中文文档中的 `/api/v1/cloudNumber/autoRenewal`。

文档自身的不一致：

- 修改循环任务（`/automation/savePlan`）：示例中 `weeks`/`days` 为数字，而创建循环任务中为字符串。SDK 发送字符串。
- 购买/续费接口的 `duration` 标注为必填，但服务端默认 30 天，因此为空时 SDK 不发送该字段。
- 参数表与示例的类型不一致时（例如模板 `id` 标注为 `int`，示例却是 `"mgRg1"`），SDK 采用示例及其他接口使用的类型。

## 发布建议

如果你准备长期对外提供这个 SDK，建议保持下面的发布习惯：

- 使用语义化版本号，例如 `v0.1.0`、`v0.2.0`
- 发生 breaking change 时及时升级主版本或至少在 README 标明
- 新增 API 时同步更新 README 的“功能列表”
- 保持 `examples/` 中示例可运行，避免文档和实现漂移

## 说明

- 当前 SDK 是依据 DuoPlus 公开文档封装的非官方实现
- 若 DuoPlus 后续调整字段、路径或业务码含义，SDK 也需要同步更新
- 在生产环境接入前，建议先用测试资源验证关键工作流

## 许可证

[MIT](LICENSE)
