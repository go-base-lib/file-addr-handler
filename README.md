# 文件地址解析处理器

> 多协议文件地址处理, 目前完成功能进度
> 1. 多协协议文件地址内文件拷贝至本地
> - [x] `file://`协议格式文件拷贝至本地
> - [x] `http(s)://`Get无参协议地址文件拷贝到本地
> - [x] `http(s)://`自定义请求方式、参数等内容的文件地址拷贝到本地
> - [x] `data:application/pdf;base64,`Base64格式的MIME类型的地址文件拷贝到本地
> - [ ] `(s)ftp://`协议文件拷贝到本地
>
> 2. 本地文件保存至多协议地址
> - [x] `file://`保存至本地
> - [x] `http(s)://`保存至http服务
> - [ ] `(s)ftp://`保存至ftp服务

# 安装依赖库

```go
go get github.com/byzk-worker/file-addr-handler
```

# 注意!!!
> 库版本进行升级内部API发生巨大变化, 现在Copy方法的源和目标必须均为URI格式地址

# 示例

```go
package main

import (
	"fmt"
	fileaddrhandler "github.com/byzk-worker/file-addr-handler"
)

func main() {
	parser := fileaddrhandler.New(fileaddrhandler.FileTypePDF)

	// 本地文件拷贝
	ft, err := parser.CopyByURI("file:///C:\\a.pdf", "file:///C:\target.pdf")
	if err != nil {
		panic(err)
	}

	fmt.Println(ft == fileaddrhandler.FileTypePDF)
	
	// 拷贝本地文件至http, 采用默认POST方式
	_, err = parser.CopyByURI("file:///C:\\a.pdf", "http://127.0.0.1:8090/a.pdf")
	if err != nil {
		panic(err)
	}
	
	// 拷贝本地文件至http, 自定义目标的请求方式以及文件字段和名称
	_, err = parser.CopyWithOption(fileaddrhandler.WithEmptySourceOption().SetUri("file:///C:\a.pdf"), 
		fileaddrhandler.WithHttpTargetOption(&fileaddrhandler.TargetHttpOption{
			FieldName: "pdfFile",
			FileName: "a.pdf",
			Method: "PUT",
        }).SetUri("http://127.0.0.1:7890/upload"))
	_, err = parser.CopyByURI("file:///C:\\a.pdf", "http://127.0.0.1:8090/a.pdf")
	if err != nil {
		panic(err)
	}
}

```

[更多示例](parser_test.go)

> Option可以携带的更多参数请自行参考源代码