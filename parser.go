package fileaddrhandler

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
)

const isWindows = runtime.GOOS == "windows"

var httpsSupportClient = &http.Client{Transport: &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}}

var (
	// mimeTypeJudgeReg MIME类型判断正则
	mimeTypeJudgeReg = regexp.MustCompile("^data:([a-z]+)/([a-z]+);base64,([\\da-zA-Z+=/]+)$")
)

type CodeFlag byte

const (
	CodeFlagB64 CodeFlag = iota
	CodeFlagHex
)

// Parser 解析器
type Parser struct {
	// SupportFileTypes 支持的类型列表
	supportFileTypeMap map[FileType]struct{}
}

// New 初始化解析器对象
func New(supportTypes ...FileType) *Parser {
	supportMap := make(map[FileType]struct{})
	for i := range supportTypes {
		supportMap[supportTypes[i]] = struct{}{}
	}
	return &Parser{
		supportFileTypeMap: supportMap,
	}
}

// AddSupportTypes 添加支持的类型
func (p *Parser) AddSupportTypes(supportTypes ...FileType) {
	for i := range supportTypes {
		ft := supportTypes[i]
		if _, ok := p.supportFileTypeMap[ft]; !ok {
			p.supportFileTypeMap[ft] = struct{}{}
		}
	}
}

// DelSupportTypes 删除支持的类型
func (p *Parser) DelSupportTypes(fts ...FileType) {
	for i := range fts {
		delete(p.supportFileTypeMap, fts[i])
	}
}

// writeSupportFile 向目标写入支持的文件
func (p *Parser) writeSupportFile(src io.Reader, target io.Writer) (FileType, error) {
	buf := make([]byte, 10)
	n, err := src.Read(buf)
	if err != nil {
		return "", ErrCodeProtoFileRead.ErrorWithRawErrf(err, "协议文件内容读取失败: %s", err.Error())
	}

	buf = buf[:n]
	rawHeadHex := byteToHex(buf)

	ft := FileEmpty
	for k := range p.supportFileTypeMap {
		if k.Is(rawHeadHex) {
			ft = k
			goto StartCopy
		}
	}
	return "", ErrCodeUnsupportedFileType.Error("不支持当前原始的文件类型")
StartCopy:
	if _, err = target.Write(buf); err != nil {
		return "", ErrCodeTargetFileWrite.ErrorWithRawErrf(err, "向目标文件写出内容失败: %s", err)
	}

	if _, err = io.Copy(target, src); err != nil {
		return "", ErrCodeTargetFileWrite.ErrorWithRawErrf(err, "向目标文件写出内容失败: %s.", err)
	}

	return ft, nil
}

// mimeFileWrite mime类型文件写出
func (p *Parser) mimeFileWrite(mimeStr string, w io.Writer) (FileType, error) {
	index := strings.Index(mimeStr, "base64,")
	if index == -1 {
		return "", ErrCodeUnsupportedProtocols.Errorf("非法的协议地址[%s]", mimeStr)
	}
	b64Data := mimeStr[index+7:]
	dataBytes, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return "", ErrCodeProtoFileOpen.ErrorWithRawErrf(err, "协议文件格式解析失败: %s", err.Error())
	}
	return p.writeSupportFile(bytes.NewReader(dataBytes), w)
}

// httpProtoWrite http协议文件写出
// 支持http(s)协议类型, 暂时只支持Get 无参请求, 示例类型如下
// http://127.0.0.1/1.pdf
// https://127.0.0.1/1.pdf
func (p *Parser) httpProtoWrite(uri string, w io.Writer) (FileType, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return "", ErrCodeHttpRequestCreate.ErrorWithRawErrf(err, "创建http请求对象失败: %s", err.Error())
	}

	resp, err := httpsSupportClient.Do(req)
	if err != nil {
		return "", ErrCodeHttpRequest.ErrorWithRawErrf(err, "访问http请求资源失败: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == 404 {
			return "", ErrCodeProtoFileNoExist.Errorf("http资源[%s]不存在", uri)
		}
		return "", ErrCodeResStatusCode.Errorf("非法的http响应状态码: %d", resp.StatusCode)
	}

	return p.writeSupportFile(resp.Body, w)
}

// fileProtoWrite file协议文件写出
// 文件协议如下类似, 以file://开头
// Windows:  file://C:\Windows\System32\cmd.exe
// Linux:    file:///usr/bin/ls
func (p *Parser) fileProtoWrite(uri string, w io.Writer) (FileType, error) {
	localPath := uri[7:]
	if stat, err := os.Stat(localPath); err != nil || stat.IsDir() {
		return "", ErrCodeProtoFileNoExist.ErrorWithRawErrf(err, "本地文件[%s]不存在", localPath)
	}

	f, err := os.OpenFile(localPath, os.O_RDONLY, 0655)
	if err != nil {
		return "", ErrCodeProtoFileOpen.ErrorWithRawErrf(err, "打开本地文件[%s]失败: %s", localPath, err.Error())
	}
	defer f.Close()

	return p.writeSupportFile(f, w)
}

// Copy 拷贝文件流
func (p *Parser) Copy(reader io.Reader, writer io.Writer) (FileType, error) {
	if reader == nil {
		return "", ErrCodeEmptyStream.Error("读取流不能为空")
	}

	if writer == nil {
		return "", ErrCodeEmptyStream.Error("写出流不能为空")
	}

	return p.writeSupportFile(reader, writer)
}

// CopyByURI 拷贝文件通过路径
func (p *Parser) CopyByURI(srcFilePath, targetFilePath string) (FileType, error) {
	return p.CopyWithOption(WithEmptySourceOption().SetUri(srcFilePath), WithEmptyTargetOption().SetUri(targetFilePath))
}

// CopyWithOption 拷贝文件通过选项
func (p *Parser) CopyWithOption(src *sourceOption, target *targetOption) (FileType, error) {
	var (
		t   FileType
		err error
	)

	if e := src.parse(func(r io.Reader) error {
		t, err = target.writeByReader(r, p)
		return nil
	}); e != nil {
		return "", e
	}
	return t, err
}

func (p *Parser) CopyToBytes(srcFile string) (FileType, BytesResult, error) {
	return p.CopyToBytesWithOption(WithEmptySourceOption().SetUri(srcFile))
}

func (p *Parser) CopyToBytesWithOption(srcFile *sourceOption) (FileType, BytesResult, error) {
	var t FileType
	buf := &bytes.Buffer{}
	if err := srcFile.parse(func(r io.Reader) error {
		fileType, err := p.Copy(r, buf)
		if err != nil {
			return ErrCodeTargetFileWrite.ErrorWithRawErrf(err, "拷贝文件数据失败: %s", err.Error())
		}
		t = fileType
		return nil
	}); err != nil {
		return "", nil, err
	}
	return t, buf.Bytes(), nil
}

type BytesResult []byte

func (b BytesResult) Hex() string {
	return hex.EncodeToString(b)
}

func (b BytesResult) Base64() string {
	return base64.StdEncoding.EncodeToString(b)
}
