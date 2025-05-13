package exporter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewService(t *testing.T) {
	apiClientMock := newMockApiClient(t)

	sut := NewService("http://localhost:8080", apiClientMock)

	assert.NotNil(t, sut)
	assert.NotNil(t, sut.MaintenanceModeService)
}
