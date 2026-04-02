package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mostlygeek/llama-swap/event"
	"github.com/mostlygeek/llama-swap/proxy/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRawConfigYAML = `models:
  test-model:
    cmd: echo hello
    proxy: http://localhost:9999
`

func setupConfigAPITestServer(t *testing.T, enableConfig bool, enableModelsDir bool) (*httptest.Server, string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(testRawConfigYAML), 0o644))

	modelsDir := filepath.Join(tmpDir, "models")
	require.NoError(t, os.MkdirAll(modelsDir, 0o755))

	proxyConfig := config.AddDefaultGroupToConfig(config.Config{
		Models: map[string]config.ModelConfig{
			"test-model": {
				Cmd:   "echo hello",
				Proxy: "http://localhost:9999",
			},
		},
		LogToStdout: config.LogToStdoutNone,
		LogLevel:    "error",
	})

	pm := New(proxyConfig)
	t.Cleanup(func() {
		pm.StopProcesses(StopWaitForInflightRequest)
	})

	setConfigPath := ""
	if enableConfig {
		setConfigPath = configPath
	}

	setModelsDir := ""
	if enableModelsDir {
		setModelsDir = modelsDir
	}

	pm.SetConfigPath(setConfigPath, setModelsDir)

	server := httptest.NewServer(pm)
	t.Cleanup(server.Close)

	return server, configPath, modelsDir
}

func doRequest(t *testing.T, method, url string, body []byte, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, respBody
}

func TestConfigAPI_GetConfig(t *testing.T) {
	server, _, _ := setupConfigAPITestServer(t, true, true)

	resp, body := doRequest(t, http.MethodGet, server.URL+"/api/config", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/yaml")
	assert.Contains(t, string(body), "models:")
	assert.Contains(t, string(body), "test-model:")
}

func TestConfigAPI_SaveConfig(t *testing.T) {
	server, configPath, _ := setupConfigAPITestServer(t, true, true)

	updated := []byte(`models:
  test-model:
    cmd: echo updated
    proxy: http://localhost:9999
`)

	resp, body := doRequest(t, http.MethodPut, server.URL+"/api/config", updated, map[string]string{"Content-Type": "text/yaml"})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "ok")

	fileBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(fileBytes), "echo updated")
}

func TestConfigAPI_SaveConfigInvalid(t *testing.T) {
	server, _, _ := setupConfigAPITestServer(t, true, true)

	invalid := []byte("models:\n  bad:\n    cmd:\n")
	resp, _ := doRequest(t, http.MethodPut, server.URL+"/api/config", invalid, map[string]string{"Content-Type": "text/yaml", "Accept": "application/json"})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestConfigAPI_ValidateConfig(t *testing.T) {
	server, _, _ := setupConfigAPITestServer(t, true, true)

	valid := []byte(`models:
  model-a:
    cmd: echo ok
    proxy: http://localhost:9999
`)
	resp, body := doRequest(t, http.MethodPost, server.URL+"/api/config/validate", valid, map[string]string{"Content-Type": "text/yaml"})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"valid":true}`, string(body))

	invalid := []byte("models:\n  broken:\n    cmd:\n")
	resp, body = doRequest(t, http.MethodPost, server.URL+"/api/config/validate", invalid, map[string]string{"Content-Type": "text/yaml"})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `"valid":false`)
	assert.Contains(t, string(body), `"error"`)
}

func TestConfigAPI_ListModels(t *testing.T) {
	server, _, _ := setupConfigAPITestServer(t, true, true)

	resp, body := doRequest(t, http.MethodGet, server.URL+"/api/config/models", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), `"models"`)
	assert.Contains(t, string(body), `"test-model"`)
}

func TestConfigAPI_UpsertModel(t *testing.T) {
	server, configPath, _ := setupConfigAPITestServer(t, true, true)

	createBody := []byte(`{"cmd":"echo new model","proxy":"http://localhost:9001"}`)
	resp, _ := doRequest(t, http.MethodPut, server.URL+"/api/config/models/new-model", createBody, map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := readRawConfig(configPath)
	require.NoError(t, err)
	models, ok := raw["models"].(map[string]any)
	require.True(t, ok)
	_, exists := models["new-model"]
	assert.True(t, exists)

	updateBody := []byte(`{"cmd":"echo updated model","proxy":"http://localhost:9002"}`)
	resp, _ = doRequest(t, http.MethodPut, server.URL+"/api/config/models/test-model", updateBody, map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err = readRawConfig(configPath)
	require.NoError(t, err)
	models = raw["models"].(map[string]any)
	testModel := models["test-model"].(map[string]any)
	assert.Equal(t, "echo updated model", testModel["cmd"])
}

func TestConfigAPI_DeleteModel(t *testing.T) {
	server, configPath, _ := setupConfigAPITestServer(t, true, true)

	resp, _ := doRequest(t, http.MethodDelete, server.URL+"/api/config/models/test-model", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := readRawConfig(configPath)
	require.NoError(t, err)
	models := raw["models"].(map[string]any)
	_, exists := models["test-model"]
	assert.False(t, exists)

	resp, _ = doRequest(t, http.MethodDelete, server.URL+"/api/config/models/missing-model", nil, map[string]string{"Accept": "application/json"})
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestConfigAPI_RenameModel(t *testing.T) {
	server, configPath, _ := setupConfigAPITestServer(t, true, true)

	resp, _ := doRequest(t, http.MethodPost, server.URL+"/api/config/models/rename", []byte(`{"old":"test-model","new":"renamed-model"}`), map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := readRawConfig(configPath)
	require.NoError(t, err)
	models := raw["models"].(map[string]any)
	_, oldExists := models["test-model"]
	_, newExists := models["renamed-model"]
	assert.False(t, oldExists)
	assert.True(t, newExists)

	resp, _ = doRequest(t, http.MethodPut, server.URL+"/api/config/models/existing", []byte(`{"cmd":"echo x","proxy":"http://localhost:9003"}`), map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _ = doRequest(t, http.MethodPost, server.URL+"/api/config/models/rename", []byte(`{"old":"renamed-model","new":"existing"}`), map[string]string{"Content-Type": "application/json", "Accept": "application/json"})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestConfigAPI_SaveMacros(t *testing.T) {
	server, configPath, _ := setupConfigAPITestServer(t, true, true)

	resp, _ := doRequest(t, http.MethodPost, server.URL+"/api/config/macros", []byte(`{"PORT":"9000","MODEL_DIR":"/models"}`), map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := readRawConfig(configPath)
	require.NoError(t, err)
	macros, ok := raw["macros"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "9000", macros["PORT"])
	assert.Equal(t, "/models", macros["MODEL_DIR"])
}

func TestConfigAPI_ListFiles(t *testing.T) {
	server, _, modelsDir := setupConfigAPITestServer(t, true, true)

	require.NoError(t, os.WriteFile(filepath.Join(modelsDir, "a.gguf"), []byte("1234"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(modelsDir, "nested"), 0o755))

	resp, body := doRequest(t, http.MethodGet, server.URL+"/api/config/files", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	entries, ok := payload["entries"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(entries), 2)
	assert.Equal(t, "", payload["current"])
}

func TestConfigAPI_DeleteFile(t *testing.T) {
	server, _, modelsDir := setupConfigAPITestServer(t, true, true)

	targetFile := filepath.Join(modelsDir, "delete-me.gguf")
	require.NoError(t, os.WriteFile(targetFile, []byte("to-delete"), 0o644))

	resp, _ := doRequest(t, http.MethodDelete, server.URL+"/api/config/files", []byte(`{"path":"delete-me.gguf"}`), map[string]string{"Content-Type": "application/json"})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, err := os.Stat(targetFile)
	assert.True(t, os.IsNotExist(err))

	resp, _ = doRequest(t, http.MethodDelete, server.URL+"/api/config/files", []byte(`{"path":"../outside.txt"}`), map[string]string{"Content-Type": "application/json", "Accept": "application/json"})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestConfigAPI_Disabled(t *testing.T) {
	server, _, _ := setupConfigAPITestServer(t, false, false)

	resp, _ := doRequest(t, http.MethodGet, server.URL+"/api/config", nil, map[string]string{"Accept": "application/json"})
	require.Equal(t, http.StatusNotImplemented, resp.StatusCode)

	resp, _ = doRequest(t, http.MethodGet, server.URL+"/api/config/files", nil, map[string]string{"Accept": "application/json"})
	require.Equal(t, http.StatusNotImplemented, resp.StatusCode)
}

func TestConfigAPI_ReloadConfig(t *testing.T) {
	server, _, _ := setupConfigAPITestServer(t, true, true)

	evtCh := make(chan ConfigFileChangedEvent, 1)
	cancel := event.On(func(e ConfigFileChangedEvent) {
		if e.ReloadingState == ReloadingStateStart {
			select {
			case evtCh <- e:
			default:
			}
		}
	})
	defer cancel()

	resp, body := doRequest(t, http.MethodPost, server.URL+"/api/config/reload", nil, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, string(body), "ok")

	select {
	case <-evtCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for config reload event")
	}
}
