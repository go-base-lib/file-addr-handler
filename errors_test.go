package fileparser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestError_Equal(t *testing.T) {

	a := assert.New(t)

	var err error = ErrCodeMkdir.Error("测试错误")

	a.True(ErrCodeMkdir.Equal(err))
	a.False(ErrCodeMkFile.Equal(err))

	rawErr, ok := ErrParse(err)
	a.True(ok)

	a.True(rawErr.Equal(ErrCodeMkdir))
	a.False(rawErr.Equal(ErrCodeMkFile))

}
