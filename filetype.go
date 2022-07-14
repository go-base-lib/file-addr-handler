package fileparser

import (
	"encoding/hex"
	"strings"
)

// FileType 文件类型
type FileType string

func (f *FileType) Is(t string) bool {
	return strings.HasPrefix(t, string(*f))
}

var (
	FileEmpty   FileType = ""
	FileTypePDF FileType = "255044462d312e"
)

func byteToHex(src []byte) string {
	if src == nil || len(src) <= 0 {
		return ""
	}

	if len(src) > 10 {
		src = src[:10]
	}

	return hex.EncodeToString(src)

}
