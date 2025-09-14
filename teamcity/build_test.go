package teamcity

import (
	"bbox/pkg/types"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildService_WaitForBuild(t *testing.T) {
	tests := []struct {
		name             string
		expectedStatuses []string
		mockResponse     types.BuildStatusResponse // Single response that represents finished state
		expectedResult   types.BuildStatusResponse
		expectedError    string
	}{
		{
			name:           "Successful build without expected statuses",
			mockResponse:   types.BuildStatusResponse{ID: 123, Status: "SUCCESS", State: "finished"},
			expectedResult: types.BuildStatusResponse{ID: 123, Status: "SUCCESS", State: "finished"},
			expectedError:  "",
		},
		{
			name:             "Successful build with matching expected status",
			expectedStatuses: []string{"SUCCESS", "FAILURE"},
			mockResponse:     types.BuildStatusResponse{ID: 123, Status: "SUCCESS", State: "finished"},
			expectedResult:   types.BuildStatusResponse{ID: 123, Status: "SUCCESS", State: "finished"},
			expectedError:    "",
		},
		{
			name:             "Build finished with unexpected status",
			expectedStatuses: []string{"SUCCESS"},
			mockResponse:     types.BuildStatusResponse{ID: 123, Status: "FAILURE", State: "finished"},
			expectedResult:   types.BuildStatusResponse{ID: 123, Status: "FAILURE", State: "finished"},
			expectedError:    "build TestBuild finished with status 'FAILURE', but expected one of: [SUCCESS]",
		},
		{
			name:             "Build finished with one of multiple expected statuses",
			expectedStatuses: []string{"SUCCESS", "FAILURE", "UNKNOWN"},
			mockResponse:     types.BuildStatusResponse{ID: 123, Status: "FAILURE", State: "finished"},
			expectedResult:   types.BuildStatusResponse{ID: 123, Status: "FAILURE", State: "finished"},
			expectedError:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that returns finished state immediately
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			// Parse server URL
			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			// Create client
			client, err := NewTeamCityClient(serverURL, "testuser", "testpass")
			require.NoError(t, err)

			// Call WaitForBuild
			result, err := client.Build.WaitForBuild("TestBuild", 123, 30*time.Second, tt.expectedStatuses)

			// Assert results
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedResult.ID, result.ID)
			assert.Equal(t, tt.expectedResult.Status, result.Status)
			assert.Equal(t, tt.expectedResult.State, result.State)
		})
	}
}

func TestBuildService_WaitForBuild_ExpectedStatusesValidation(t *testing.T) {
	tests := []struct {
		name             string
		expectedStatuses []string
		actualStatus     string
		shouldError      bool
		errorMessage     string
	}{
		{
			name:             "Empty expected statuses - no validation",
			expectedStatuses: []string{},
			actualStatus:     "FAILURE",
			shouldError:      false,
		},
		{
			name:             "Nil expected statuses - no validation",
			expectedStatuses: nil,
			actualStatus:     "FAILURE",
			shouldError:      false,
		},
		{
			name:             "Single expected status matches",
			expectedStatuses: []string{"SUCCESS"},
			actualStatus:     "SUCCESS",
			shouldError:      false,
		},
		{
			name:             "Single expected status doesn't match",
			expectedStatuses: []string{"SUCCESS"},
			actualStatus:     "FAILURE",
			shouldError:      true,
			errorMessage:     "finished with status 'FAILURE', but expected one of: [SUCCESS]",
		},
		{
			name:             "Multiple expected statuses - one matches",
			expectedStatuses: []string{"SUCCESS", "FAILURE", "UNKNOWN"},
			actualStatus:     "FAILURE",
			shouldError:      false,
		},
		{
			name:             "Multiple expected statuses - none match",
			expectedStatuses: []string{"SUCCESS", "UNKNOWN"},
			actualStatus:     "FAILURE",
			shouldError:      true,
			errorMessage:     "finished with status 'FAILURE', but expected one of: [SUCCESS UNKNOWN]",
		},
		{
			name:             "Case-insensitive match - lowercase expected, uppercase actual",
			expectedStatuses: []string{"success"},
			actualStatus:     "SUCCESS",
			shouldError:      false,
		},
		{
			name:             "Case-insensitive match - uppercase expected, lowercase actual",
			expectedStatuses: []string{"FAILURE"},
			actualStatus:     "failure",
			shouldError:      false,
		},
		{
			name:             "Case-insensitive match - mixed case",
			expectedStatuses: []string{"SuCcEsS"},
			actualStatus:     "success",
			shouldError:      false,
		},
		{
			name:             "Case-insensitive no match",
			expectedStatuses: []string{"success"},
			actualStatus:     "FAILURE",
			shouldError:      true,
			errorMessage:     "finished with status 'FAILURE', but expected one of: [success]",
		},
		{
			name:             "Multiple expected statuses - case-insensitive match",
			expectedStatuses: []string{"success", "FAILURE", "Unknown"},
			actualStatus:     "UNKNOWN",
			shouldError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(types.BuildStatusResponse{
					ID:     123,
					Status: tt.actualStatus,
					State:  "finished",
				})
			}))
			defer server.Close()

			// Parse server URL
			serverURL, err := url.Parse(server.URL)
			require.NoError(t, err)

			// Create client
			client, err := NewTeamCityClient(serverURL, "testuser", "testpass")
			require.NoError(t, err)

			// Call WaitForBuild
			result, err := client.Build.WaitForBuild("TestBuild", 123, 10*time.Second, tt.expectedStatuses)

			// Assert results
			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, 123, result.ID)
			assert.Equal(t, tt.actualStatus, result.Status)
			assert.Equal(t, "finished", result.State)
		})
	}
}
