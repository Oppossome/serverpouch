package http_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"oppossome/serverpouch/internal/delivery/http"
	"oppossome/serverpouch/internal/domain/usecases"

	mockUsecases "oppossome/serverpouch/internal/common/test/mocks/domain/usecases"

	"github.com/Eun/go-hit"
	"github.com/stretchr/testify/assert"
)

func NewTestServer(t *testing.T) (context.Context, *mockUsecases.MockUsecases, *httptest.Server) {
	mockUsc := mockUsecases.NewMockUsecases(t)
	tCtx := usecases.WithUsecases(t.Context(), mockUsc)

	router, err := http.New(tCtx)
	assert.NoError(t, err, "Failed to initialize testing server")

	return tCtx, mockUsc, httptest.NewServer(router)
}

func hitBodyJSONEquals(t *testing.T, expected interface{}) hit.IStep {
	expectedStr, err := json.Marshal(expected)
	assert.NoError(t, err)

	var expectedMap map[string]interface{}
	err = json.Unmarshal(expectedStr, &expectedMap)
	assert.NoError(t, err)

	return hit.Expect().Body().JSON().Equal(expectedMap)
}
