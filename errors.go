package fileaddrhandler

import "fmt"

// ErrCode 错误代码
type ErrCode uint8

// Error 从错误代码构建Error结构
func (e ErrCode) Error(msg string) *Error {
	return e.ErrorWithRawErr(nil, msg)
}

// Errorf 从错误代码构建Error结构, 内部msg使用 format 字符串格式
func (e ErrCode) Errorf(format string, args ...any) *Error {
	return e.ErrorWithRawErrf(nil, format, args...)
}

// ErrorWithRawErr 从错误代码构建Error结构，允许传入原始异常与错误消息
func (e ErrCode) ErrorWithRawErr(err error, msg string) *Error {
	return &Error{
		Code:   e,
		Msg:    msg,
		RawErr: err,
	}
}

// ErrorWithRawErrf 从错误代码构建Error结构，允许传入原始异常与错误消息
func (e ErrCode) ErrorWithRawErrf(err error, msg string, args ...any) *Error {
	return e.ErrorWithRawErr(err, fmt.Sprintf(msg, args...))
}

// Equal 判断错误是否相等
func (e ErrCode) Equal(err error) bool {
	targetErr, ok := ErrParse(err)
	if !ok {
		return ok
	}
	return targetErr.Equal(e)
}

// ErrParse 将一个异常尝试解析为一个 Error 类型
func ErrParse(err error) (*Error, bool) {
	switch t := err.(type) {
	case *Error:
		return t, true
	default:
		return nil, false
	}
}

// Error 具体的错误内容
type Error struct {
	// Code 错误代码
	Code ErrCode
	// Msg 错误消息
	Msg string
	// RawErr 原始异常
	RawErr error
}

// Error 实现error接口
func (e *Error) Error() string {
	return e.Msg
}

// Equal 判断Error内的Code是否与预期匹配
func (e *Error) Equal(errCode ErrCode) bool {
	return errCode == e.Code
}

const (
	// ErrCodeMkdir 创建文件夹失败
	ErrCodeMkdir ErrCode = iota + 1
	// ErrCodeMkFile 创建文件失败
	ErrCodeMkFile
	// ErrCodeUnsupportedProtocols 不支持的协议类型
	ErrCodeUnsupportedProtocols
	// ErrCodeNoSupportFileTypes 没有支持的文件类型
	ErrCodeNoSupportFileTypes
	// ErrCodeProtoFileNoExist 协议文件不存在
	ErrCodeProtoFileNoExist
	// ErrCodeProtoFileOpen 协议文件打开失败
	ErrCodeProtoFileOpen
	// ErrCodeProtoFileRead 协议文件读取失败
	ErrCodeProtoFileRead
	// ErrCodeUnsupportedFileType 不支持的文件类型
	ErrCodeUnsupportedFileType
	// ErrCodeTargetFileWrite 目标文件写出失败
	ErrCodeTargetFileWrite
	// ErrCodeHttpRequestCreate 创建http请求失败
	ErrCodeHttpRequestCreate
	// ErrCodeHttpRequest http 请求失败
	ErrCodeHttpRequest
	// ErrCodeResStatusCode http响应状态码非200-299之间
	ErrCodeResStatusCode
	// ErrCodeEmptyStream 空流错误
	ErrCodeEmptyStream
	// ErrOption 错误的选项
	ErrOption
)
