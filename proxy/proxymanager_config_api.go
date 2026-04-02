package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mostlygeek/llama-swap/event"
	"github.com/mostlygeek/llama-swap/proxy/config"
	"gopkg.in/yaml.v3"
)

func addConfigApiHandlers(pm *ProxyManager) {
	configGroup := pm.ginEngine.Group("/api/config", pm.apiKeyAuth())
	{
		configGroup.GET("", pm.configGetRawConfig)
		configGroup.PUT("", pm.configSaveRawConfig)
		configGroup.POST("/validate", pm.configValidateConfig)
		configGroup.POST("/reload", pm.configReloadConfig)

		configGroup.GET("/models", pm.configListModels)
		configGroup.PUT("/models/:name", pm.configUpsertModel)
		configGroup.DELETE("/models/:name", pm.configDeleteModel)
		configGroup.POST("/models/rename", pm.configRenameModel)

		configGroup.POST("/macros", pm.configSaveMacros)

		configGroup.GET("/files", pm.configListFiles)
		configGroup.DELETE("/files", pm.configDeleteFile)
	}
}

func readRawConfig(path string) (map[string]any, error) {
	yamlBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	data := map[string]any{}
	if len(bytes.TrimSpace(yamlBytes)) == 0 {
		return data, nil
	}

	if err := yaml.Unmarshal(yamlBytes, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func writeRawConfig(path string, data map[string]any) error {
	rootNode := toYAMLNode(data)
	if rootNode == nil {
		rootNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}

	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{rootNode}}

	buf := bytes.Buffer{}
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}

	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func toYAMLNode(value any) *yaml.Node {
	switch v := value.(type) {
	case map[string]any:
		n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for k, mv := range v {
			n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}, toYAMLNode(mv))
		}
		return n
	case []any:
		n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, item := range v {
			n.Content = append(n.Content, toYAMLNode(item))
		}
		return n
	case string:
		n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
		if strings.Contains(v, "\n") {
			n.Style = yaml.FoldedStyle
		}
		return n
	case int:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", v)}
	case int64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", v)}
	case float64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: fmt.Sprintf("%v", v)}
	case bool:
		if v {
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"}
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"}
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
	default:
		n := &yaml.Node{}
		if err := n.Encode(v); err != nil {
			return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: fmt.Sprintf("%v", v)}
		}
		if n.Kind == yaml.ScalarNode && strings.Contains(n.Value, "\n") {
			n.Style = yaml.FoldedStyle
		}
		return n
	}
}

func validateConfigBytes(yamlBytes []byte) error {
	_, err := config.LoadConfigFromReader(bytes.NewReader(yamlBytes))
	return err
}

func (pm *ProxyManager) configGetRawConfig(c *gin.Context) {
	if pm.configPath == "" {
		pm.sendErrorResponse(c, http.StatusNotImplemented, "config management not enabled")
		return
	}

	yamlBytes, err := os.ReadFile(pm.configPath)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.Header("Content-Type", "text/yaml; charset=utf-8")
	c.String(http.StatusOK, string(yamlBytes))
}

func (pm *ProxyManager) configSaveRawConfig(c *gin.Context) {
	if pm.configPath == "" {
		pm.sendErrorResponse(c, http.StatusNotImplemented, "config management not enabled")
		return
	}

	yamlBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, "failed to read request body")
		return
	}

	if err := validateConfigBytes(yamlBytes); err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := os.WriteFile(pm.configPath, yamlBytes, 0o644); err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	event.Emit(ConfigFileChangedEvent{ReloadingState: ReloadingStateStart})
	pm.proxyLogger.Infof("Config saved: %s", pm.configPath)
	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

func (pm *ProxyManager) configValidateConfig(c *gin.Context) {
	yamlBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, "failed to read request body")
		return
	}

	if err := validateConfigBytes(yamlBytes); err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

func (pm *ProxyManager) configReloadConfig(c *gin.Context) {
	if pm.configPath == "" {
		pm.sendErrorResponse(c, http.StatusNotImplemented, "config management not enabled")
		return
	}

	event.Emit(ConfigFileChangedEvent{ReloadingState: ReloadingStateStart})
	pm.proxyLogger.Infof("Config reload requested")
	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

func (pm *ProxyManager) configListModels(c *gin.Context) {
	if pm.configPath == "" {
		pm.sendErrorResponse(c, http.StatusNotImplemented, "config management not enabled")
		return
	}

	rawConfig, err := readRawConfig(pm.configPath)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	models, ok := rawConfig["models"].(map[string]any)
	if !ok {
		models = map[string]any{}
	}

	c.JSON(http.StatusOK, gin.H{"models": models})
}

func (pm *ProxyManager) configUpsertModel(c *gin.Context) {
	if pm.configPath == "" {
		pm.sendErrorResponse(c, http.StatusNotImplemented, "config management not enabled")
		return
	}

	name := c.Param("name")
	if name == "" {
		pm.sendErrorResponse(c, http.StatusBadRequest, "model name is required")
		return
	}

	rawConfig, err := readRawConfig(pm.configPath)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, "failed to read request body")
		return
	}

	body := map[string]any{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, "invalid JSON body")
		return
	}

	models, ok := rawConfig["models"].(map[string]any)
	if !ok {
		models = map[string]any{}
		rawConfig["models"] = models
	}

	models[name] = body
	if err := writeRawConfig(pm.configPath, rawConfig); err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	event.Emit(ConfigFileChangedEvent{ReloadingState: ReloadingStateStart})
	pm.proxyLogger.Infof("Config model upserted: %s", name)
	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

func (pm *ProxyManager) configDeleteModel(c *gin.Context) {
	if pm.configPath == "" {
		pm.sendErrorResponse(c, http.StatusNotImplemented, "config management not enabled")
		return
	}

	name := c.Param("name")
	if name == "" {
		pm.sendErrorResponse(c, http.StatusBadRequest, "model name is required")
		return
	}

	rawConfig, err := readRawConfig(pm.configPath)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	models, ok := rawConfig["models"].(map[string]any)
	if !ok {
		pm.sendErrorResponse(c, http.StatusNotFound, "model not found")
		return
	}

	if _, exists := models[name]; !exists {
		pm.sendErrorResponse(c, http.StatusNotFound, "model not found")
		return
	}

	delete(models, name)
	if err := writeRawConfig(pm.configPath, rawConfig); err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	event.Emit(ConfigFileChangedEvent{ReloadingState: ReloadingStateStart})
	pm.proxyLogger.Infof("Config model deleted: %s", name)
	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

type configRenameRequest struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func (pm *ProxyManager) configRenameModel(c *gin.Context) {
	if pm.configPath == "" {
		pm.sendErrorResponse(c, http.StatusNotImplemented, "config management not enabled")
		return
	}

	body := configRenameRequest{}
	if err := c.ShouldBindJSON(&body); err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if body.Old == "" || body.New == "" {
		pm.sendErrorResponse(c, http.StatusBadRequest, "old and new model names are required")
		return
	}

	rawConfig, err := readRawConfig(pm.configPath)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	models, ok := rawConfig["models"].(map[string]any)
	if !ok {
		pm.sendErrorResponse(c, http.StatusNotFound, "model not found")
		return
	}

	oldModel, exists := models[body.Old]
	if !exists {
		pm.sendErrorResponse(c, http.StatusNotFound, "model not found")
		return
	}

	if _, conflict := models[body.New]; conflict {
		pm.sendErrorResponse(c, http.StatusBadRequest, "target model already exists")
		return
	}

	delete(models, body.Old)
	models[body.New] = oldModel

	if err := writeRawConfig(pm.configPath, rawConfig); err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	event.Emit(ConfigFileChangedEvent{ReloadingState: ReloadingStateStart})
	pm.proxyLogger.Infof("Config model renamed: %s -> %s", body.Old, body.New)
	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

func (pm *ProxyManager) configSaveMacros(c *gin.Context) {
	if pm.configPath == "" {
		pm.sendErrorResponse(c, http.StatusNotImplemented, "config management not enabled")
		return
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, "failed to read request body")
		return
	}

	macros := map[string]any{}
	if err := json.Unmarshal(bodyBytes, &macros); err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, "invalid JSON body")
		return
	}

	rawConfig, err := readRawConfig(pm.configPath)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	rawConfig["macros"] = macros
	if err := writeRawConfig(pm.configPath, rawConfig); err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	event.Emit(ConfigFileChangedEvent{ReloadingState: ReloadingStateStart})
	pm.proxyLogger.Infof("Config macros updated")
	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

func (pm *ProxyManager) configListFiles(c *gin.Context) {
	if pm.modelsDir == "" {
		pm.sendErrorResponse(c, http.StatusNotImplemented, "models directory not configured")
		return
	}

	subpath := c.DefaultQuery("path", "")
	target, err := resolveModelPath(pm.modelsDir, subpath)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		if os.IsNotExist(err) {
			pm.sendErrorResponse(c, http.StatusNotFound, "path not found")
			return
		}
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	responseEntries := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		entryPath := filepath.Join(target, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		relPath, err := filepath.Rel(pm.modelsDir, entryPath)
		if err != nil {
			continue
		}

		sizeBytes := info.Size()
		responseEntries = append(responseEntries, gin.H{
			"name":       entry.Name(),
			"path":       filepath.ToSlash(relPath),
			"is_dir":     entry.IsDir(),
			"size":       humanReadableSize(sizeBytes),
			"size_bytes": sizeBytes,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"entries": responseEntries,
		"current": subpath,
	})
}

type configDeleteFileRequest struct {
	Path string `json:"path"`
}

func (pm *ProxyManager) configDeleteFile(c *gin.Context) {
	if pm.modelsDir == "" {
		pm.sendErrorResponse(c, http.StatusNotImplemented, "models directory not configured")
		return
	}

	body := configDeleteFileRequest{}
	if err := c.ShouldBindJSON(&body); err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if body.Path == "" {
		pm.sendErrorResponse(c, http.StatusBadRequest, "path is required")
		return
	}

	target, err := resolveModelPath(pm.modelsDir, body.Path)
	if err != nil {
		pm.sendErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			pm.sendErrorResponse(c, http.StatusNotFound, "path not found")
			return
		}
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if info.IsDir() {
		err = os.RemoveAll(target)
	} else {
		err = os.Remove(target)
	}
	if err != nil {
		pm.sendErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	pm.proxyLogger.Infof("Deleted file path: %s", body.Path)
	c.JSON(http.StatusOK, gin.H{"msg": "ok"})
}

func resolveModelPath(modelsDir, subpath string) (string, error) {
	base := filepath.Clean(modelsDir)
	target := filepath.Clean(filepath.Join(base, subpath))

	if target != base && !strings.HasPrefix(target, base+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid path")
	}

	return target, nil
}

func humanReadableSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}

	units := []string{"KB", "MB", "GB", "TB"}
	fSize := float64(size)
	for _, unit := range units {
		fSize = fSize / 1024
		if fSize < 1024 {
			return fmt.Sprintf("%.1f %s", fSize, unit)
		}
	}

	return fmt.Sprintf("%.1f TB", fSize)
}
