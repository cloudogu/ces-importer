package migration

import (
	"context"
	"errors"
	"testing"
)

func TestExportModeValidatorApiClient_Validate(t *testing.T) {
	testCtx := context.Background()

	tests := []struct {
		name          string
		mockResponse  bool
		mockError     error
		expectedError string
	}{
		{
			name:          "success",
			mockResponse:  true,
			expectedError: "",
		},
		{
			name:          "export mode inactive",
			mockResponse:  false,
			expectedError: "export mode is not active",
		},
		{
			name:          "error from api client",
			mockError:     errors.New("api error"),
			expectedError: "failed to validate export mode: api error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := newMockExportModeClient(t)
			mockClient.EXPECT().GetExportMode(testCtx).Return(tt.mockResponse, tt.mockError)

			validator := NewExportModeValidatorApiClient(mockClient)

			err := validator.Validate(testCtx)
			if tt.expectedError == "" && err != nil {
				t.Errorf("expected no error but got %v", err)
			}
			if tt.expectedError != "" && (err == nil || err.Error() != tt.expectedError) {
				t.Errorf("expected error %v but got %v", tt.expectedError, err)
			}
		})
	}
}
