package app

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tcp404/OneTiny/internal/accesslog"
	"github.com/tcp404/OneTiny/internal/config"
	"github.com/tcp404/OneTiny/internal/runtime"
	"github.com/tcp404/OneTiny/internal/security"
	"github.com/tcp404/OneTiny/internal/server"
	"github.com/tcp404/OneTiny/internal/updater"
)

type Dependencies struct {
	ConfigStore       *config.Store
	Runtime           *runtime.Runtime
	Manager           *server.Manager
	Logger            *accesslog.Logger
	Updater           updateBackend
	CurrentExecutable func() (string, error)
	StartUpdateHelper func(helperPath string, plan updater.InstallPlan) error
}

type Service struct {
	mu                  sync.Mutex
	configStore         *config.Store
	runtime             *runtime.Runtime
	manager             *server.Manager
	logger              *accesslog.Logger
	lastErr             string
	portRestartRequired bool
	pendingPort         *int
	updater             updateBackend
	currentExecutable   func() (string, error)
	startUpdateHelper   func(helperPath string, plan updater.InstallPlan) error
	updateStatus        UpdateStatusDTO
	updateCheck         updater.CheckResult
	hasUpdateCheck      bool
	updateDownload      updater.DownloadResult
	updateStage         updater.StageResult
}

func NewService(deps Dependencies) *Service {
	logger := deps.Logger
	if logger == nil {
		logger = accesslog.New(accesslog.DefaultPath())
	}
	manager := deps.Manager
	if manager == nil && deps.Runtime != nil {
		manager = server.NewManagerWithDependencies(server.Dependencies{
			Runtime:   deps.Runtime,
			AccessLog: logger,
		})
	}
	updateBackend := deps.Updater
	if updateBackend == nil {
		updateBackend = updater.Service{}
	}
	currentExecutable := deps.CurrentExecutable
	if currentExecutable == nil {
		currentExecutable = os.Executable
	}
	startUpdateHelper := deps.StartUpdateHelper
	if startUpdateHelper == nil {
		startUpdateHelper = updater.StartHelper
	}
	return &Service{
		configStore:       deps.ConfigStore,
		runtime:           deps.Runtime,
		manager:           manager,
		logger:            logger,
		updater:           updateBackend,
		currentExecutable: currentExecutable,
		startUpdateHelper: startUpdateHelper,
	}
}

func (s *Service) GetStatus() (StatusDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusLocked(), nil
}

func (s *Service) StartSharing() (StatusDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.configStore.ValidateSecureConfigFor(s.runtime.Snapshot().IsSecure); err != nil {
		s.lastErr = err.Error()
		return s.statusLocked(), err
	}
	if err := s.manager.Start(); err != nil {
		s.lastErr = err.Error()
		return s.statusLocked(), err
	}
	s.lastErr = ""
	return s.statusLocked(), nil
}

func (s *Service) StopSharing() (StatusDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.manager.Stop(); err != nil && !errors.Is(err, server.ErrServerNotRunning) {
		s.lastErr = err.Error()
		return s.statusLocked(), err
	}
	if s.portRestartRequired || s.pendingPort != nil {
		port := s.configStore.Current().Port
		if err := s.manager.ApplyRuntime(runtime.Patch{Port: &port}); err != nil {
			s.lastErr = err.Error()
			return s.statusLocked(), err
		}
		s.pendingPort = nil
		s.portRestartRequired = false
	}
	s.lastErr = ""
	return s.statusLocked(), nil
}

func (s *Service) UpdateConfig(patch ConfigPatchDTO) (StatusDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	active := s.runtime.Snapshot()
	running := s.manager.Running()
	targetPort, hasPortTarget := s.portTargetLocked(patch, active)
	confirmPort := running && patch.RestartPort && hasPortTarget
	portChanged := hasPortTarget && targetPort != active.Port

	persistPatch := patch
	if confirmPort {
		persistPatch.Port = &targetPort
	}
	if confirmPort && portChanged {
		previousConfig := s.configStore.Current()
		savedSnapshot, err := s.persistConfigPatch(persistPatch, runtime.ProcessFromSnapshot(active))
		if err != nil {
			s.pendingPort = nil
			s.portRestartRequired = true
			s.lastErr = err.Error()
			return s.statusLocked(), err
		}
		if err := s.manager.RestartWithSnapshot(savedSnapshot, nil); err != nil {
			if rollbackErr := s.restoreConfigLocked(previousConfig); rollbackErr != nil {
				s.lastErr = fmt.Sprintf("%s; 回滚配置失败: %s", err.Error(), rollbackErr.Error())
				s.pendingPort = nil
				s.portRestartRequired = true
				return s.statusLocked(), err
			}
			s.pendingPort = nil
			s.portRestartRequired = true
			s.lastErr = err.Error()
			return s.statusLocked(), err
		}
		if err := s.manager.ApplyRuntime(runtimePatchFromSnapshot(active, savedSnapshot)); err != nil {
			s.lastErr = err.Error()
			return s.statusLocked(), err
		}
		s.pendingPort = nil
		s.portRestartRequired = false
		s.lastErr = ""
		return s.statusLocked(), nil
	}

	savedSnapshot, err := s.persistConfigPatch(persistPatch, runtime.ProcessFromSnapshot(active))
	if err != nil {
		s.lastErr = err.Error()
		return s.statusLocked(), err
	}

	runtimePatch := runtimePatchFromSnapshot(active, savedSnapshot)
	if running && hasPortTarget && !confirmPort {
		runtimePatch.Port = nil
	}
	if err := s.manager.ApplyRuntime(runtimePatch); err != nil {
		s.lastErr = err.Error()
		return s.statusLocked(), err
	}

	if running && hasPortTarget && portChanged {
		s.pendingPort = new(targetPort)
		s.portRestartRequired = true
		s.lastErr = ErrPortRestartRequiresConfirm.Error()
		return s.statusLocked(), nil
	}
	if hasPortTarget && (!running || !portChanged) {
		s.pendingPort = nil
		s.portRestartRequired = false
	}
	if confirmPort {
		s.pendingPort = nil
		s.portRestartRequired = false
	}

	s.lastErr = ""
	return s.statusLocked(), nil
}

func (s *Service) SetCredentials(patch CredentialPatchDTO) (StatusDTO, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username := strings.TrimSpace(patch.Username)
	if username == "" {
		s.lastErr = ErrUsernameRequired.Error()
		return s.statusLocked(), ErrUsernameRequired
	}
	if strings.TrimSpace(patch.Password) == "" {
		s.lastErr = ErrPasswordRequired.Error()
		return s.statusLocked(), ErrPasswordRequired
	}
	if patch.Password != patch.ConfirmPassword {
		s.lastErr = ErrPasswordConfirmationMismatch.Error()
		return s.statusLocked(), ErrPasswordConfirmationMismatch
	}
	hash, err := security.HashPassword(patch.Password)
	if err != nil {
		s.lastErr = err.Error()
		return s.statusLocked(), err
	}
	credentials := security.CredentialConfig{
		Username:     username,
		PasswordHash: hash,
		HashAlgo:     security.HashAlgoBcrypt,
	}
	if err := credentials.ValidateForSecureMode(); err != nil {
		s.lastErr = err.Error()
		return s.statusLocked(), err
	}

	securityPatch := config.SecurityPatch{
		Username:     &username,
		PasswordHash: &hash,
	}
	if patch.EnableSecure {
		securityPatch.IsSecure = &patch.EnableSecure
	}
	savedConfig, err := s.configStore.PatchSecurity(securityPatch)
	if err != nil {
		s.lastErr = err.Error()
		return s.statusLocked(), err
	}
	active := s.runtime.Snapshot()
	savedSnapshot := runtime.SnapshotFromConfig(runtimeConfigFromConfig(savedConfig), runtime.ProcessFromSnapshot(active))
	if err := s.manager.ApplyRuntime(runtime.Patch{
		Username:     &savedSnapshot.Username,
		PasswordHash: &savedSnapshot.PasswordHash,
		IsSecure:     &savedSnapshot.IsSecure,
	}); err != nil {
		s.lastErr = err.Error()
		return s.statusLocked(), err
	}

	s.lastErr = ""
	return s.statusLocked(), nil
}

func (s *Service) GetLogs(filter LogFilterDTO) ([]LogEntryDTO, error) {
	events, err := s.logger.Read(accesslog.Filter{
		Event: filter.Event,
		Since: timeValue(filter.Since),
		Until: timeValue(filter.Until),
	})
	if err != nil {
		return nil, err
	}
	return logEntriesFromEvents(events), nil
}

func (s *Service) ClearLogs() error {
	return s.logger.Clear()
}

func (s *Service) ExportLogs(path string, filter LogFilterDTO) error {
	if strings.TrimSpace(path) == "" {
		return ErrInvalidExportPath
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return ErrInvalidExportPath
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	entries, err := s.GetLogs(filter)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.Write([]string{"time", "client_ip", "method", "event", "path", "status", "result"}); err != nil {
		return err
	}
	for _, entry := range entries {
		if err := writer.Write([]string{
			entry.Time.Format(time.RFC3339),
			entry.ClientIP,
			entry.Method,
			entry.Event,
			entry.Path,
			strconv.Itoa(entry.Status),
			entry.Result,
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func (s *Service) statusLocked() StatusDTO {
	snapshot := s.runtime.Snapshot()
	running := s.manager.Running()
	state := "未运行"
	if running {
		state = "运行中"
	}

	return StatusDTO{
		Running:             running,
		StateLabel:          state,
		Address:             addressFromSnapshot(snapshot, running),
		Config:              s.configDTOLocked(snapshot),
		HasCredentials:      hasCredentials(snapshot),
		ConfigPath:          s.configStore.Path(),
		AccessLogPath:       accesslog.DefaultPath(),
		PortRestartRequired: s.portRestartRequired,
		LastError:           s.lastErr,
	}
}

func (s *Service) portTargetLocked(patch ConfigPatchDTO, active runtime.Snapshot) (int, bool) {
	if patch.RestartPort && s.pendingPort != nil {
		return *s.pendingPort, true
	}
	if savedPort := s.configStore.Current().Port; patch.RestartPort && s.portRestartRequired && savedPort != active.Port {
		return savedPort, true
	}
	if patch.Port != nil {
		return *patch.Port, true
	}
	return 0, false
}

func (s *Service) configDTOLocked(snapshot runtime.Snapshot) ConfigDTO {
	dto := configDTOFromSnapshot(snapshot)
	if s.pendingPort != nil {
		dto.Port = *s.pendingPort
	}
	return dto
}

func (s *Service) restoreConfigLocked(cfg config.Config) error {
	_, err := s.configStore.Patch(config.ConfigPatch{
		RootPath:           &cfg.RootPath,
		Port:               &cfg.Port,
		MaxLevel:           &cfg.MaxLevel,
		IsAllowUpload:      &cfg.IsAllowUpload,
		IsSecure:           &cfg.IsSecure,
		ScratchMaxItems:    &cfg.ScratchMaxItems,
		ScratchMaxItemSize: &cfg.ScratchMaxItemSize,
	})
	return err
}

func (s *Service) persistConfigPatch(patch ConfigPatchDTO, process runtime.Process) (runtime.Snapshot, error) {
	cfg, err := s.configStore.Patch(configPatchFromDTO(patch))
	if err != nil {
		return runtime.Snapshot{}, err
	}
	return runtime.SnapshotFromConfig(runtimeConfigFromConfig(cfg), process), nil
}

func runtimeConfigFromConfig(cfg config.Config) runtime.PersistentConfig {
	sizeBytes, _ := config.ParseByteSize(cfg.ScratchMaxItemSize)
	return runtime.PersistentConfig{
		RootPath:            cfg.RootPath,
		Port:                cfg.Port,
		MaxLevel:            cfg.MaxLevel,
		IsAllowUpload:       cfg.IsAllowUpload,
		IsSecure:            cfg.IsSecure,
		Username:            cfg.Username,
		PasswordHash:        cfg.PasswordHash,
		ScratchMaxItems:     cfg.ScratchMaxItems,
		ScratchMaxItemSize:  cfg.ScratchMaxItemSize,
		ScratchMaxItemBytes: sizeBytes,
	}
}

func configPatchFromDTO(patch ConfigPatchDTO) config.ConfigPatch {
	return config.ConfigPatch{
		RootPath:           patch.RootPath,
		Port:               patch.Port,
		MaxLevel:           patch.MaxLevel,
		IsAllowUpload:      patch.IsAllowUpload,
		IsSecure:           patch.IsSecure,
		ScratchMaxItems:    patch.ScratchMaxItems,
		ScratchMaxItemSize: patch.ScratchMaxItemSize,
	}
}

func runtimePatchFromSnapshot(old, next runtime.Snapshot) runtime.Patch {
	patch := runtime.Patch{}
	if old.RootPath != next.RootPath {
		patch.RootPath = &next.RootPath
	}
	if old.Port != next.Port {
		patch.Port = &next.Port
	}
	if old.MaxLevel != next.MaxLevel {
		patch.MaxLevel = &next.MaxLevel
	}
	if old.IsAllowUpload != next.IsAllowUpload {
		patch.IsAllowUpload = &next.IsAllowUpload
	}
	if old.IsSecure != next.IsSecure {
		patch.IsSecure = &next.IsSecure
	}
	if old.ScratchMaxItems != next.ScratchMaxItems {
		patch.ScratchMaxItems = &next.ScratchMaxItems
	}
	if old.ScratchMaxItemSize != next.ScratchMaxItemSize {
		patch.ScratchMaxItemSize = &next.ScratchMaxItemSize
	}
	if old.ScratchMaxItemBytes != next.ScratchMaxItemBytes {
		patch.ScratchMaxItemBytes = &next.ScratchMaxItemBytes
	}
	return patch
}

func snapshotWithPatch(snapshot runtime.Snapshot, patch ConfigPatchDTO) runtime.Snapshot {
	if patch.RootPath != nil {
		snapshot.RootPath = *patch.RootPath
	}
	if patch.Port != nil {
		snapshot.Port = *patch.Port
	}
	if patch.MaxLevel != nil {
		snapshot.MaxLevel = *patch.MaxLevel
	}
	if patch.IsAllowUpload != nil {
		snapshot.IsAllowUpload = *patch.IsAllowUpload
	}
	if patch.IsSecure != nil {
		snapshot.IsSecure = *patch.IsSecure
	}
	if patch.ScratchMaxItems != nil {
		snapshot.ScratchMaxItems = *patch.ScratchMaxItems
	}
	if patch.ScratchMaxItemSize != nil {
		snapshot.ScratchMaxItemSize = *patch.ScratchMaxItemSize
		sizeBytes, err := config.ParseByteSize(*patch.ScratchMaxItemSize)
		if err == nil {
			snapshot.ScratchMaxItemBytes = sizeBytes
		} else {
			snapshot.ScratchMaxItemBytes = 0
		}
	}
	return snapshot
}

func configDTOFromSnapshot(snapshot runtime.Snapshot) ConfigDTO {
	return ConfigDTO{
		RootPath:           snapshot.RootPath,
		Port:               snapshot.Port,
		MaxLevel:           snapshot.MaxLevel,
		IsAllowUpload:      snapshot.IsAllowUpload,
		IsSecure:           snapshot.IsSecure,
		ScratchMaxItems:    snapshot.ScratchMaxItems,
		ScratchMaxItemSize: snapshot.ScratchMaxItemSize,
	}
}

func addressFromSnapshot(snapshot runtime.Snapshot, running bool) string {
	if !running {
		return ""
	}
	host := snapshot.IP
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d", host, snapshot.Port)
}

func hasCredentials(snapshot runtime.Snapshot) bool {
	return security.CredentialConfig{
		Username:     snapshot.Username,
		PasswordHash: snapshot.PasswordHash,
		HashAlgo:     security.HashAlgoBcrypt,
	}.IsConfigured()
}

func logEntriesFromEvents(events []accesslog.Event) []LogEntryDTO {
	entries := make([]LogEntryDTO, len(events))
	for i, event := range events {
		entries[i] = LogEntryDTO{
			Time:     event.Time,
			ClientIP: event.ClientIP,
			Method:   event.Method,
			Event:    event.Event,
			Path:     event.Path,
			Status:   event.Status,
			Result:   event.Result,
		}
	}
	return entries
}

func timeValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
