# 文件地址解析处理器

> 多协议文件地址处理, 目前完成功能进度
> 1. 多协协议文件地址内文件拷贝至本地
> - [x] `file://`协议格式文件拷贝至本地
> - [x] `http(s)://`Get无参协议地址文件拷贝到本地
> - [ ] `http(s)://`自定义请求方式、参数等内容的文件地址拷贝到本地( 暂未实现 )
> - [x] `data:application/pdf;base64,`Base64格式的MIME类型的地址文件拷贝到本地
> - [ ] `(s)ftp://`协议文件拷贝到本地
>
> 2. 本地文件保存至多协议地址
> - [ ] `file://`保存至本地
> - [ ] `http(s)://`保存至http服务
> - [ ] `(s)ftp://`保存至ftp服务

# 安装依赖库

```go
go get github.com/byzk-worker/file-addr-handler
```

# 示例

```go
package main

import (
	"fmt"
	fileaddrhandler "github.com/byzk-worker/file-addr-handler"
)

func main() {
	parser := fileaddrhandler.New(fileaddrhandler.FileTypePDF)

	ft, err := parser.CopyToPath("file://C:\\a.pdf", "target.pdf")
	if err != nil {
		panic(err)
	}

	fmt.Println(ft == fileaddrhandler.FileTypePDF)
}

```

[更多示例](parser_test.go)