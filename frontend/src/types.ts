import type {
  ConfigPatchDTO,
  CredentialPatchDTO,
  LogEntryDTO,
  LogFilterDTO,
  StatusDTO,
  UpdateInstallDTO,
  UpdateStatusDTO,
} from "../bindings/github.com/tcp404/OneTiny/internal/app/models.js";

export type {
  ConfigDTO,
  ConfigPatchDTO,
  CredentialPatchDTO,
  LogEntryDTO,
  LogFilterDTO,
  StatusDTO,
  UpdateInstallDTO,
  UpdateStatusDTO,
} from "../bindings/github.com/tcp404/OneTiny/internal/app/models.js";

export type TabKey = "panel" | "security" | "logs" | "about";

export type AppMode = "browser-preview" | "wails-desktop";

export interface OneTinyService {
  GetStatus(): Promise<StatusDTO>;
  StartSharing(): Promise<StatusDTO>;
  StopSharing(): Promise<StatusDTO>;
  UpdateConfig(patch: ConfigPatchDTO): Promise<StatusDTO>;
  SetCredentials(patch: CredentialPatchDTO): Promise<StatusDTO>;
  GetLogs(filter: LogFilterDTO): Promise<LogEntryDTO[]>;
  ClearLogs(): Promise<void>;
  ChooseDirectory(current: string): Promise<string>;
  ExportLogs(filter: LogFilterDTO): Promise<string>;
  OpenConfigDir(): Promise<void>;
  OpenShareAddress(): Promise<void>;
  GetUpdateStatus(): Promise<UpdateStatusDTO>;
  CheckUpdate(): Promise<UpdateStatusDTO>;
  DownloadUpdate(): Promise<UpdateStatusDTO>;
  InstallUpdate(): Promise<UpdateInstallDTO>;
}
