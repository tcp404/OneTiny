import "./styles.css";
import { appVersionLabel, updatePanelClass } from "./updatePanel";
import type {
  AppMode,
  ConfigDTO,
  ConfigPatchDTO,
  LogEntryDTO,
  LogFilterDTO,
  OneTinyService,
  StatusDTO,
  TabKey,
  UpdateInstallDTO,
  UpdateStatusDTO,
} from "./types";

type GeneratedServiceModule = typeof import("../bindings/github.com/tcp404/OneTiny/internal/gui/service.js");
type GeneratedServiceLoader = () => Promise<GeneratedServiceModule>;
type CredentialDialogState = {
  targetSecure: boolean;
  username: string;
  password: string;
  confirmPassword: string;
  error: string;
};

const appRoot = document.querySelector<HTMLDivElement>("#app");

if (!appRoot) {
  throw new Error("missing #app");
}

const app = appRoot;

const mockStatus: StatusDTO = {
  running: false,
  stateLabel: "未运行",
  address: "",
  config: {
    rootPath: "/Users/me/Downloads",
    port: 9090,
    maxLevel: 0,
    isAllowUpload: false,
    isSecure: false,
    scratchMaxItems: 500,
    scratchMaxItemSize: "10MB",
  },
  hasCredentials: false,
  configPath: "~/Library/Application Support/tiny/config.yml",
  accessLogPath: "~/Library/Application Support/tiny/access.log",
  portRestartRequired: false,
  lastError: "",
};

const mockUpdateStatus: UpdateStatusDTO = {
  currentVersion: "dev",
  latestVersion: "",
  available: false,
  state: "idle",
  message: "",
  releaseURL: "",
  downloadedPath: "",
  logPath: "",
};

const logEventOptions = [
  { value: "", label: "全部" },
  { value: "access", label: "access" },
  { value: "download", label: "download" },
  { value: "upload", label: "upload" },
  { value: "login", label: "login" },
  { value: "reject", label: "reject" },
  { value: "error", label: "error" },
];

let status: StatusDTO = mockStatus;
let logs: LogEntryDTO[] = [];
let mockLogs: LogEntryDTO[] = createMockLogs();
let logFilter: LogFilterDTO = {};
let activeTab: TabKey = "panel";
let notice = "";
let usingMock = false;
let credentialDialog: CredentialDialogState | null = null;
let updateStatus: UpdateStatusDTO = mockUpdateStatus;
let updateCheckStarted = false;

const service = createService();

void refresh().finally(() => {
  void checkUpdateOnStartup();
});

function createService(): OneTinyService {
  const loadService = () => import("../bindings/github.com/tcp404/OneTiny/internal/gui/service.js");
  const call = async <T>(
    method: keyof OneTinyService & string,
    args: unknown[],
    invoke: (generated: GeneratedServiceModule) => PromiseLike<T> | T,
  ): Promise<T> => {
    const generated = await loadGeneratedService(loadService);
    if (!generated) {
      usingMock = true;
      return mockCall<T>(method, args);
    }

    usingMock = false;
    return await invoke(generated);
  };

  return {
    GetStatus: () => call("GetStatus", [], (generated) => generated.GetStatus()),
    StartSharing: () => call("StartSharing", [], (generated) => generated.StartSharing()),
    StopSharing: () => call("StopSharing", [], (generated) => generated.StopSharing()),
    UpdateConfig: (patch) => call("UpdateConfig", [patch], (generated) => generated.UpdateConfig(patch)),
    SetCredentials: (patch) => call("SetCredentials", [patch], (generated) => generated.SetCredentials(patch)),
    GetLogs: (filter) => call("GetLogs", [filter], (generated) => generated.GetLogs(filter)),
    ClearLogs: () => call("ClearLogs", [], (generated) => generated.ClearLogs()),
    ChooseDirectory: (current) => call("ChooseDirectory", [current], (generated) => generated.ChooseDirectory(current)),
    ExportLogs: (filter) => call("ExportLogs", [filter], (generated) => generated.ExportLogs(filter)),
    OpenConfigDir: () => call("OpenConfigDir", [], (generated) => generated.OpenConfigDir()),
    OpenShareAddress: () => call("OpenShareAddress", [], (generated) => generated.OpenShareAddress()),
    GetUpdateStatus: () => call("GetUpdateStatus", [], (generated) => generated.GetUpdateStatus()),
    CheckUpdate: () => call("CheckUpdate", [], (generated) => generated.CheckUpdate()),
    DownloadUpdate: () => call("DownloadUpdate", [], (generated) => generated.DownloadUpdate()),
    InstallUpdate: () => call("InstallUpdate", [], (generated) => generated.InstallUpdate()),
  };
}

async function loadGeneratedService(loader: GeneratedServiceLoader): Promise<GeneratedServiceModule | null> {
  if (import.meta.env.VITE_ONETINY_MOCK === "1") {
    return null;
  }

  try {
    return await loader();
  } catch {
    return null;
  }
}

async function mockCall<T>(method: string, args: unknown[]): Promise<T> {
  await new Promise((resolve) => window.setTimeout(resolve, 120));
  switch (method) {
    case "GetStatus":
      return status as T;
    case "StartSharing":
      status = {
        ...status,
        running: true,
        stateLabel: "运行中",
        address: addressForPort(status.config.port),
        lastError: "",
      };
      return status as T;
    case "StopSharing":
      status = {
        ...status,
        running: false,
        stateLabel: "未运行",
        address: "",
        lastError: "",
      };
      return status as T;
    case "UpdateConfig": {
      const patch = args[0] as ConfigPatchDTO;
      if (
        status.running &&
        patch.port !== undefined &&
        patch.port !== status.config.port &&
        !patch.restartPort
      ) {
        status = { ...status, lastError: "修改端口需要确认并重启服务" };
        throw new Error(status.lastError);
      }

      const nextConfig = applyConfigPatch(status.config, patch);
      status = {
        ...status,
        config: nextConfig,
        address: status.running ? addressForPort(nextConfig.port) : "",
        portRestartRequired: false,
        lastError: "",
      };
      return status as T;
    }
    case "ChooseDirectory":
      return "/Users/me/Shared" as T;
    case "GetLogs":
      return applyLogFilter(mockLogs, (args[0] as LogFilterDTO | undefined) ?? {}) as T;
    case "ClearLogs":
      mockLogs = [];
      return undefined as T;
    case "ExportLogs":
      return "/Users/me/Downloads/onetiny-access.csv" as T;
    case "OpenConfigDir":
      return undefined as T;
    case "OpenShareAddress":
      return undefined as T;
    case "GetUpdateStatus":
      return updateStatus as T;
    case "CheckUpdate":
      updateStatus = {
        ...updateStatus,
        latestVersion: "v0.12.0",
        available: true,
        state: "available",
        message: "发现新版本: v0.12.0",
        releaseURL: "https://github.com/tcp404/OneTiny/releases/tag/v0.12.0",
      };
      return updateStatus as T;
    case "DownloadUpdate":
      updateStatus = {
        ...updateStatus,
        state: "downloaded",
        message: "更新包已下载",
        downloadedPath: "/tmp/onetiny-update/onetiny-gui-darwin-arm64.zip",
      };
      return updateStatus as T;
    case "InstallUpdate":
      updateStatus = {
        ...updateStatus,
        state: "installing",
        message: "更新安装已启动",
        logPath: "/tmp/onetiny-update.log",
      };
      return { started: true, logPath: updateStatus.logPath, message: updateStatus.message } as UpdateInstallDTO as T;
    case "SetCredentials": {
      const patch = args[0] as {
        username: string;
        password: string;
        confirmPassword: string;
        enableSecure: boolean;
      };
      const username = patch.username.trim();
      if (!username) {
        throw new Error("用户名不能为空");
      }
      if (!patch.password.trim()) {
        throw new Error("密码不能为空");
      }
      if (patch.password !== patch.confirmPassword) {
        throw new Error("两次输入的密码不一致");
      }
      status = {
        ...status,
        config: {
          ...status.config,
          isSecure: patch.enableSecure ? true : status.config.isSecure,
        },
        hasCredentials: true,
        lastError: "",
      };
      return status as T;
    }
    default:
      throw new Error(`unknown mock method: ${method}`);
  }
}

function applyConfigPatch(config: ConfigDTO, patch: ConfigPatchDTO): ConfigDTO {
  const next = { ...config };
  if (patch.rootPath != null) {
    next.rootPath = patch.rootPath;
  }
  if (patch.port != null) {
    next.port = patch.port;
  }
  if (patch.maxLevel != null) {
    next.maxLevel = patch.maxLevel;
  }
  if (patch.isAllowUpload != null) {
    next.isAllowUpload = patch.isAllowUpload;
  }
  if (patch.isSecure != null) {
    next.isSecure = patch.isSecure;
  }
  if (patch.scratchMaxItems != null) {
    next.scratchMaxItems = patch.scratchMaxItems;
  }
  if (patch.scratchMaxItemSize != null) {
    next.scratchMaxItemSize = patch.scratchMaxItemSize;
  }
  return next;
}

async function refresh(): Promise<void> {
  try {
    status = await service.GetStatus();
    updateStatus = await service.GetUpdateStatus();
    if (activeTab === "logs") {
      logs = await service.GetLogs(logFilter);
    }
    notice = runtimeModeMessage();
  } catch (error) {
    notice = errorMessage(error);
  }
  render();
}

async function checkUpdateOnStartup(): Promise<void> {
  if (updateCheckStarted) {
    return;
  }
  updateCheckStarted = true;
  updateStatus = {
    ...updateStatus,
    state: "checking",
    message: updateStateLabel("checking"),
  };
  if (activeTab === "about") {
    render();
  }
  try {
    updateStatus = await service.CheckUpdate();
    if (activeTab === "about") {
      render();
    }
  } catch (error) {
    updateStatus = {
      ...updateStatus,
      state: "error",
      message: errorMessage(error),
    };
    if (activeTab === "about") {
      render();
    }
  }
}

function render(): void {
  const effectiveNotice = notice || status.lastError;
  app.innerHTML = `
    <main class="shell">
      <section class="top-control" aria-label="共享状态">
        <div class="access-block">
          <div class="access-labels">
            <span class="label">访问地址</span>
            <span class="state ${status.running ? "state-running" : ""}">${escapeHtml(status.stateLabel)}</span>
          </div>
          <code>${escapeHtml(status.address || "服务未启动")}</code>
        </div>
        <div class="top-actions">
          <button data-action="copy" ${status.address ? "" : "disabled"}>复制地址</button>
          <button data-action="open-browser" ${status.address ? "" : "disabled"}>浏览器打开</button>
          <button class="primary" data-action="${status.running ? "stop" : "start"}">
            ${status.running ? "停止共享" : "启动共享"}
          </button>
        </div>
      </section>

      ${effectiveNotice ? `<p class="notice">${escapeHtml(effectiveNotice)}</p>` : ""}

      <header class="app-header">
        <div class="brand">
          <img class="brand-logo" src="/logo.png" alt="OneTiny">
          <div>
            <h1>OneTiny</h1>
            <p>局域网文件共享控制面板</p>
          </div>
        </div>
      </header>

      <nav class="tabs">
        ${tabButton("panel", "控制面板")}
        ${tabButton("security", "安全设置")}
        ${tabButton("logs", "访问日志")}
        ${tabButton("about", "关于")}
      </nav>

      <section class="content">
        ${renderTab()}
      </section>
      ${renderCredentialDialog()}
    </main>
  `;
  bindEvents();
}

function tabButton(key: TabKey, label: string): string {
  return `<button class="tab ${activeTab === key ? "active" : ""}" data-tab="${key}">${label}</button>`;
}

function renderTab(): string {
  switch (activeTab) {
    case "panel":
      return renderPanelTab();
    case "security":
      return renderSecurityTab();
    case "logs":
      return renderLogsTab();
    case "about":
      return renderAboutTab();
  }
}

function renderPanelTab(): string {
  return `
    <div class="control-list">
      <label class="control-row directory-row">
        <span>共享目录</span>
        <input class="readonly-input" type="text" value="${escapeHtml(status.config.rootPath)}" readonly>
        <button type="button" data-action="choose-dir">选择</button>
      </label>

      <div class="control-row">
        <span>允许上传</span>
        <label class="switch">
          <input type="checkbox" data-toggle="upload" ${status.config.isAllowUpload ? "checked" : ""}>
          <span></span>
        </label>
      </div>

      ${renderSecureControlRow()}

      <label class="control-row">
        <span>端口</span>
        <input class="number-input" type="number" min="1" max="65535" step="1" value="${status.config.port}" data-number="port">
      </label>

      <label class="control-row">
        <span>最大访问层级</span>
        <input class="number-input" type="number" min="0" max="255" step="1" value="${status.config.maxLevel}" data-number="maxLevel">
      </label>

      <label class="control-row">
        <span>临时列表容量</span>
        <input class="number-input" type="number" min="1" step="1" value="${status.config.scratchMaxItems}" data-number="scratchMaxItems">
      </label>

      <label class="control-row">
        <span>单条大小上限</span>
        <input class="number-input" type="text" value="${escapeHtml(status.config.scratchMaxItemSize)}" data-text-setting="scratchMaxItemSize">
      </label>
    </div>
  `;
}

function renderSecurityTab(): string {
  return `
    <div class="control-list">
      ${renderSecureControlRow()}
      <div class="control-row">
        <span>账号状态</span>
        <strong class="value-pill ${status.hasCredentials ? "ok" : ""}">
          ${status.hasCredentials ? "已配置" : "未配置"}
        </strong>
      </div>
      <div class="control-row">
        <span>登录保护</span>
        <strong class="value-pill ${status.config.isSecure ? "ok" : ""}">
          ${status.config.isSecure ? "已开启" : "已关闭"}
        </strong>
      </div>
    </div>
  `;
}

function renderSecureControlRow(): string {
  return `
    <div class="control-row">
      <span>登录保护</span>
      <div class="inline-actions">
        <label class="switch">
          <input type="checkbox" data-toggle="secure" ${status.config.isSecure ? "checked" : ""}>
          <span></span>
        </label>
        <button type="button" data-action="credentials">账号设置</button>
      </div>
    </div>
  `;
}

function renderLogsTab(): string {
  return `
    <form class="log-filters" aria-label="访问日志筛选">
      <label>
        <span>事件</span>
        <select name="event">
          ${logEventOptions
            .map(
              (option) => `
                <option value="${escapeHtml(option.value)}" ${option.value === (logFilter.event ?? "") ? "selected" : ""}>
                  ${escapeHtml(option.label)}
                </option>
              `,
            )
            .join("")}
        </select>
      </label>
      <label>
        <span>开始时间</span>
        <input name="since" type="datetime-local" value="${escapeHtml(dateTimeFilterToInput(logFilter.since))}">
      </label>
      <label>
        <span>结束时间</span>
        <input name="until" type="datetime-local" value="${escapeHtml(dateTimeFilterToInput(logFilter.until))}">
      </label>
      <div class="toolbar">
        <button type="button" data-action="refresh-logs">刷新</button>
        <button type="button" data-action="export-logs">导出 CSV</button>
        <button type="button" class="danger" data-action="clear-logs">清空</button>
      </div>
    </form>
    <div class="log-table">
      ${renderLogs()}
    </div>
  `;
}

function renderAboutTab(): string {
  return `
    <div class="about-panel">
      <dl class="about">
        <dt>版本</dt>
        <dd>OneTiny GUI ${escapeHtml(appVersionLabel(updateStatus.currentVersion))}</dd>
        <dt>模式</dt>
        <dd>${escapeHtml(modeLabel(currentMode()))}</dd>
        <dt>配置文件</dt>
        <dd>${escapeHtml(status.configPath || "-")}</dd>
        <dt>访问日志</dt>
        <dd>${escapeHtml(status.accessLogPath || "-")}</dd>
      </dl>
      <section class="${updatePanelClass(updateStatus)}" aria-label="软件更新">
        <div class="update-copy">
          <strong>软件更新</strong>
          <span>${escapeHtml(updateStatus.message || updateStateLabel(updateStatus.state))}</span>
          ${updateStatus.downloadedPath ? `<code>${escapeHtml(updateStatus.downloadedPath)}</code>` : ""}
          ${updateStatus.logPath ? `<code>${escapeHtml(updateStatus.logPath)}</code>` : ""}
        </div>
        <div class="update-actions">
          <button type="button" data-action="check-update" ${isUpdateBusy() ? "disabled" : ""}>检查更新</button>
          <button type="button" data-action="download-update" ${canDownloadUpdate() ? "" : "disabled"}>下载更新</button>
          <button class="primary" type="button" data-action="install-update" ${canInstallUpdate() ? "" : "disabled"}>安装并重启</button>
        </div>
      </section>
      <button data-action="open-config">打开配置目录</button>
    </div>
  `;
}

function renderCredentialDialog(): string {
  if (!credentialDialog) {
    return "";
  }

  return `
    <dialog class="credential-dialog" aria-labelledby="credential-title">
      <form class="credential-form" method="dialog">
        <div class="dialog-header">
          <h2 id="credential-title">账号设置</h2>
          <button class="icon-button" type="button" data-action="close-credentials" aria-label="关闭">×</button>
        </div>
        ${credentialDialog.error ? `<p class="dialog-error">${escapeHtml(credentialDialog.error)}</p>` : ""}
        <label>
          <span>用户名</span>
          <input name="username" autocomplete="username" value="${escapeHtml(credentialDialog.username)}">
        </label>
        <label>
          <span>密码</span>
          <input name="password" type="password" autocomplete="new-password" value="${escapeHtml(credentialDialog.password)}">
        </label>
        <label>
          <span>确认密码</span>
          <input name="confirmPassword" type="password" autocomplete="new-password" value="${escapeHtml(credentialDialog.confirmPassword)}">
        </label>
        <div class="dialog-actions">
          <button type="button" data-action="close-credentials">取消</button>
          <button class="primary" type="submit">保存</button>
        </div>
      </form>
    </dialog>
  `;
}

function renderLogs(): string {
  if (logs.length === 0) {
    return `<p class="empty">暂无访问日志</p>`;
  }
  return `
    <table>
      <thead>
        <tr>
          <th class="log-time">时间</th>
          <th class="log-ip">客户端 IP</th>
          <th class="log-method">方法</th>
          <th class="log-event">事件</th>
          <th class="log-path">路径</th>
          <th class="log-status">状态</th>
          <th class="log-result">结果</th>
        </tr>
      </thead>
      <tbody>
        ${logs
          .map(
            (entry) => `
              <tr>
                <td class="log-time">${escapeHtml(formatLogTime(entry.time))}</td>
                <td>${escapeHtml(entry.clientIP)}</td>
                <td>${escapeHtml(entry.method || "-")}</td>
                <td>${escapeHtml(entry.event)}</td>
                <td class="log-path">${escapeHtml(entry.path || "-")}</td>
                <td>${escapeHtml(entry.status ? String(entry.status) : "-")}</td>
                <td>${escapeHtml(entry.result || "-")}</td>
              </tr>
            `,
          )
          .join("")}
      </tbody>
    </table>
  `;
}

function bindEvents(): void {
  app.querySelectorAll<HTMLButtonElement>("[data-tab]").forEach((button) => {
    button.addEventListener("click", () => {
      activeTab = button.dataset.tab as TabKey;
      void refresh();
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="start"]')?.addEventListener("click", () => {
    runAction(async () => {
      status = await service.StartSharing();
      notice = runtimeModeMessage();
      render();
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="stop"]')?.addEventListener("click", () => {
    runAction(async () => {
      status = await service.StopSharing();
      notice = runtimeModeMessage();
      render();
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="copy"]')?.addEventListener("click", () => {
    runAction(async () => {
      if (status.address) {
        await navigator.clipboard.writeText(status.address);
        notice = "访问地址已复制";
        render();
      }
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="open-browser"]')?.addEventListener("click", () => {
    runAction(async () => {
      if (status.address) {
        await service.OpenShareAddress();
        notice = "已在浏览器打开";
        render();
      }
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="choose-dir"]')?.addEventListener("click", () => {
    runAction(async () => {
      const rootPath = await service.ChooseDirectory(status.config.rootPath);
      if (rootPath) {
        status = await service.UpdateConfig({ rootPath });
        notice = runtimeModeMessage();
        render();
      }
    });
  });
  app.querySelectorAll<HTMLInputElement>('[data-toggle="upload"]').forEach((input) => {
    input.addEventListener("change", () => {
      runAction(async () => {
        status = await service.UpdateConfig({ isAllowUpload: input.checked });
        notice = runtimeModeMessage();
        render();
      });
    });
  });
  app.querySelectorAll<HTMLInputElement>('[data-toggle="secure"]').forEach((input) => {
    input.addEventListener("change", () => {
      void handleSecureToggle(input.checked);
    });
  });
  app.querySelectorAll<HTMLButtonElement>('[data-action="credentials"]').forEach((button) => {
    button.addEventListener("click", () => {
      openCredentialDialog(status.config.isSecure);
    });
  });
  app.querySelectorAll<HTMLInputElement>("[data-number]").forEach((input) => {
    input.addEventListener("change", () => {
      if (input.dataset.number === "port") {
        void handlePortChange(input);
      } else if (input.dataset.number === "maxLevel") {
        void handleMaxLevelChange(input);
      } else if (input.dataset.number === "scratchMaxItems") {
        void handleScratchMaxItemsChange(input);
      }
    });
  });
  app.querySelectorAll<HTMLInputElement>("[data-text-setting]").forEach((input) => {
    input.addEventListener("change", () => {
      if (input.dataset.textSetting === "scratchMaxItemSize") {
        void handleScratchMaxItemSizeChange(input);
      }
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="open-config"]')?.addEventListener("click", () => {
    runAction(async () => {
      await service.OpenConfigDir();
      notice = runtimeModeMessage();
      render();
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="check-update"]')?.addEventListener("click", () => {
    runUpdateAction(async () => {
      updateStatus = {
        ...updateStatus,
        state: "checking",
        message: updateStateLabel("checking"),
      };
      render();
      updateStatus = await service.CheckUpdate();
      notice = runtimeModeMessage();
      render();
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="download-update"]')?.addEventListener("click", () => {
    runUpdateAction(async () => {
      updateStatus = {
        ...updateStatus,
        state: "downloading",
        message: updateStateLabel("downloading"),
      };
      render();
      updateStatus = await service.DownloadUpdate();
      notice = runtimeModeMessage();
      render();
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="install-update"]')?.addEventListener("click", () => {
    const confirmed = window.confirm("安装更新会退出 OneTiny 并停止共享服务，是否继续？");
    if (!confirmed) {
      return;
    }
    runUpdateAction(async () => {
      updateStatus = {
        ...updateStatus,
        state: "installing",
        message: updateStateLabel("installing"),
      };
      render();
      const result = await service.InstallUpdate();
      updateStatus = {
        ...updateStatus,
        state: "installing",
        message: result.message || updateStateLabel("installing"),
        logPath: result.logPath,
      };
      notice = runtimeModeMessage();
      render();
    });
  });
  app.querySelector<HTMLFormElement>(".log-filters")?.addEventListener("submit", (event) => {
    event.preventDefault();
    runAction(async () => {
      logFilter = readLogFilterFromInputs();
      logs = await service.GetLogs(logFilter);
      notice = runtimeModeMessage();
      render();
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="refresh-logs"]')?.addEventListener("click", () => {
    runAction(async () => {
      logFilter = readLogFilterFromInputs();
      logs = await service.GetLogs(logFilter);
      notice = runtimeModeMessage();
      render();
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="export-logs"]')?.addEventListener("click", () => {
    runAction(async () => {
      logFilter = readLogFilterFromInputs();
      const path = await service.ExportLogs(logFilter);
      notice = path ? `已导出到 ${path}` : runtimeModeMessage();
      render();
    });
  });
  app.querySelector<HTMLButtonElement>('[data-action="clear-logs"]')?.addEventListener("click", () => {
    const confirmed = window.confirm("确定清空访问日志？");
    if (!confirmed) {
      return;
    }

    runAction(async () => {
      await service.ClearLogs();
      logs = [];
      notice = runtimeModeMessage();
      render();
    });
  });
  app.querySelectorAll<HTMLButtonElement>('[data-action="close-credentials"]').forEach((button) => {
    button.addEventListener("click", () => {
      closeCredentialDialog();
    });
  });
  app.querySelector<HTMLFormElement>(".credential-form")?.addEventListener("submit", (event) => {
    event.preventDefault();
    void handleCredentialSave();
  });
  activateCredentialDialog();
}

function runAction(action: () => Promise<void>): void {
  void action().catch((error) => {
    notice = errorMessage(error);
    render();
  });
}

function runUpdateAction(action: () => Promise<void>): void {
  void action().catch((error) => {
    void handleUpdateActionError(error);
  });
}

async function handleUpdateActionError(error: unknown): Promise<void> {
  const message = errorMessage(error);
  notice = runtimeModeMessage();
  try {
    updateStatus = await service.GetUpdateStatus();
    if (!updateStatus.message) {
      updateStatus = { ...updateStatus, message };
    }
  } catch {
    updateStatus = {
      ...updateStatus,
      state: "error",
      message,
    };
  }
  render();
}

async function handleSecureToggle(enabled: boolean): Promise<void> {
  if (enabled && !status.hasCredentials) {
    openCredentialDialog(true);
    return;
  }

  runAction(async () => {
    status = await service.UpdateConfig({ isSecure: enabled });
    notice = runtimeModeMessage();
    render();
  });
}

async function handlePortChange(input: HTMLInputElement): Promise<void> {
  const port = parseIntegerInput(input.value, 1, 65535, "端口");
  if (port === null) {
    render();
    return;
  }
  if (port === status.config.port) {
    return;
  }

  if (status.running) {
    const confirmed = window.confirm("修改端口需要重启共享服务，是否继续？");
    if (!confirmed) {
      render();
      return;
    }
  }

  runAction(async () => {
    status = await service.UpdateConfig({ port, restartPort: status.running });
    notice = runtimeModeMessage();
    render();
  });
}

async function handleMaxLevelChange(input: HTMLInputElement): Promise<void> {
  const maxLevel = parseIntegerInput(input.value, 0, 255, "最大访问层级");
  if (maxLevel === null) {
    render();
    return;
  }
  if (maxLevel === status.config.maxLevel) {
    return;
  }

  runAction(async () => {
    status = await service.UpdateConfig({ maxLevel });
    notice = runtimeModeMessage();
    render();
  });
}

async function handleScratchMaxItemsChange(input: HTMLInputElement): Promise<void> {
  const scratchMaxItems = parsePositiveIntegerInput(input.value, "临时列表容量");
  if (scratchMaxItems === null) {
    render();
    return;
  }
  if (scratchMaxItems === status.config.scratchMaxItems) {
    return;
  }
  runAction(async () => {
    status = await service.UpdateConfig({ scratchMaxItems });
    notice = runtimeModeMessage();
    render();
  });
}

async function handleScratchMaxItemSizeChange(input: HTMLInputElement): Promise<void> {
  const scratchMaxItemSize = input.value.trim();
  if (!/^[1-9][0-9]*\s*(B|KB|K|MB|M|GB|G)?$/i.test(scratchMaxItemSize)) {
    notice = "单条大小上限格式无效";
    render();
    return;
  }
  if (scratchMaxItemSize === status.config.scratchMaxItemSize) {
    return;
  }
  runAction(async () => {
    status = await service.UpdateConfig({ scratchMaxItemSize });
    notice = runtimeModeMessage();
    render();
  });
}

async function handleCredentialSave(): Promise<void> {
  if (!credentialDialog) {
    return;
  }

  const username = formValue("username").trim();
  const password = formValue("password");
  const confirmPassword = formValue("confirmPassword");
  const targetSecure = credentialDialog.targetSecure;

  credentialDialog = {
    ...credentialDialog,
    username,
    password,
    confirmPassword,
    error: "",
  };

  const validationError = validateCredentials(username, password, confirmPassword);
  if (validationError) {
    credentialDialog.error = validationError;
    notice = "";
    render();
    return;
  }

  runAction(async () => {
    status = await service.SetCredentials({
      username,
      password,
      confirmPassword,
      enableSecure: targetSecure,
    });
    credentialDialog = null;
    notice = runtimeModeMessage();
    render();
  });
}

function openCredentialDialog(targetSecure: boolean): void {
  credentialDialog = {
    targetSecure,
    username: "",
    password: "",
    confirmPassword: "",
    error: "",
  };
  notice = "";
  render();
}

function closeCredentialDialog(): void {
  credentialDialog = null;
  notice = runtimeModeMessage();
  render();
}

function activateCredentialDialog(): void {
  const dialog = app.querySelector<HTMLDialogElement>(".credential-dialog");
  if (!dialog) {
    return;
  }

  const clearDialog = () => {
    if (credentialDialog) {
      credentialDialog = null;
      notice = runtimeModeMessage();
      render();
    }
  };

  dialog.addEventListener("cancel", (event) => {
    event.preventDefault();
    clearDialog();
  });
  dialog.addEventListener("close", clearDialog);

  if (!dialog.open) {
    dialog.showModal();
  }
  dialog.querySelector<HTMLInputElement>('input[name="username"]')?.focus();
}

function validateCredentials(username: string, password: string, confirmPassword: string): string {
  if (!username) {
    return "用户名不能为空";
  }
  if (!password.trim()) {
    return "密码不能为空";
  }
  if (password !== confirmPassword) {
    return "两次输入的密码不一致";
  }
  return "";
}

function formValue(name: string): string {
  return app.querySelector<HTMLInputElement>(`.credential-form [name="${name}"]`)?.value ?? "";
}

function parseIntegerInput(value: string, min: number, max: number, label: string): number | null {
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed < min || parsed > max) {
    notice = `${label}必须在 ${min}-${max} 之间`;
    return null;
  }
  return parsed;
}

function parsePositiveIntegerInput(value: string, label: string): number | null {
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed < 1) {
    notice = `${label}必须为正整数`;
    return null;
  }
  return parsed;
}

function runtimeModeMessage(): string {
  return usingMock ? "浏览器预览模式" : "";
}

function currentMode(): AppMode {
  return usingMock ? "browser-preview" : "wails-desktop";
}

function modeLabel(mode: AppMode): string {
  return mode === "browser-preview" ? "浏览器预览模式" : "Wails 桌面运行时";
}

function updateStateLabel(state: string): string {
  switch (state) {
    case "checking":
      return "正在检查更新";
    case "available":
      return "有新版本可用";
    case "downloading":
      return "正在下载更新";
    case "downloaded":
      return "更新包已下载";
    case "installing":
      return "更新安装已启动";
    case "current":
      return "当前已是最新版本";
    case "unknown":
      return "无法判断更新状态";
    case "error":
      return "检查更新失败";
    default:
      return "尚未检查";
  }
}

function isUpdateBusy(): boolean {
  return updateStatus.state === "checking" || updateStatus.state === "downloading" || updateStatus.state === "installing";
}

function canDownloadUpdate(): boolean {
  return updateStatus.available && updateStatus.state !== "downloaded" && !isUpdateBusy();
}

function canInstallUpdate(): boolean {
  return updateStatus.state === "downloaded";
}

function addressForPort(port: number): string {
  return `http://127.0.0.1:${port}`;
}

function readLogFilterFromInputs(): LogFilterDTO {
  const form = app.querySelector<HTMLFormElement>(".log-filters");
  if (!form) {
    return logFilter;
  }

  const formData = new FormData(form);
  const event = String(formData.get("event") ?? "").trim();
  const since = dateTimeInputToIso(String(formData.get("since") ?? ""));
  const until = dateTimeInputToIso(String(formData.get("until") ?? ""));
  const filter: LogFilterDTO = {};

  if (event) {
    filter.event = event;
  }
  if (since) {
    filter.since = since;
  }
  if (until) {
    filter.until = until;
  }

  return filter;
}

function dateTimeInputToIso(value: string): string | null {
  if (!value) {
    return null;
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date.toISOString();
}

function dateTimeFilterToInput(value?: string | null): string {
  if (!value) {
    return "";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }

  const localDate = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return localDate.toISOString().slice(0, 16);
}

function formatLogTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value || "-";
  }
  return new Intl.DateTimeFormat(undefined, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(date);
}

function createMockLogs(): LogEntryDTO[] {
  const now = Date.now();
  const minutesAgo = (minutes: number): string => new Date(now - minutes * 60_000).toISOString();

  return [
    {
      time: minutesAgo(4),
      clientIP: "192.168.31.18",
      method: "GET",
      event: "access",
      path: "/",
      status: 200,
      result: "ok",
    },
    {
      time: minutesAgo(16),
      clientIP: "192.168.31.42",
      method: "GET",
      event: "download",
      path: "/photos/2026/spring-trip/very-long-file-name-that-should-wrap-in-the-log-table-without-breaking-layout.jpg",
      status: 200,
      result: "sent",
    },
    {
      time: minutesAgo(28),
      clientIP: "192.168.31.42",
      method: "POST",
      event: "upload",
      path: "/uploads/report-final.pdf",
      status: 201,
      result: "created",
    },
    {
      time: minutesAgo(44),
      clientIP: "192.168.31.9",
      method: "POST",
      event: "login",
      path: "/login",
      status: 200,
      result: "authenticated",
    },
    {
      time: minutesAgo(63),
      clientIP: "192.168.31.77",
      method: "GET",
      event: "reject",
      path: "/private/<script>alert(1)</script>.txt",
      status: 403,
      result: "blocked",
    },
    {
      time: minutesAgo(87),
      clientIP: "192.168.31.51",
      method: "GET",
      event: "error",
      path: "/archive.zip",
      status: 500,
      result: "read failed",
    },
  ];
}

function applyLogFilter(entries: LogEntryDTO[], filter: LogFilterDTO): LogEntryDTO[] {
  const event = filter.event?.trim();
  const since = filterTime(filter.since);
  const until = filterTime(filter.until);

  return entries.filter((entry) => {
    const entryTime = filterTime(entry.time);
    if (event && entry.event !== event) {
      return false;
    }
    if (since !== null && entryTime !== null && entryTime < since) {
      return false;
    }
    if (until !== null && entryTime !== null && entryTime > until) {
      return false;
    }
    return true;
  });
}

function filterTime(value?: string | null): number | null {
  if (!value) {
    return null;
  }

  const timestamp = new Date(value).getTime();
  return Number.isNaN(timestamp) ? null : timestamp;
}

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (char) => {
    const map: Record<string, string> = {
      "&": "&amp;",
      "<": "&lt;",
      ">": "&gt;",
      '"': "&quot;",
      "'": "&#39;",
    };
    return map[char];
  });
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}
