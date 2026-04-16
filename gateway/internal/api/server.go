package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"code-agent-gateway/common/domain"
	"code-agent-gateway/common/protocol"
	"code-agent-gateway/common/transport"
	"code-agent-gateway/gateway/internal/registry"
	"code-agent-gateway/gateway/internal/routing"
	"code-agent-gateway/gateway/internal/runtimeindex"
	"code-agent-gateway/gateway/internal/settings"
	toml "github.com/pelletier/go-toml/v2"
)

type CommandSender interface {
	SendCommand(ctx context.Context, machineID string, name string, payload any) (protocol.CommandCompletedPayload, error)
}

type approvalRequestResolver interface {
	ResolveApprovalMachine(requestID string) (string, bool)
}

type approvalRequestCleaner interface {
	ClearApprovalRequest(requestID string)
}

type threadDetailResponse struct {
	Thread           domain.Thread                      `json:"thread"`
	ActiveTurnID     string                             `json:"activeTurnId,omitempty"`
	PendingApprovals []protocol.ApprovalRequiredPayload `json:"pendingApprovals"`
}

type threadDeleteResponse struct {
	ThreadID string `json:"threadId"`
	Deleted  bool   `json:"deleted"`
	Archived bool   `json:"archived"`
}

type createThreadRequest struct {
	MachineID string `json:"machineId"`
	AgentID   string `json:"agentId"`
	Title     string `json:"title"`
}

type threadRenameRequest struct {
	Title string `json:"title"`
}

type startTurnRequest struct {
	Input string `json:"input"`
}

type approvalRespondRequest struct {
	Decision string         `json:"decision"`
	Answers  map[string]any `json:"answers"`
}

type environmentMutationRequest struct {
	MachineID string `json:"machineId"`
	AgentID   string `json:"agentId"`
}

type environmentSkillCreateRequest struct {
	MachineID   string `json:"machineId"`
	AgentID     string `json:"agentId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type environmentMCPUpsertRequest struct {
	MachineID  string         `json:"machineId"`
	AgentID    string         `json:"agentId"`
	ResourceID string         `json:"resourceId"`
	Config     map[string]any `json:"config"`
}

type environmentPluginInstallRequest struct {
	MachineID       string `json:"machineId"`
	AgentID         string `json:"agentId"`
	PluginID        string `json:"pluginId"`
	PluginName      string `json:"pluginName"`
	MarketplacePath string `json:"marketplacePath"`
}

type machineAgentInstallRequest struct {
	AgentType   string `json:"agentType"`
	DisplayName string `json:"displayName"`
}

type configDocumentRequest struct {
	Content string `json:"content"`
}

type consoleSettingsRequest struct {
	Preferences *domain.ConsolePreferences `json:"preferences"`
}

type settingsApplyResponse struct {
	MachineID string `json:"machineId"`
	AgentType string `json:"agentType"`
	Source    string `json:"source"`
	FilePath  string `json:"filePath,omitempty"`
}

type activeTurnReader interface {
	ActiveTurnID(threadID string) (string, bool)
}

type threadUpdateEmitter interface {
	EmitThreadUpdated(payload protocol.ThreadUpdatedPayload, timestamp string)
}

func resolveThreadRoute(router *routing.Router, idx *runtimeindex.Store, threadID string) (domain.ThreadRoute, bool) {
	if router != nil {
		if route, ok := router.ResolveThread(threadID); ok {
			return route, true
		}
	}
	if idx != nil {
		if route, ok := idx.ThreadRoute(threadID); ok && strings.TrimSpace(route.MachineID) != "" {
			return route, true
		}
		if thread, ok := findThread(idx, threadID); ok && strings.TrimSpace(thread.MachineID) != "" {
			return domain.ThreadRoute{
				MachineID: thread.MachineID,
				AgentID:   thread.AgentID,
			}, true
		}
	}
	return domain.ThreadRoute{}, false
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func findThread(idx *runtimeindex.Store, threadID string) (domain.Thread, bool) {
	if idx == nil || strings.TrimSpace(threadID) == "" {
		return domain.Thread{}, false
	}
	for _, thread := range idx.Threads() {
		if thread.ThreadID == threadID {
			return thread, true
		}
	}
	return domain.Thread{}, false
}

func resolveThreadTitleOverrides(store settings.Store) map[string]string {
	if store == nil {
		return nil
	}
	preferences, ok, err := store.GetConsolePreferences()
	if err != nil || !ok || len(preferences.ThreadTitles) == 0 {
		return nil
	}
	return preferences.ThreadTitles
}

func applyThreadTitleOverride(thread domain.Thread, overrides map[string]string) domain.Thread {
	if overrides == nil {
		return thread
	}
	if title, ok := overrides[thread.ThreadID]; ok && strings.TrimSpace(title) != "" {
		thread.Title = title
	}
	return thread
}

func upsertMachineAgent(items []domain.AgentInstance, agent domain.AgentInstance) []domain.AgentInstance {
	if strings.TrimSpace(agent.AgentID) == "" {
		return append([]domain.AgentInstance(nil), items...)
	}

	next := append([]domain.AgentInstance(nil), items...)
	for idx := range next {
		if next[idx].AgentID != agent.AgentID {
			continue
		}
		next[idx] = agent
		return next
	}
	return append(next, agent)
}

func removeMachineAgent(items []domain.AgentInstance, agentID string) []domain.AgentInstance {
	if strings.TrimSpace(agentID) == "" {
		return append([]domain.AgentInstance(nil), items...)
	}

	next := make([]domain.AgentInstance, 0, len(items))
	for _, item := range items {
		if item.AgentID == agentID {
			continue
		}
		next = append(next, item)
	}
	return next
}

func findEnvironmentResource(idx *runtimeindex.Store, kind domain.EnvironmentKind, machineID string, agentID string, resourceID string) (domain.EnvironmentResource, bool) {
	if idx == nil || strings.TrimSpace(machineID) == "" || strings.TrimSpace(resourceID) == "" {
		return domain.EnvironmentResource{}, false
	}
	for _, resource := range idx.Environment(kind) {
		if resource.ResourceID == resourceID && resource.MachineID == machineID {
			if strings.TrimSpace(agentID) != "" && resource.AgentID != agentID {
				continue
			}
			return resource, true
		}
	}
	return domain.EnvironmentResource{}, false
}

func decodeEnvironmentMutationRequest(r *http.Request) (environmentMutationRequest, error) {
	var req environmentMutationRequest
	if r == nil {
		return req, nil
	}

	req.MachineID = strings.TrimSpace(r.URL.Query().Get("machineId"))
	req.AgentID = strings.TrimSpace(r.URL.Query().Get("agentId"))

	if r.Body == nil {
		return req, nil
	}

	var body environmentMutationRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if err == io.EOF {
			return req, nil
		}
		return environmentMutationRequest{}, err
	}

	if req.MachineID == "" {
		req.MachineID = strings.TrimSpace(body.MachineID)
	}
	if req.AgentID == "" {
		req.AgentID = strings.TrimSpace(body.AgentID)
	}

	return req, nil
}

func decodeEnvironmentMCPUpsertRequest(r *http.Request) (environmentMCPUpsertRequest, error) {
	if r == nil || r.Body == nil {
		return environmentMCPUpsertRequest{}, nil
	}

	var req environmentMCPUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			return environmentMCPUpsertRequest{}, nil
		}
		return environmentMCPUpsertRequest{}, err
	}
	return req, nil
}

func decodeEnvironmentPluginInstallRequest(r *http.Request) (environmentPluginInstallRequest, error) {
	var req environmentPluginInstallRequest
	if r == nil {
		return req, nil
	}

	req.MachineID = strings.TrimSpace(r.URL.Query().Get("machineId"))
	req.AgentID = strings.TrimSpace(r.URL.Query().Get("agentId"))
	if r.Body == nil {
		return req, nil
	}

	var body environmentPluginInstallRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		if err == io.EOF {
			return req, nil
		}
		return environmentPluginInstallRequest{}, err
	}
	if req.MachineID == "" {
		req.MachineID = strings.TrimSpace(body.MachineID)
	}
	if req.AgentID == "" {
		req.AgentID = strings.TrimSpace(body.AgentID)
	}
	req.PluginID = strings.TrimSpace(body.PluginID)
	req.PluginName = strings.TrimSpace(body.PluginName)
	req.MarketplacePath = strings.TrimSpace(body.MarketplacePath)
	return req, nil
}

func stringDetail(resource domain.EnvironmentResource, key string) string {
	if resource.Details == nil || strings.TrimSpace(key) == "" {
		return ""
	}
	value, ok := resource.Details[key]
	if !ok {
		return ""
	}
	stringValue, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringValue)
}

func allowedMarketplacePaths(idx *runtimeindex.Store, machineID string, agentID string) map[string]struct{} {
	if idx == nil || strings.TrimSpace(machineID) == "" {
		return nil
	}

	allowed := map[string]struct{}{}
	for _, resource := range idx.Environment(domain.EnvironmentKindPlugin) {
		if resource.MachineID != machineID {
			continue
		}
		if strings.TrimSpace(agentID) != "" && resource.AgentID != agentID {
			continue
		}
		path := stringDetail(resource, "marketplacePath")
		if path == "" {
			continue
		}
		allowed[path] = struct{}{}
	}
	return allowed
}

func machineIsOnline(reg *registry.Store, machineID string) bool {
	if reg == nil || strings.TrimSpace(machineID) == "" {
		return false
	}
	machine, ok := reg.Get(machineID)
	return ok && machine.Status == domain.MachineStatusOnline
}

func resolveActiveTurnID(sender CommandSender, threadID string) string {
	if sender == nil || strings.TrimSpace(threadID) == "" {
		return ""
	}

	reader, ok := sender.(activeTurnReader)
	if !ok {
		return ""
	}

	activeTurnID, ok := reader.ActiveTurnID(threadID)
	if !ok {
		return ""
	}
	return activeTurnID
}

func NewServer(reg *registry.Store, idx *runtimeindex.Store, router *routing.Router, sender CommandSender, clientWS http.Handler, consoleWS http.Handler) http.Handler {
	apiKey, hasAPIKey := os.LookupEnv("GATEWAY_API_KEY")
	return newServerWithSettingsAndAPIKey(reg, idx, router, sender, nil, strings.TrimSpace(apiKey), clientWS, consoleWS, hasAPIKey)
}

func NewServerWithAPIKey(reg *registry.Store, idx *runtimeindex.Store, router *routing.Router, sender CommandSender, apiKey string, clientWS http.Handler, consoleWS http.Handler) http.Handler {
	return newServerWithSettingsAndAPIKey(reg, idx, router, sender, nil, apiKey, clientWS, consoleWS, true)
}

func defaultAgentDescriptors() []domain.AgentDescriptor {
	return []domain.AgentDescriptor{
		{
			AgentType:   domain.AgentTypeCodex,
			DisplayName: "Codex",
		},
	}
}

func resolveAgentType(raw string) (domain.AgentType, bool) {
	switch domain.AgentType(strings.TrimSpace(raw)) {
	case domain.AgentTypeCodex:
		return domain.AgentTypeCodex, true
	default:
		return "", false
	}
}

func decodeConfigDocumentRequest(r *http.Request) (configDocumentRequest, error) {
	if r == nil || r.Body == nil {
		return configDocumentRequest{}, nil
	}

	var req configDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err == io.EOF {
			return configDocumentRequest{}, nil
		}
		return configDocumentRequest{}, err
	}
	return req, nil
}

func validateTOMLContent(content string) error {
	var document map[string]any
	return toml.Unmarshal([]byte(content), &document)
}

func buildCapabilitySnapshot(reg *registry.Store, idx *runtimeindex.Store, router *routing.Router, sender CommandSender, settingsStore settings.Store) domain.CapabilitySnapshot {
	hasSender := sender != nil
	hasRegistry := reg != nil
	hasRuntimeIndex := idx != nil
	hasRouter := router != nil
	hasSettings := settingsStore != nil

	approvals := false
	if hasSender {
		if _, ok := sender.(approvalRequestResolver); ok {
			approvals = true
		}
	}

	threadRouting := hasRouter || hasRuntimeIndex
	threadHub := hasRuntimeIndex && hasRegistry
	threadWorkspace := hasRuntimeIndex || (hasSender && hasRouter)
	environmentMutate := hasSender && hasRuntimeIndex

	return domain.CapabilitySnapshot{
		ThreadHub:                   threadHub,
		ThreadWorkspace:             threadWorkspace,
		Approvals:                   approvals,
		StartTurn:                   hasSender && threadRouting,
		SteerTurn:                   hasSender && threadRouting,
		InterruptTurn:               hasSender && threadRouting,
		MachineInstallAgent:         hasSender && hasRegistry,
		MachineRemoveAgent:          hasSender && hasRegistry,
		EnvironmentSyncCatalog:      hasSender && hasRegistry,
		EnvironmentRestartBridge:    hasSender && hasRegistry,
		EnvironmentOpenMarketplace:  true,
		EnvironmentMutateResources:  environmentMutate,
		EnvironmentWriteMcp:         hasSender,
		EnvironmentWriteSkills:      hasSender,
		SettingsEditGatewayEndpoint: hasSettings,
		SettingsEditConsoleProfile:  hasSettings,
		SettingsEditSafetyPolicy:    hasSettings,
		SettingsGlobalDefault:       hasSettings,
		SettingsMachineOverride:     hasSettings,
		SettingsApplyMachine:        hasSettings && hasSender,
		DashboardMetrics:            hasRegistry && hasRuntimeIndex,
		AgentLifecycle:              hasSender && hasRegistry,
	}
}

func NewServerWithSettings(reg *registry.Store, idx *runtimeindex.Store, router *routing.Router, sender CommandSender, settingsStore settings.Store, clientWS http.Handler, consoleWS http.Handler) http.Handler {
	apiKey, hasAPIKey := os.LookupEnv("GATEWAY_API_KEY")
	return newServerWithSettingsAndAPIKey(reg, idx, router, sender, settingsStore, strings.TrimSpace(apiKey), clientWS, consoleWS, hasAPIKey)
}

func NewServerWithSettingsAndAPIKey(reg *registry.Store, idx *runtimeindex.Store, router *routing.Router, sender CommandSender, settingsStore settings.Store, apiKey string, clientWS http.Handler, consoleWS http.Handler) http.Handler {
	return newServerWithSettingsAndAPIKey(reg, idx, router, sender, settingsStore, apiKey, clientWS, consoleWS, true)
}

func newServerWithSettingsAndAPIKey(reg *registry.Store, idx *runtimeindex.Store, router *routing.Router, sender CommandSender, settingsStore settings.Store, apiKey string, clientWS http.Handler, consoleWS http.Handler, failClosedOnBlankKey bool) http.Handler {
	mux := http.NewServeMux()
	var deletedThreadsMu sync.RWMutex
	deletedThreads := map[string]struct{}{}

	if settingsStore == nil {
		settingsStore = settings.NewMemoryStore(defaultAgentDescriptors())
	}

	isDeleted := func(threadID string) bool {
		deletedThreadsMu.RLock()
		defer deletedThreadsMu.RUnlock()
		_, ok := deletedThreads[threadID]
		return ok
	}
	markDeleted := func(threadID string) {
		if strings.TrimSpace(threadID) == "" {
			return
		}
		deletedThreadsMu.Lock()
		deletedThreads[threadID] = struct{}{}
		deletedThreadsMu.Unlock()
	}
	clearDeleted := func(threadID string) {
		if strings.TrimSpace(threadID) == "" {
			return
		}
		deletedThreadsMu.Lock()
		delete(deletedThreads, threadID)
		deletedThreadsMu.Unlock()
	}

	if clientWS != nil {
		mux.Handle("/ws/client", clientWS)
	}
	if consoleWS != nil {
		mux.Handle("/ws", consoleWS)
	}

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	mux.HandleFunc("GET /capabilities", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, buildCapabilitySnapshot(reg, idx, router, sender, settingsStore))
	})

	mux.HandleFunc("GET /overview/metrics", func(w http.ResponseWriter, _ *http.Request) {
		metrics := domain.OverviewMetrics{}
		if idx != nil {
			metrics = idx.OverviewMetrics()
		}
		if reg != nil {
			for _, machine := range reg.List() {
				if machine.Status == domain.MachineStatusOnline {
					metrics.OnlineMachines++
				}
				for _, agent := range machine.Agents {
					if agent.Status == domain.AgentInstanceStatusRunning {
						metrics.RunningAgents++
					}
				}
			}
			metrics.PendingApprovals = reg.PendingApprovalCount()
		}
		writeJSON(w, http.StatusOK, metrics)
	})

	mux.HandleFunc("POST /environment/sync", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil || reg == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		targeted := 0
		for _, machine := range reg.List() {
			if machine.Status != domain.MachineStatusOnline {
				continue
			}
			if _, err := sender.SendCommand(r.Context(), machine.ID, "environment.refresh", protocol.EnvironmentRefreshCommandPayload{}); err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			targeted++
		}

		writeJSON(w, http.StatusOK, map[string]any{"targetedMachines": targeted})
	})

	mux.HandleFunc("POST /environment/mcps/restart-bridge", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil || reg == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		targeted := 0
		for _, machine := range reg.List() {
			if machine.Status != domain.MachineStatusOnline {
				continue
			}
			if _, err := sender.SendCommand(r.Context(), machine.ID, "environment.mcp.reload", protocol.EnvironmentMCPReloadCommandPayload{}); err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			targeted++
		}

		writeJSON(w, http.StatusOK, map[string]any{"targetedMachines": targeted})
	})

	mux.HandleFunc("GET /settings/agents", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": settingsStore.ListAgentTypes()})
	})

	mux.HandleFunc("GET /settings/console", func(w http.ResponseWriter, _ *http.Request) {
		preferences, ok, err := settingsStore.GetConsolePreferences()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{"preferences": nil})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"preferences": preferences})
	})

	mux.HandleFunc("PUT /settings/console", func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "preferences is required", http.StatusBadRequest)
			return
		}

		var req consoleSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if err == io.EOF {
				http.Error(w, "preferences is required", http.StatusBadRequest)
				return
			}
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if req.Preferences == nil {
			http.Error(w, "preferences is required", http.StatusBadRequest)
			return
		}
		if err := settingsStore.PutConsolePreferences(*req.Preferences); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"preferences": req.Preferences})
	})

	mux.HandleFunc("GET /settings/agents/{agentType}/global", func(w http.ResponseWriter, r *http.Request) {
		agentType, ok := resolveAgentType(r.PathValue("agentType"))
		if !ok {
			http.Error(w, "agentType is not supported", http.StatusNotFound)
			return
		}
		document, exists, err := settingsStore.GetGlobal(agentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !exists {
			writeJSON(w, http.StatusOK, map[string]any{"document": nil})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"document": document})
	})

	mux.HandleFunc("PUT /settings/agents/{agentType}/global", func(w http.ResponseWriter, r *http.Request) {
		agentType, ok := resolveAgentType(r.PathValue("agentType"))
		if !ok {
			http.Error(w, "agentType is not supported", http.StatusNotFound)
			return
		}
		req, err := decodeConfigDocumentRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			http.Error(w, "content is required", http.StatusBadRequest)
			return
		}
		if err := validateTOMLContent(req.Content); err != nil {
			http.Error(w, "content must be valid toml", http.StatusBadRequest)
			return
		}
		document := domain.AgentConfigDocument{
			AgentType: agentType,
			Format:    domain.AgentConfigFormatTOML,
			Content:   req.Content,
		}
		if err := settingsStore.PutGlobal(agentType, document); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		document, _, _ = settingsStore.GetGlobal(agentType)
		writeJSON(w, http.StatusOK, map[string]any{"document": document})
	})

	mux.HandleFunc("GET /settings/machines/{machineId}/agents/{agentType}", func(w http.ResponseWriter, r *http.Request) {
		machineID := strings.TrimSpace(r.PathValue("machineId"))
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		agentType, ok := resolveAgentType(r.PathValue("agentType"))
		if !ok {
			http.Error(w, "agentType is not supported", http.StatusNotFound)
			return
		}

		globalDocument, globalExists, err := settingsStore.GetGlobal(agentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		machineDocument, machineExists, err := settingsStore.GetMachine(machineID, agentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := domain.MachineAgentConfigAssignment{
			MachineID:         machineID,
			AgentType:         agentType,
			UsesGlobalDefault: !machineExists,
		}
		if globalExists {
			response.GlobalDefault = &globalDocument
		}
		if machineExists {
			response.MachineOverride = &machineDocument
		}
		writeJSON(w, http.StatusOK, response)
	})

	mux.HandleFunc("PUT /settings/machines/{machineId}/agents/{agentType}", func(w http.ResponseWriter, r *http.Request) {
		machineID := strings.TrimSpace(r.PathValue("machineId"))
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		agentType, ok := resolveAgentType(r.PathValue("agentType"))
		if !ok {
			http.Error(w, "agentType is not supported", http.StatusNotFound)
			return
		}
		req, err := decodeConfigDocumentRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			http.Error(w, "content is required", http.StatusBadRequest)
			return
		}
		if err := validateTOMLContent(req.Content); err != nil {
			http.Error(w, "content must be valid toml", http.StatusBadRequest)
			return
		}
		document := domain.AgentConfigDocument{
			AgentType: agentType,
			Format:    domain.AgentConfigFormatTOML,
			Content:   req.Content,
		}
		if err := settingsStore.PutMachine(machineID, agentType, document); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		document, _, _ = settingsStore.GetMachine(machineID, agentType)
		writeJSON(w, http.StatusOK, map[string]any{"document": document})
	})

	mux.HandleFunc("DELETE /settings/machines/{machineId}/agents/{agentType}", func(w http.ResponseWriter, r *http.Request) {
		machineID := strings.TrimSpace(r.PathValue("machineId"))
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		agentType, ok := resolveAgentType(r.PathValue("agentType"))
		if !ok {
			http.Error(w, "agentType is not supported", http.StatusNotFound)
			return
		}
		if err := settingsStore.DeleteMachine(machineID, agentType); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /settings/machines/{machineId}/agents/{agentType}/apply", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID := strings.TrimSpace(r.PathValue("machineId"))
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		agentType, ok := resolveAgentType(r.PathValue("agentType"))
		if !ok {
			http.Error(w, "agentType is not supported", http.StatusNotFound)
			return
		}

		document, machineExists, err := settingsStore.GetMachine(machineID, agentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		source := "machine"
		if !machineExists {
			document, machineExists, err = settingsStore.GetGlobal(agentType)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			source = "global"
		}
		if !machineExists {
			http.Error(w, "no config document available to apply", http.StatusConflict)
			return
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "agent.config.apply", protocol.AgentConfigApplyCommandPayload{
			AgentType: string(agentType),
			Source:    source,
			Document:  document,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.AgentConfigApplyCommandResult
		if err := transport.Decode(completed.Result, &result); err != nil {
			http.Error(w, "invalid agent.config.apply result", http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, settingsApplyResponse{
			MachineID: machineID,
			AgentType: string(agentType),
			Source:    source,
			FilePath:  result.FilePath,
		})
	})

	mux.HandleFunc("POST /machines/{machineId}/agents", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID := strings.TrimSpace(r.PathValue("machineId"))
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if r.Body == nil {
			http.Error(w, "request body is required", http.StatusBadRequest)
			return
		}

		var req machineAgentInstallRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if err == io.EOF {
				http.Error(w, "request body is required", http.StatusBadRequest)
				return
			}
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if _, ok := resolveAgentType(req.AgentType); !ok {
			http.Error(w, "agentType is not supported", http.StatusNotFound)
			return
		}
		if strings.TrimSpace(req.DisplayName) == "" {
			http.Error(w, "displayName is required", http.StatusBadRequest)
			return
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "machine.agent.install", protocol.MachineAgentInstallCommandPayload{
			AgentType:   req.AgentType,
			DisplayName: req.DisplayName,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.MachineAgentInstallCommandResult
		if err := transport.Decode(completed.Result, &result); err != nil {
			http.Error(w, "invalid machine.agent.install result", http.StatusBadGateway)
			return
		}

		if reg != nil {
			machine, ok := reg.Get(machineID)
			if !ok {
				machine = domain.Machine{
					ID:     machineID,
					Name:   machineID,
					Status: domain.MachineStatusOnline,
				}
			}
			machine.Agents = upsertMachineAgent(machine.Agents, result.Agent)
			reg.Upsert(machine)
		}

		writeJSON(w, http.StatusCreated, map[string]any{"agent": result.Agent})
	})

	mux.HandleFunc("DELETE /machines/{machineId}/agents/{agentId}", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID := strings.TrimSpace(r.PathValue("machineId"))
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		agentID := strings.TrimSpace(r.PathValue("agentId"))
		if agentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		if _, err := sender.SendCommand(r.Context(), machineID, "machine.agent.delete", protocol.MachineAgentDeleteCommandPayload{
			AgentID: agentID,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		if reg != nil {
			if machine, ok := reg.Get(machineID); ok {
				machine.Agents = removeMachineAgent(machine.Agents, agentID)
				reg.Upsert(machine)
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /machines/{machineId}/agents/{agentId}/config", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID := strings.TrimSpace(r.PathValue("machineId"))
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		agentID := strings.TrimSpace(r.PathValue("agentId"))
		if agentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "machine.agent.config.read", protocol.MachineAgentConfigReadCommandPayload{
			AgentID: agentID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.MachineAgentConfigReadCommandResult
		if err := transport.Decode(completed.Result, &result); err != nil {
			http.Error(w, "invalid machine.agent.config.read result", http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"document": result.Document})
	})

	mux.HandleFunc("PUT /machines/{machineId}/agents/{agentId}/config", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID := strings.TrimSpace(r.PathValue("machineId"))
		if machineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		agentID := strings.TrimSpace(r.PathValue("agentId"))
		if agentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}
		req, err := decodeConfigDocumentRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			http.Error(w, "content is required", http.StatusBadRequest)
			return
		}
		if err := validateTOMLContent(req.Content); err != nil {
			http.Error(w, "content must be valid toml", http.StatusBadRequest)
			return
		}

		agentType := domain.AgentTypeCodex
		if reg != nil {
			if machine, ok := reg.Get(machineID); ok {
				for _, agent := range machine.Agents {
					if agent.AgentID == agentID && agent.AgentType != "" {
						agentType = agent.AgentType
						break
					}
				}
			}
		}
		document := domain.AgentConfigDocument{
			AgentType: agentType,
			Format:    domain.AgentConfigFormatTOML,
			Content:   req.Content,
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "machine.agent.config.write", protocol.MachineAgentConfigWriteCommandPayload{
			AgentID:  agentID,
			Document: document,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.MachineAgentConfigWriteCommandResult
		if err := transport.Decode(completed.Result, &result); err != nil {
			http.Error(w, "invalid machine.agent.config.write result", http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"document": result.Document})
	})

	mux.HandleFunc("GET /machines", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": reg.List()})
	})

	mux.HandleFunc("GET /machines/{machineId}", func(w http.ResponseWriter, r *http.Request) {
		machineID := r.PathValue("machineId")
		if strings.TrimSpace(machineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		machine, ok := reg.Get(machineID)
		if !ok {
			http.Error(w, "machine not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"machine": machine})
	})

	mux.HandleFunc("POST /machines/{machineId}/runtime/start", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID := r.PathValue("machineId")
		if strings.TrimSpace(machineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		if _, err := sender.SendCommand(r.Context(), machineID, "runtime.start", protocol.RuntimeStartCommandPayload{}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"machineId": machineID})
	})

	mux.HandleFunc("POST /machines/{machineId}/runtime/stop", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		machineID := r.PathValue("machineId")
		if strings.TrimSpace(machineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}

		if _, err := sender.SendCommand(r.Context(), machineID, "runtime.stop", protocol.RuntimeStopCommandPayload{}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"machineId": machineID})
	})

	mux.HandleFunc("GET /threads", func(w http.ResponseWriter, _ *http.Request) {
		overrides := resolveThreadTitleOverrides(settingsStore)
		threads := make([]domain.Thread, 0)
		for _, thread := range idx.Threads() {
			if isDeleted(thread.ThreadID) {
				continue
			}
			threads = append(threads, applyThreadTitleOverride(thread, overrides))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": threads})
	})

	mux.HandleFunc("GET /threads/{threadId}", func(w http.ResponseWriter, r *http.Request) {
		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}
		if isDeleted(threadID) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}

		overrides := resolveThreadTitleOverrides(settingsStore)
		pendingApprovals := []protocol.ApprovalRequiredPayload{}
		if reg != nil {
			pendingApprovals = reg.PendingApprovalsForThread(threadID)
		}
		activeTurnID := resolveActiveTurnID(sender, threadID)

		if sender != nil && router != nil {
			if route, ok := router.ResolveThread(threadID); ok {
				liveReadRequired := machineIsOnline(reg, route.MachineID)
				completed, err := sender.SendCommand(r.Context(), route.MachineID, "thread.read", protocol.ThreadReadCommandPayload{
					ThreadID: threadID,
					AgentID:  route.AgentID,
				})
				if err == nil {
					var result protocol.ThreadReadCommandResult
					if err := json.Unmarshal(completed.Result, &result); err == nil {
						if result.Thread.MachineID == "" {
							result.Thread.MachineID = route.MachineID
						}
						if result.Thread.AgentID == "" {
							result.Thread.AgentID = route.AgentID
						}
						result.Thread = applyThreadTitleOverride(result.Thread, overrides)
						if strings.TrimSpace(result.Thread.ThreadID) != "" {
							router.TrackThread(result.Thread.ThreadID, result.Thread.MachineID, result.Thread.AgentID)
							if idx != nil {
								idx.UpsertThread(result.Thread.MachineID, result.Thread)
							}
						}
						clearDeleted(result.Thread.ThreadID)
						writeJSON(w, http.StatusOK, threadDetailResponse{
							Thread:           result.Thread,
							ActiveTurnID:     activeTurnID,
							PendingApprovals: pendingApprovals,
						})
						return
					}
					if liveReadRequired {
						http.Error(w, "invalid thread.read result", http.StatusBadGateway)
						return
					}
				}
				if liveReadRequired {
					http.Error(w, err.Error(), http.StatusBadGateway)
					return
				}
			}
		}

		thread, ok := findThread(idx, threadID)
		if !ok {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		thread = applyThreadTitleOverride(thread, overrides)
		writeJSON(w, http.StatusOK, threadDetailResponse{
			Thread:           thread,
			ActiveTurnID:     activeTurnID,
			PendingApprovals: pendingApprovals,
		})
	})

	mux.HandleFunc("POST /threads", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		var req createThreadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.MachineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.AgentID) == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		completed, err := sender.SendCommand(r.Context(), req.MachineID, "thread.create", protocol.ThreadCreateCommandPayload{
			AgentID: req.AgentID,
			Title:   req.Title,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.ThreadCreateCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid thread.create result", http.StatusBadGateway)
			return
		}
		if result.Thread.MachineID == "" {
			result.Thread.MachineID = req.MachineID
		}
		if result.Thread.AgentID == "" {
			result.Thread.AgentID = req.AgentID
		}
		result.Thread = applyThreadTitleOverride(result.Thread, resolveThreadTitleOverrides(settingsStore))
		if router != nil && strings.TrimSpace(result.Thread.ThreadID) != "" {
			router.TrackThread(result.Thread.ThreadID, result.Thread.MachineID, result.Thread.AgentID)
		}
		if idx != nil && strings.TrimSpace(result.Thread.ThreadID) != "" {
			idx.UpsertThread(result.Thread.MachineID, result.Thread)
		}
		clearDeleted(result.Thread.ThreadID)

		writeJSON(w, http.StatusCreated, map[string]any{"thread": result.Thread})
	})

	mux.HandleFunc("PATCH /threads/{threadId}", func(w http.ResponseWriter, r *http.Request) {
		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}
		if isDeleted(threadID) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}

		var req threadRenameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		nextTitle := strings.TrimSpace(req.Title)
		if nextTitle == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		thread, ok := findThread(idx, threadID)
		if !ok {
			if route, ok := resolveThreadRoute(router, idx, threadID); ok {
				thread = domain.Thread{
					ThreadID:  threadID,
					MachineID: route.MachineID,
					AgentID:   route.AgentID,
					Status:    domain.ThreadStatusUnknown,
				}
			} else {
				http.Error(w, "thread not found", http.StatusNotFound)
				return
			}
		}

		preferences, ok, err := settingsStore.GetConsolePreferences()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !ok {
			preferences = domain.ConsolePreferences{}
		}
		if preferences.ThreadTitles == nil {
			preferences.ThreadTitles = map[string]string{}
		}
		preferences.ThreadTitles[threadID] = nextTitle
		if err := settingsStore.PutConsolePreferences(preferences); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		thread.Title = nextTitle
		if idx != nil && strings.TrimSpace(thread.MachineID) != "" {
			idx.UpsertThread(thread.MachineID, thread)
		}
		clearDeleted(threadID)

		if emitter, ok := sender.(threadUpdateEmitter); ok {
			emitter.EmitThreadUpdated(protocol.ThreadUpdatedPayload{
				MachineID: thread.MachineID,
				AgentID:   thread.AgentID,
				ThreadID:  thread.ThreadID,
				Thread:    &thread,
			}, time.Now().UTC().Format(time.RFC3339))
		}

		writeJSON(w, http.StatusOK, map[string]any{"thread": thread})
	})

	mux.HandleFunc("POST /threads/{threadId}/resume", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}

		route, ok := resolveThreadRoute(router, idx, threadID)
		if !ok {
			http.Error(w, "thread route not found", http.StatusNotFound)
			return
		}

		completed, err := sender.SendCommand(r.Context(), route.MachineID, "thread.resume", protocol.ThreadResumeCommandPayload{
			ThreadID: threadID,
			AgentID:  route.AgentID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.ThreadResumeCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid thread.resume result", http.StatusBadGateway)
			return
		}
		if result.Thread.MachineID == "" {
			result.Thread.MachineID = route.MachineID
		}
		if result.Thread.AgentID == "" {
			result.Thread.AgentID = route.AgentID
		}
		result.Thread = applyThreadTitleOverride(result.Thread, resolveThreadTitleOverrides(settingsStore))
		if router != nil && strings.TrimSpace(result.Thread.ThreadID) != "" {
			router.TrackThread(result.Thread.ThreadID, result.Thread.MachineID, result.Thread.AgentID)
		}
		if idx != nil && strings.TrimSpace(result.Thread.ThreadID) != "" {
			idx.UpsertThread(result.Thread.MachineID, result.Thread)
		}
		clearDeleted(result.Thread.ThreadID)

		writeJSON(w, http.StatusOK, map[string]any{"thread": result.Thread})
	})

	mux.HandleFunc("POST /threads/{threadId}/archive", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}

		route, ok := resolveThreadRoute(router, idx, threadID)
		if !ok {
			http.Error(w, "thread route not found", http.StatusNotFound)
			return
		}

		completed, err := sender.SendCommand(r.Context(), route.MachineID, "thread.archive", protocol.ThreadArchiveCommandPayload{
			ThreadID: threadID,
			AgentID:  route.AgentID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.ThreadArchiveCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid thread.archive result", http.StatusBadGateway)
			return
		}
		if result.ThreadID == "" {
			result.ThreadID = threadID
		}

		writeJSON(w, http.StatusAccepted, map[string]any{"threadId": result.ThreadID})
	})

	mux.HandleFunc("DELETE /threads/{threadId}", func(w http.ResponseWriter, r *http.Request) {
		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}
		if isDeleted(threadID) {
			writeJSON(w, http.StatusOK, threadDeleteResponse{
				ThreadID: threadID,
				Deleted:  true,
				Archived: false,
			})
			return
		}
		thread, ok := findThread(idx, threadID)
		if !ok {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}

		archived := false
		if sender != nil {
			if route, ok := resolveThreadRoute(router, idx, threadID); ok {
				completed, err := sender.SendCommand(r.Context(), route.MachineID, "thread.archive", protocol.ThreadArchiveCommandPayload{
					ThreadID: threadID,
					AgentID:  route.AgentID,
				})
				if err == nil {
					var result protocol.ThreadArchiveCommandResult
					if err := json.Unmarshal(completed.Result, &result); err == nil {
						archived = true
					}
				}
			}
		}

		markDeleted(threadID)
		if emitter, ok := sender.(threadUpdateEmitter); ok {
			route := domain.ThreadRoute{
				MachineID: thread.MachineID,
				AgentID:   thread.AgentID,
			}
			if route.MachineID == "" {
				route, _ = resolveThreadRoute(router, idx, threadID)
			}
			emitter.EmitThreadUpdated(protocol.ThreadUpdatedPayload{
				MachineID: route.MachineID,
				AgentID:   route.AgentID,
				ThreadID:  threadID,
			}, time.Now().UTC().Format(time.RFC3339))
		}
		writeJSON(w, http.StatusOK, threadDeleteResponse{
			ThreadID: threadID,
			Deleted:  true,
			Archived: archived,
		})
	})

	mux.HandleFunc("POST /threads/{threadId}/turns", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		threadID := r.PathValue("threadId")
		if strings.TrimSpace(threadID) == "" {
			http.Error(w, "threadId is required", http.StatusBadRequest)
			return
		}

		route, ok := resolveThreadRoute(router, idx, threadID)
		if !ok {
			http.Error(w, "thread route not found", http.StatusNotFound)
			return
		}

		var req startTurnRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		completed, err := sender.SendCommand(r.Context(), route.MachineID, "turn.start", protocol.TurnStartCommandPayload{
			ThreadID: threadID,
			AgentID:  route.AgentID,
			Input:    req.Input,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.TurnStartCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid turn.start result", http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]any{"turn": result})
	})

	mux.HandleFunc("POST /threads/{threadId}/turns/{turnId}/steer", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		threadID := r.PathValue("threadId")
		turnID := r.PathValue("turnId")
		if strings.TrimSpace(threadID) == "" || strings.TrimSpace(turnID) == "" {
			http.Error(w, "threadId and turnId are required", http.StatusBadRequest)
			return
		}

		route, ok := resolveThreadRoute(router, idx, threadID)
		if !ok {
			http.Error(w, "thread route not found", http.StatusNotFound)
			return
		}

		var req startTurnRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		completed, err := sender.SendCommand(r.Context(), route.MachineID, "turn.steer", protocol.TurnSteerCommandPayload{
			ThreadID: threadID,
			TurnID:   turnID,
			AgentID:  route.AgentID,
			Input:    req.Input,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.TurnSteerCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid turn.steer result", http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]any{"turn": result})
	})

	mux.HandleFunc("POST /threads/{threadId}/turns/{turnId}/interrupt", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		threadID := r.PathValue("threadId")
		turnID := r.PathValue("turnId")
		if strings.TrimSpace(threadID) == "" || strings.TrimSpace(turnID) == "" {
			http.Error(w, "threadId and turnId are required", http.StatusBadRequest)
			return
		}

		route, ok := resolveThreadRoute(router, idx, threadID)
		if !ok {
			http.Error(w, "thread route not found", http.StatusNotFound)
			return
		}

		completed, err := sender.SendCommand(r.Context(), route.MachineID, "turn.interrupt", protocol.TurnInterruptCommandPayload{
			ThreadID: threadID,
			TurnID:   turnID,
			AgentID:  route.AgentID,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.TurnInterruptCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid turn.interrupt result", http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"turn": result.Turn})
	})

	mux.HandleFunc("POST /approvals/{requestId}/respond", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		resolver, ok := sender.(approvalRequestResolver)
		if !ok {
			http.Error(w, "approval resolver unavailable", http.StatusServiceUnavailable)
			return
		}

		requestID := r.PathValue("requestId")
		if strings.TrimSpace(requestID) == "" {
			http.Error(w, "requestId is required", http.StatusBadRequest)
			return
		}

		var req approvalRespondRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Decision) == "" {
			http.Error(w, "decision is required", http.StatusBadRequest)
			return
		}

		machineID, ok := resolver.ResolveApprovalMachine(requestID)
		if !ok {
			http.Error(w, "approval route not found", http.StatusNotFound)
			return
		}

		storedApproval := protocol.ApprovalRequiredPayload{}
		if reg != nil {
			if payload, ok := reg.PendingApproval(requestID); ok {
				storedApproval = payload
			}
		}

		completed, err := sender.SendCommand(r.Context(), machineID, "approval.respond", protocol.ApprovalRespondCommandPayload{
			RequestID: requestID,
			ThreadID:  storedApproval.ThreadID,
			TurnID:    storedApproval.TurnID,
			Decision:  req.Decision,
			Answers:   req.Answers,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if reg != nil {
			reg.RemovePendingApproval(requestID)
		}
		if cleaner, ok := sender.(approvalRequestCleaner); ok {
			cleaner.ClearApprovalRequest(requestID)
		}

		var result protocol.ApprovalRespondCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid approval.respond result", http.StatusBadGateway)
			return
		}
		if result.RequestID == "" {
			result.RequestID = requestID
		}
		if result.Decision == "" {
			result.Decision = req.Decision
		}

		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("GET /environment/skills", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": idx.Environment(domain.EnvironmentKindSkill)})
	})

	mux.HandleFunc("GET /environment/mcps", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": idx.Environment(domain.EnvironmentKindMCP)})
	})

	mux.HandleFunc("GET /environment/plugins", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"items": idx.Environment(domain.EnvironmentKindPlugin)})
	})

	mux.HandleFunc("POST /environment/skills", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}
		if r.Body == nil {
			http.Error(w, "skill payload is required", http.StatusBadRequest)
			return
		}

		var req environmentSkillCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			if err == io.EOF {
				http.Error(w, "skill payload is required", http.StatusBadRequest)
				return
			}
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.MachineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.AgentID) == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		completed, err := sender.SendCommand(r.Context(), req.MachineID, "environment.skill.create", protocol.EnvironmentSkillCreateCommandPayload{
			AgentID:     req.AgentID,
			Name:        req.Name,
			Description: req.Description,
		})
		if err != nil {
			if strings.Contains(err.Error(), "skill scaffold already exists") {
				http.Error(w, "skill scaffold already exists", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		var result protocol.EnvironmentSkillCreateCommandResult
		if err := json.Unmarshal(completed.Result, &result); err != nil {
			http.Error(w, "invalid skill create result", http.StatusBadGateway)
			return
		}
		if result.SkillID == "" {
			result.SkillID = req.Name
		}

		writeJSON(w, http.StatusOK, map[string]any{"skillId": result.SkillID})
	})

	mux.HandleFunc("POST /environment/skills/{id}/enable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMutationRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if req.MachineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if req.AgentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		skillID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindSkill, req.MachineID, req.AgentID, skillID)
		if !ok {
			http.Error(w, "skill not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.skill.enable", protocol.EnvironmentSkillSetEnabledCommandPayload{
			SkillID: skillID,
			AgentID: resource.AgentID,
			Enabled: true,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"skillId": skillID, "enabled": true})
	})

	mux.HandleFunc("POST /environment/skills/{id}/disable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMutationRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if req.MachineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if req.AgentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		skillID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindSkill, req.MachineID, req.AgentID, skillID)
		if !ok {
			http.Error(w, "skill not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.skill.disable", protocol.EnvironmentSkillSetEnabledCommandPayload{
			SkillID: skillID,
			AgentID: resource.AgentID,
			Enabled: false,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"skillId": skillID, "enabled": false})
	})

	mux.HandleFunc("DELETE /environment/skills/{id}", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMutationRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if req.MachineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if req.AgentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		skillID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindSkill, req.MachineID, req.AgentID, skillID)
		if !ok {
			http.Error(w, "skill not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.skill.delete", protocol.EnvironmentSkillDeleteCommandPayload{
			SkillID: skillID,
			AgentID: resource.AgentID,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"skillId": skillID})
	})

	mux.HandleFunc("DELETE /environment/plugins/{id}", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMutationRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if req.MachineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if req.AgentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		pluginID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindPlugin, req.MachineID, req.AgentID, pluginID)
		if !ok {
			http.Error(w, "plugin not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.plugin.uninstall", protocol.EnvironmentPluginUninstallCommandPayload{
			PluginID: pluginID,
			AgentID:  resource.AgentID,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"pluginId": pluginID})
	})

	mux.HandleFunc("POST /environment/mcps", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMCPUpsertRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.MachineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.AgentID) == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.ResourceID) == "" {
			http.Error(w, "resourceId is required", http.StatusBadRequest)
			return
		}

		if _, err := sender.SendCommand(r.Context(), req.MachineID, "environment.mcp.upsert", protocol.EnvironmentMCPUpsertCommandPayload{
			ServerID: req.ResourceID,
			AgentID:  req.AgentID,
			Config:   req.Config,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"resourceId": req.ResourceID})
	})

	mux.HandleFunc("POST /environment/mcps/{id}/enable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMutationRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if req.MachineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if req.AgentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		serverID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindMCP, req.MachineID, req.AgentID, serverID)
		if !ok {
			http.Error(w, "mcp not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.mcp.enable", protocol.EnvironmentMCPSetEnabledCommandPayload{
			ServerID: serverID,
			AgentID:  resource.AgentID,
			Enabled:  true,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"resourceId": serverID, "enabled": true})
	})

	mux.HandleFunc("POST /environment/mcps/{id}/disable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMutationRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if req.MachineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if req.AgentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		serverID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindMCP, req.MachineID, req.AgentID, serverID)
		if !ok {
			http.Error(w, "mcp not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.mcp.disable", protocol.EnvironmentMCPSetEnabledCommandPayload{
			ServerID: serverID,
			AgentID:  resource.AgentID,
			Enabled:  false,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"resourceId": serverID, "enabled": false})
	})

	mux.HandleFunc("DELETE /environment/mcps/{id}", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMutationRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if req.MachineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if req.AgentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		serverID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindMCP, req.MachineID, req.AgentID, serverID)
		if !ok {
			http.Error(w, "mcp not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.mcp.remove", protocol.EnvironmentMCPRemoveCommandPayload{
			ServerID: serverID,
			AgentID:  resource.AgentID,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"resourceId": serverID})
	})

	mux.HandleFunc("POST /environment/plugins/{id}/install", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentPluginInstallRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.MachineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.AgentID) == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		pluginID := r.PathValue("id")
		if strings.TrimSpace(req.PluginID) != "" && req.PluginID != pluginID {
			http.Error(w, "pluginId does not match path", http.StatusBadRequest)
			return
		}
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindPlugin, req.MachineID, req.AgentID, pluginID)
		if !ok {
			http.Error(w, "plugin not found", http.StatusNotFound)
			return
		}
		pluginInstallID := pluginID
		marketplacePath := strings.TrimSpace(req.MarketplacePath)
		pluginName := strings.TrimSpace(req.PluginName)
		allowedMarketplaces := allowedMarketplacePaths(idx, req.MachineID, req.AgentID)
		if marketplacePath != "" {
			if _, ok := allowedMarketplaces[marketplacePath]; !ok {
				http.Error(w, "marketplacePath is not recognized", http.StatusBadRequest)
				return
			}
			resourceMarketplace := stringDetail(resource, "marketplacePath")
			if resourceMarketplace != "" && resourceMarketplace != marketplacePath {
				http.Error(w, "marketplacePath does not match plugin", http.StatusBadRequest)
				return
			}
		}
		if marketplacePath == "" {
			marketplacePath = stringDetail(resource, "marketplacePath")
		}
		if pluginName == "" {
			pluginName = stringDetail(resource, "pluginName")
		}
		if marketplacePath != "" {
			if _, ok := allowedMarketplaces[marketplacePath]; !ok {
				http.Error(w, "marketplacePath is not recognized", http.StatusBadRequest)
				return
			}
		}
		if marketplacePath == "" || pluginName == "" {
			http.Error(w, "plugin install details unavailable", http.StatusConflict)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.plugin.install", protocol.EnvironmentPluginInstallCommandPayload{
			PluginID:        pluginInstallID,
			AgentID:         resource.AgentID,
			MarketplacePath: marketplacePath,
			PluginName:      pluginName,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"pluginId": pluginInstallID})
	})

	mux.HandleFunc("POST /environment/plugins/install", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentPluginInstallRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.MachineID) == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.AgentID) == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.PluginName) == "" {
			http.Error(w, "pluginName is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.MarketplacePath) == "" {
			http.Error(w, "marketplacePath is required", http.StatusBadRequest)
			return
		}

		allowedMarketplaces := allowedMarketplacePaths(idx, req.MachineID, req.AgentID)
		if _, ok := allowedMarketplaces[strings.TrimSpace(req.MarketplacePath)]; !ok {
			http.Error(w, "marketplacePath is not recognized", http.StatusBadRequest)
			return
		}

		pluginID := strings.TrimSpace(req.PluginID)
		if pluginID == "" {
			pluginID = strings.TrimSpace(req.PluginName)
		}

		if _, err := sender.SendCommand(r.Context(), req.MachineID, "environment.plugin.install", protocol.EnvironmentPluginInstallCommandPayload{
			PluginID:        pluginID,
			AgentID:         req.AgentID,
			MarketplacePath: strings.TrimSpace(req.MarketplacePath),
			PluginName:      strings.TrimSpace(req.PluginName),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"pluginId": pluginID})
	})

	mux.HandleFunc("POST /environment/plugins/{id}/enable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMutationRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if req.MachineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if req.AgentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		pluginID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindPlugin, req.MachineID, req.AgentID, pluginID)
		if !ok {
			http.Error(w, "plugin not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.plugin.enable", protocol.EnvironmentPluginSetEnabledCommandPayload{
			PluginID: pluginID,
			AgentID:  resource.AgentID,
			Enabled:  true,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"pluginId": pluginID, "enabled": true})
	})

	mux.HandleFunc("POST /environment/plugins/{id}/disable", func(w http.ResponseWriter, r *http.Request) {
		if sender == nil {
			http.Error(w, "command sender unavailable", http.StatusServiceUnavailable)
			return
		}

		req, err := decodeEnvironmentMutationRequest(r)
		if err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		if req.MachineID == "" {
			http.Error(w, "machineId is required", http.StatusBadRequest)
			return
		}
		if req.AgentID == "" {
			http.Error(w, "agentId is required", http.StatusBadRequest)
			return
		}

		pluginID := r.PathValue("id")
		resource, ok := findEnvironmentResource(idx, domain.EnvironmentKindPlugin, req.MachineID, req.AgentID, pluginID)
		if !ok {
			http.Error(w, "plugin not found", http.StatusNotFound)
			return
		}

		if _, err := sender.SendCommand(r.Context(), resource.MachineID, "environment.plugin.disable", protocol.EnvironmentPluginSetEnabledCommandPayload{
			PluginID: pluginID,
			AgentID:  resource.AgentID,
			Enabled:  false,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"pluginId": pluginID, "enabled": false})
	})

	if strings.TrimSpace(apiKey) == "" && !failClosedOnBlankKey {
		return mux
	}
	return requireConsoleAuth(apiKey, mux)
}

func requireConsoleAuth(apiKey string, next http.Handler) http.Handler {
	expected := strings.TrimSpace(apiKey)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health", "/ws/client":
			// Intentionally open: this endpoint is for local client bridge connections and
			// relies on loopback-by-default host posture.
			next.ServeHTTP(w, r)
			return
		case "/ws":
			if !validateWSAPIKey(r, expected) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
			return
		default:
			if !validateBearerToken(r, expected) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		}
	})
}

func requireConsoleAuthFunc(apiKey string, next http.HandlerFunc) http.HandlerFunc {
	wrapped := requireConsoleAuth(apiKey, next)
	return wrapped.ServeHTTP
}

func validateBearerToken(r *http.Request, expected string) bool {
	if r == nil || strings.TrimSpace(expected) == "" {
		return false
	}
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.Fields(authorization)
	if len(parts) != 2 {
		return false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return false
	}
	return strings.TrimSpace(parts[1]) == strings.TrimSpace(expected)
}

func validateWSAPIKey(r *http.Request, expected string) bool {
	if r == nil || strings.TrimSpace(expected) == "" {
		return false
	}
	return strings.TrimSpace(r.URL.Query().Get("apiKey")) == strings.TrimSpace(expected)
}
