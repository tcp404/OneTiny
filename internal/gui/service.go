package gui

import (
	"strings"
	"time"

	"github.com/tcp404/OneTiny/internal/app"
)

const updateQuitDelay = 200 * time.Millisecond

type Service struct {
	service       *app.Service
	dialogs       DialogAdapter
	quitForUpdate func()
}

func NewService(service *app.Service, dialogs DialogAdapter) *Service {
	return &Service{service: service, dialogs: dialogs}
}

func (s *Service) setDialogAdapter(dialogs DialogAdapter) {
	s.dialogs = dialogs
}

func (s *Service) setQuitForUpdate(quit func()) {
	s.quitForUpdate = quit
}

func (s *Service) GetStatus() (app.StatusDTO, error) {
	return s.service.GetStatus()
}

func (s *Service) StartSharing() (app.StatusDTO, error) {
	return s.service.StartSharing()
}

func (s *Service) StopSharing() (app.StatusDTO, error) {
	return s.service.StopSharing()
}

func (s *Service) UpdateConfig(patch app.ConfigPatchDTO) (app.StatusDTO, error) {
	return s.service.UpdateConfig(patch)
}

func (s *Service) SetCredentials(patch app.CredentialPatchDTO) (app.StatusDTO, error) {
	return s.service.SetCredentials(patch)
}

func (s *Service) GetLogs(filter app.LogFilterDTO) ([]app.LogEntryDTO, error) {
	return s.service.GetLogs(filter)
}

func (s *Service) ClearLogs() error {
	return s.service.ClearLogs()
}

func (s *Service) GetUpdateStatus() (app.UpdateStatusDTO, error) {
	return s.service.GetUpdateStatus()
}

func (s *Service) CheckUpdate() (app.UpdateStatusDTO, error) {
	return s.service.CheckUpdate()
}

func (s *Service) DownloadUpdate() (app.UpdateStatusDTO, error) {
	return s.service.DownloadUpdate()
}

func (s *Service) InstallUpdate() (app.UpdateInstallDTO, error) {
	result, err := s.service.StartUpdateInstall()
	if err != nil {
		return result, err
	}
	if s.quitForUpdate != nil {
		go func() {
			time.Sleep(updateQuitDelay)
			s.quitForUpdate()
		}()
	}
	return result, nil
}

func (s *Service) ChooseDirectory(current string) (string, error) {
	if s.dialogs == nil {
		return "", nil
	}
	return s.dialogs.ChooseDirectory(current)
}

func (s *Service) ExportLogs(filter app.LogFilterDTO) (string, error) {
	if s.dialogs == nil {
		return "", nil
	}
	path, err := s.dialogs.ChooseExportPath()
	if err != nil || path == "" {
		return path, err
	}
	if err := s.service.ExportLogs(path, filter); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) OpenConfigDir() error {
	if s.dialogs == nil {
		return nil
	}
	return s.dialogs.OpenConfigDir()
}

func (s *Service) OpenShareAddress() error {
	if s.dialogs == nil {
		return nil
	}
	status, err := s.service.GetStatus()
	if err != nil {
		return err
	}
	address := strings.TrimSpace(status.Address)
	if address == "" {
		return nil
	}
	return s.dialogs.OpenURL(address)
}

func (s *Service) requestQuit(onConfirmed func()) bool {
	status, err := s.service.GetStatus()
	if err != nil {
		return false
	}
	if !status.Running {
		return true
	}
	if s.dialogs == nil {
		return false
	}
	err = s.dialogs.ConfirmQuitWhileRunning(func() {
		s.shutdown()
		if onConfirmed != nil {
			onConfirmed()
		}
	})
	if err != nil {
		return false
	}
	return false
}

func (s *Service) shutdown() {
	_, _ = s.service.StopSharing()
}
