package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveSetupConfigSQLitePathWithHash(t *testing.T) {
	originalWD, err := os.Getwd()
	require.NoError(t, err)

	tempDir := t.TempDir()
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterSetupRoutes(r)

	body := []byte(`{"engine":"sqlite","sqlitePath":"./data/status#forge.db"}`)
	req, err := http.NewRequest(http.MethodPost, "/api/setup/save", bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var resp setupStatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.SetupDone)
	assert.Equal(t, "sqlite", resp.Engine)
	assert.Equal(t, "sqlite", resp.DBStatus.Engine)

	_, statErr := os.Stat(filepath.Join(tempDir, "data", "status#forge.db"))
	assert.NoError(t, statErr)

	envBytes, readErr := os.ReadFile(filepath.Join(tempDir, ".env"))
	require.NoError(t, readErr)
	assert.Contains(t, string(envBytes), "DB_ENGINE=\"sqlite\"")
	assert.Contains(t, string(envBytes), "SQLITE_PATH=\"./data/status#forge.db\"")
}
