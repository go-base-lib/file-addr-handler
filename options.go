package fileaddrhandler

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// SourceHttpOption 原始文件的http选项
type SourceHttpOption struct {
	// Method 请求方法
	Method string
	// Headers 请求头
	Headers http.Header
	// Form 表单数据
	Form url.Values
	// ReqBody 请求体
	ReqBody string
}

// TargetHttpOption 目标文件的http选项
type TargetHttpOption struct {
	Method    string
	FieldName string
	Filename  string
	Headers   http.Header
	Form      map[string]string
}

// parseOptionData 解析选项数据
func parseOptionData[T SourceHttpOption | TargetHttpOption | []byte](d any) (res *T, err error) {
	if d == nil {
		return nil, nil
	}
	switch t := d.(type) {
	case T:
		return &t, nil
	case *T:
		return t, nil
	case string:
		if err = json.Unmarshal([]byte(t), &res); err != nil {
			return nil, ErrOption.ErrorWithRawErr(err, "解析选项内容失败： %s"+err.Error())
		}
	case []byte:
		if err = json.Unmarshal(t, &res); err != nil {
			return nil, ErrOption.ErrorWithRawErr(err, "解析选项内容失败： %s"+err.Error())
		}
	default:
		return nil, ErrOption.Error("未被认可的选项内容")
	}
	return
}

// commonOption 通用选项
type commonOption[T sourceOption | targetOption] struct {
	// uri 协议地址
	uri string
	// data 数据
	data any
	// raw 原始对象
	raw *T
}

// SetUri 设置uri
func (c *commonOption[T]) SetUri(uri string) *T {
	c.uri = uri
	return c.raw
}

// GetUri 获取uri
func (c *commonOption[T]) GetUri() string {
	return c.uri
}

// WithEmptySourceOption 空数据的option
func WithEmptySourceOption() *sourceOption {
	return WithAnySourceOption(nil)
}

// WithHttpSourceOption http数据的原始请求数据
func WithHttpSourceOption(option *SourceHttpOption) *sourceOption {
	return WithAnySourceOption(option)
}

// WithAnySourceOption 带有任意数据的option
func WithAnySourceOption(data any) *sourceOption {
	option := &sourceOption{
		commonOption: &commonOption[sourceOption]{
			data: data,
		},
	}
	option.raw = option
	return option
}

type readerCallback func(r io.Reader) error

// sourceOption 源文件选项
type sourceOption struct {
	*commonOption[sourceOption]
	// 文件读取流
	r  io.Reader
	fn readerCallback
}

// SetReader 设置原文读取流
func (s *sourceOption) SetReader(r io.Reader) *sourceOption {
	s.r = r
	return s
}

func (s *sourceOption) parseMimeReader(mimeStr string) error {
	i := strings.Index(mimeStr, ";")
	if i < 0 {
		return ErrCodeUnsupportedProtocols.Error("不支持的MIME Type类型")
	}

	mimeStr = mimeStr[i+1:]

	i = strings.Index(mimeStr, ",")
	if i < 0 {
		return ErrCodeUnsupportedProtocols.Error("不支持的MIME type类型")
	}

	t := strings.ToLower(mimeStr[:i])
	mimeStr = mimeStr[i+1:]

	var (
		res []byte
		err error
	)
	switch t {
	case "base64":
		res, err = base64.StdEncoding.DecodeString(mimeStr)
	case "hex":
		res, err = hex.DecodeString(mimeStr)
	default:
		return ErrCodeUnsupportedProtocols.Error("不支持的MIME编码方式： " + t)
	}

	if err != nil {
		return ErrCodeUnsupportedProtocols.Errorf("解析%s格式的MIME TYPE类型内容失败: %s", t, err.Error())
	}

	return s.fn(bytes.NewReader(res))
}

// parseHttpReader 解析HTTP头信息
func (s *sourceOption) parseHttpReader(uri string) error {
	var (
		option *SourceHttpOption
		err    error
	)
	if option, err = parseOptionData[SourceHttpOption](s.data); err != nil {
		return err
	}

	if option == nil {
		option = &SourceHttpOption{}
	}
	if option.Method == "" {
		option.Method = "GET"
	} else {
		option.Method = strings.ToUpper(option.Method)
	}

	var reqBody io.Reader
	if option.ReqBody != "" {
		reqBody = strings.NewReader(option.ReqBody)
	}

	req, err := http.NewRequest(option.Method, uri, reqBody)
	if err != nil {
		return ErrCodeHttpRequestCreate.ErrorWithRawErrf(err, "创建http请求对象失败: %s", err.Error())
	}

	if option.Form != nil {
		req.Form = option.Form
	}

	if option.Headers != nil {
		req.Header = option.Headers
	}

	resp, err := httpsSupportClient.Do(req)
	if err != nil {
		return ErrCodeHttpRequest.ErrorWithRawErrf(err, "访问http请求资源失败: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == 404 {
			return ErrCodeProtoFileNoExist.Errorf("http资源[%s]不存在", uri)
		}
		return ErrCodeResStatusCode.Errorf("非法的http响应状态码: %d", resp.StatusCode)
	}

	return s.fn(resp.Body)
}

func (s *sourceOption) parse(fn readerCallback) error {
	defer func() { s.fn = nil }()
	s.fn = fn
	if s.r != nil {
		return fn(s.r)
	}

	if s.uri == "" {
		return ErrCodeUnsupportedProtocols.Error("不支持空的地址")
	}

	if mimeTypeJudgeReg.MatchString(s.uri) {
		return s.parseMimeReader(s.uri)
	}

	uri, err := url.QueryUnescape(s.uri)
	if err != nil {
		return ErrCodeUnsupportedProtocols.Error("解析url编码失败: " + err.Error())
	}
	u, err := url.Parse(uri)
	if err != nil {
		return ErrCodeUnsupportedProtocols.ErrorWithRawErrf(err, "不支持的协议类型: %s", err.Error())
	}

	switch u.Scheme {
	case "http":
		fallthrough
	case "https":
		return s.parseHttpReader(uri)
	case "file":
		filePath := filepath.Join(u.Host, u.Path)
		if isWindows {
			filePath = strings.TrimLeft(filePath, "/")
			filePath = strings.TrimLeft(filePath, "\\")
		}

		file, err := os.OpenFile(filePath, os.O_RDONLY, 0655)
		if err != nil {
			return ErrCodeProtoFileOpen.ErrorWithRawErrf(err, "打开原始文件[%s]失败: %s", filePath, err.Error())
		}
		defer file.Close()
		return s.fn(file)
	default:
		return ErrCodeUnsupportedProtocols.Error("暂不支持该协议类型")
	}
}

// WithEmptyTargetOption 空数据的option
func WithEmptyTargetOption() *targetOption {
	return WithAnyTargetOption(nil)
}

// WithHttpTargetOption http数据的原始请求数据
func WithHttpTargetOption(option *TargetHttpOption) *targetOption {
	return WithAnyTargetOption(option)
}

// WithAnyTargetOption 带有任意数据的option
func WithAnyTargetOption(data any) *targetOption {
	option := &targetOption{
		commonOption: &commonOption[targetOption]{
			data: data,
		},
	}
	option.raw = option
	return option
}

// 目标选项
type targetOption struct {
	*commonOption[targetOption]
	// w 写出流
	w io.Writer
}

// SetWriter 设置目标写入流
func (t *targetOption) SetWriter(w io.Writer) *targetOption {
	t.w = w
	return t
}

type httpFileWriteResult struct {
	err error
	t   FileType
}

func (t *targetOption) writeToHttp(uri string, r io.Reader, p *Parser) (FileType, error) {
	var (
		option *TargetHttpOption
		err    error
	)
	if option, err = parseOptionData[TargetHttpOption](t.data); err != nil {
		return "", err
	}

	if option == nil {
		option = &TargetHttpOption{}
	}

	if option.Method == "" {
		option.Method = "POST"
	} else {
		option.Method = strings.ToUpper(option.Method)
	}

	if option.FieldName == "" {
		option.FieldName = "file"
	}

	if option.Filename == "" {
		i := strings.LastIndex(uri, "/")
		option.Filename = uri[i+1:]
	}

	pipeR, pipeW := io.Pipe()
	m := multipart.NewWriter(pipeW)
	if option.Form != nil {
		for k, v := range option.Form {
			if err := m.WriteField(k, v); err != nil {
				return "", ErrCodeTargetFileWrite.ErrorWithRawErrf(err, "写出字段[%s]失败: %s", k, err.Error())
			}
		}
	}
	ch := make(chan *httpFileWriteResult, 1)
	go func() {
		defer pipeW.Close()
		defer m.Close()
		defer func() { close(ch) }()
		f, err := m.CreateFormFile(option.FieldName, option.Filename)
		if err != nil {
			ch <- &httpFileWriteResult{err: err}
			return
		}
		fileType, err := p.Copy(r, f)
		ch <- &httpFileWriteResult{err: err, t: fileType}
	}()

	req, err := http.NewRequest(option.Method, uri, pipeR)
	if err != nil {
		return "", ErrCodeTargetFileWrite.ErrorWithRawErrf(err, "创建请求对象失败: %s", err.Error())
	}
	if option.Headers == nil {
		option.Headers = make(http.Header, 1)
	}
	req.Header = option.Headers
	req.Header.Add("Content-Type", m.FormDataContentType())

	res, err := httpsSupportClient.Do(req)
	if err != nil {
		return "", ErrCodeTargetFileWrite.ErrorWithRawErrf(err, "想目标请求发送数据失败: %s", err.Error())
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return "", ErrCodeTargetFileWrite.Errorf("服务器返回错误的状态码: %d", res.StatusCode)
	}

	result := <-ch
	return result.t, result.err
}

func (t *targetOption) writeByReader(r io.Reader, p *Parser) (FileType, error) {
	if t.w != nil {
		return p.Copy(r, t.w)
	}

	if t.uri == "" {
		return "", ErrCodeUnsupportedProtocols.Error("不支持空的地址")
	}

	if mimeTypeJudgeReg.MatchString(t.uri) {
		return "", ErrCodeUnsupportedProtocols.Error("不支持写出MIME类型数据")
	}

	uri, err := url.QueryUnescape(t.uri)
	if err != nil {
		return "", ErrCodeUnsupportedProtocols.Error("解析写出url编码失败: " + err.Error())
	}
	u, err := url.Parse(uri)
	if err != nil {
		return "", ErrCodeUnsupportedProtocols.ErrorWithRawErrf(err, "不支持的写出协议类型: %s", err.Error())
	}

	switch u.Scheme {
	case "http":
		fallthrough
	case "https":
		return t.writeToHttp(uri, r, p)
	case "file":
		fp := filepath.Join(u.Host, u.Path)
		if isWindows {
			fp = strings.TrimLeft(fp, "/")
			fp = strings.TrimLeft(fp, "\\")
		}

		_ = os.RemoveAll(fp)
		_ = os.MkdirAll(filepath.Dir(fp), 0755)
		file, err := os.OpenFile(fp, os.O_WRONLY|os.O_CREATE, 0655)
		if err != nil {
			return "", ErrCodeProtoFileOpen.ErrorWithRawErrf(err, "创建目标文件失败: %s", err.Error())
		}
		defer file.Close()
		return p.Copy(r, file)
	default:
		return "", ErrCodeUnsupportedProtocols.Error("暂不支持该写出协议类型")
	}
}
