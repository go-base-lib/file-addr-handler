package fileaddrhandler

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestFileTypeParse(t *testing.T) {
	a := assert.New(t)

	fileBytes, err := ioutil.ReadFile("test.pdf")
	if !a.NoError(err, "打开测试文件失败") {
		return
	}

	fType := byteToHex(fileBytes)

	if a.True(FileTypePDF.Is(fType), "文件类型不匹配") {
		return
	}
}
