package logging

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"testing"
)

func TestWrite(t *testing.T) {
	t.Run("fail on remove old file", func(t *testing.T) {
		mockRemove := newMockOsRemove(t)
		mockRemove.EXPECT().Execute("mockpath").Return(fmt.Errorf("testerror"))
		mockCreate := newMockOsCreate(t)
		w := NewWriter("mockpath", func(name string) error {
			return mockRemove.Execute(name)
		}, func(name string) (File, error) {
			return mockCreate.Execute(name)
		}, nil)
		rc := NewMockReadCloser(t)
		rc.EXPECT().Close().Return(nil)

		err := w.Write(rc)
		assert.Error(t, err)
		assert.Equal(t, "failed to clear old log file: testerror", err.Error())
	})
	t.Run("fail on create new file", func(t *testing.T) {
		mockRemove := newMockOsRemove(t)
		mockRemove.EXPECT().Execute("mockpath").Return(nil)
		mockCreate := newMockOsCreate(t)
		mockCreate.EXPECT().Execute("mockpath").Return(nil, fmt.Errorf("testerror"))
		w := NewWriter("mockpath", func(name string) error {
			return mockRemove.Execute(name)
		}, func(name string) (File, error) {
			return mockCreate.Execute(name)
		}, nil)
		rc := NewMockReadCloser(t)
		rc.EXPECT().Close().Return(nil)

		err := w.Write(rc)
		assert.Error(t, err)
		assert.Equal(t, "failed to create log file: testerror", err.Error())
	})
	t.Run("fail on copy", func(t *testing.T) {
		mockRemove := newMockOsRemove(t)
		mockRemove.EXPECT().Execute("mockpath").Return(nil)
		mockFile := NewMockFile(t)
		mockFile.EXPECT().Close().Return(nil)
		mockCreate := newMockOsCreate(t)
		mockCreate.EXPECT().Execute("mockpath").Return(mockFile, nil)
		mockCopy := newMockIoCopy(t)
		mockCopy.EXPECT().Execute(mockFile, mock.Anything).Return(0, fmt.Errorf("testerror"))
		w := NewWriter("mockpath", func(name string) error {
			return mockRemove.Execute(name)
		}, func(name string) (File, error) {
			return mockCreate.Execute(name)
		}, func(dst io.Writer, src io.Reader) (written int64, err error) {
			return mockCopy.Execute(dst, src)
		})
		rc := NewMockReadCloser(t)
		rc.EXPECT().Close().Return(nil)

		err := w.Write(rc)
		assert.Error(t, err)
		assert.Equal(t, "failed to copy log to path mockpath: testerror", err.Error())
	})
	t.Run("success", func(t *testing.T) {
		mockRemove := newMockOsRemove(t)
		mockRemove.EXPECT().Execute("mockpath").Return(nil)
		mockFile := NewMockFile(t)
		mockFile.EXPECT().Close().Return(nil)
		mockCreate := newMockOsCreate(t)
		mockCreate.EXPECT().Execute("mockpath").Return(mockFile, nil)
		rc := NewMockReadCloser(t)
		rc.EXPECT().Close().Return(nil)
		mockCopy := newMockIoCopy(t)
		mockCopy.EXPECT().Execute(mockFile, rc).Return(0, nil)
		w := NewWriter("mockpath", func(name string) error {
			return mockRemove.Execute(name)
		}, func(name string) (File, error) {
			return mockCreate.Execute(name)
		}, func(dst io.Writer, src io.Reader) (written int64, err error) {
			return mockCopy.Execute(dst, src)
		})

		err := w.Write(rc)
		assert.NoError(t, err)
	})
}
