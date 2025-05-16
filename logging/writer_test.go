package logging

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"os"
	"testing"
)

func TestWrite(t *testing.T) {
	t.Run("fail on create new file", func(t *testing.T) {
		mockCreate := newMockOsOpenFile(t)
		mockCreate.EXPECT().Execute("mockpath", logFilesMode, mock.Anything).Return(nil, fmt.Errorf("testerror"))
		w := NewWriter("mockpath")
		w.openFile = func(name string, flag int, perm os.FileMode) (file, error) {
			return mockCreate.Execute(name, flag, perm)
		}
		rc := NewMockReadCloser(t)
		rc.EXPECT().Close().Return(nil)

		err := w.Write(rc)
		assert.Error(t, err)
		assert.Equal(t, "failed to open log file: testerror", err.Error())
	})
	t.Run("fail on copy", func(t *testing.T) {
		mockFile := newMockFile(t)
		mockFile.EXPECT().Close().Return(nil)
		mockCopy := newMockIoCopy(t)
		mockCopy.EXPECT().Execute(mockFile, mock.Anything).Return(0, fmt.Errorf("testerror"))
		mockCreate := newMockOsOpenFile(t)
		mockCreate.EXPECT().Execute("mockpath", logFilesMode, mock.Anything).Return(mockFile, nil)
		w := NewWriter("mockpath")
		w.copy = func(dst io.Writer, src io.Reader) (written int64, err error) {
			return mockCopy.Execute(dst, src)
		}
		w.openFile = func(name string, flag int, perm os.FileMode) (file, error) {
			return mockCreate.Execute(name, flag, perm)
		}
		rc := NewMockReadCloser(t)
		rc.EXPECT().Close().Return(nil)

		err := w.Write(rc)
		assert.Error(t, err)
		assert.Equal(t, "failed to copy log to path mockpath: testerror", err.Error())
	})
	t.Run("success", func(t *testing.T) {
		mockFile := newMockFile(t)
		mockFile.EXPECT().Close().Return(nil)
		rc := NewMockReadCloser(t)
		rc.EXPECT().Close().Return(nil)
		mockCopy := newMockIoCopy(t)
		mockCopy.EXPECT().Execute(mockFile, rc).Return(0, nil)
		mockCreate := newMockOsOpenFile(t)
		mockCreate.EXPECT().Execute("mockpath", logFilesMode, mock.Anything).Return(mockFile, nil)
		w := NewWriter("mockpath")
		w.copy = func(dst io.Writer, src io.Reader) (written int64, err error) {
			return mockCopy.Execute(dst, src)
		}
		w.openFile = func(name string, flag int, perm os.FileMode) (file, error) {
			return mockCreate.Execute(name, flag, perm)
		}

		err := w.Write(rc)
		assert.NoError(t, err)
	})
}
