(function(){let e=document.createElement(`link`).relList;if(e&&e.supports&&e.supports(`modulepreload`))return;for(let e of document.querySelectorAll(`link[rel="modulepreload"]`))n(e);new MutationObserver(e=>{for(let t of e)if(t.type===`childList`)for(let e of t.addedNodes)e.tagName===`LINK`&&e.rel===`modulepreload`&&n(e)}).observe(document,{childList:!0,subtree:!0});function t(e){let t={};return e.integrity&&(t.integrity=e.integrity),e.referrerPolicy&&(t.referrerPolicy=e.referrerPolicy),e.crossOrigin===`use-credentials`?t.credentials=`include`:e.crossOrigin===`anonymous`?t.credentials=`omit`:t.credentials=`same-origin`,t}function n(e){if(e.ep)return;e.ep=!0;let n=t(e);fetch(e.href,n)}})();var e=new Set([`available`,`downloading`,`downloaded`,`installing`]);function t(t){let n=[`update-panel`];return t.state===`current`?n.push(`current`):(t.available||e.has(t.state))&&n.push(`available`),n.join(` `)}function n(e){return e.trim()||`dev`}var r=`modulepreload`,i=function(e){return`/`+e},a={},o=function(e,t,n){let o=Promise.resolve();if(t&&t.length>0){let e=document.getElementsByTagName(`link`),s=document.querySelector(`meta[property=csp-nonce]`),c=s?.nonce||s?.getAttribute(`nonce`);function l(e){return Promise.all(e.map(e=>Promise.resolve(e).then(e=>({status:`fulfilled`,value:e}),e=>({status:`rejected`,reason:e}))))}o=l(t.map(t=>{if(t=i(t,n),t in a)return;a[t]=!0;let o=t.endsWith(`.css`),s=o?`[rel="stylesheet"]`:``;if(n)for(let n=e.length-1;n>=0;n--){let r=e[n];if(r.href===t&&(!o||r.rel===`stylesheet`))return}else if(document.querySelector(`link[href="${t}"]${s}`))return;let l=document.createElement(`link`);if(l.rel=o?`stylesheet`:r,o||(l.as=`script`),l.crossOrigin=``,l.href=t,c&&l.setAttribute(`nonce`,c),document.head.appendChild(l),o)return new Promise((e,n)=>{l.addEventListener(`load`,e),l.addEventListener(`error`,()=>n(Error(`Unable to preload CSS for ${t}`)))})}))}function s(e){let t=new Event(`vite:preloadError`,{cancelable:!0});if(t.payload=e,window.dispatchEvent(t),!t.defaultPrevented)throw e}return o.then(t=>{for(let e of t||[])e.status===`rejected`&&s(e.reason);return e().catch(s)})},s=document.querySelector(`#app`);if(!s)throw Error(`missing #app`);var c=s,l={running:!1,stateLabel:`未运行`,address:``,config:{rootPath:`/Users/me/Downloads`,port:9090,maxLevel:0,isAllowUpload:!1,isSecure:!1,scratchMaxItems:500,scratchMaxItemSize:`10MB`},hasCredentials:!1,configPath:`~/Library/Application Support/tiny/config.yml`,accessLogPath:`~/Library/Application Support/tiny/access.log`,portRestartRequired:!1,lastError:``},ee={currentVersion:`dev`,latestVersion:``,available:!1,state:`idle`,message:``,releaseURL:``,downloadedPath:``,logPath:``},te=[{value:``,label:`全部`},{value:`access`,label:`access`},{value:`download`,label:`download`},{value:`upload`,label:`upload`},{value:`login`,label:`login`},{value:`reject`,label:`reject`},{value:`error`,label:`error`}],u=l,d=[],f=_e(),p={},m=`panel`,h=``,g=!1,_=null,v=ee,y=!1,b=x();S().finally(()=>{ae()});function x(){let e=()=>o(()=>import(`./service-DDpDXLTC.js`),[]),t=async(t,n,r)=>{let i=await ne(e);return i?(g=!1,await r(i)):(g=!0,re(t,n))};return{GetStatus:()=>t(`GetStatus`,[],e=>e.GetStatus()),StartSharing:()=>t(`StartSharing`,[],e=>e.StartSharing()),StopSharing:()=>t(`StopSharing`,[],e=>e.StopSharing()),UpdateConfig:e=>t(`UpdateConfig`,[e],t=>t.UpdateConfig(e)),SetCredentials:e=>t(`SetCredentials`,[e],t=>t.SetCredentials(e)),GetLogs:e=>t(`GetLogs`,[e],t=>t.GetLogs(e)),ClearLogs:()=>t(`ClearLogs`,[],e=>e.ClearLogs()),ChooseDirectory:e=>t(`ChooseDirectory`,[e],t=>t.ChooseDirectory(e)),ExportLogs:e=>t(`ExportLogs`,[e],t=>t.ExportLogs(e)),OpenConfigDir:()=>t(`OpenConfigDir`,[],e=>e.OpenConfigDir()),OpenShareAddress:()=>t(`OpenShareAddress`,[],e=>e.OpenShareAddress()),GetUpdateStatus:()=>t(`GetUpdateStatus`,[],e=>e.GetUpdateStatus()),CheckUpdate:()=>t(`CheckUpdate`,[],e=>e.CheckUpdate()),DownloadUpdate:()=>t(`DownloadUpdate`,[],e=>e.DownloadUpdate()),InstallUpdate:()=>t(`InstallUpdate`,[],e=>e.InstallUpdate())}}async function ne(e){try{return await e()}catch{return null}}async function re(e,t){switch(await new Promise(e=>window.setTimeout(e,120)),e){case`GetStatus`:return u;case`StartSharing`:return u={...u,running:!0,stateLabel:`运行中`,address:q(u.config.port),lastError:``},u;case`StopSharing`:return u={...u,running:!1,stateLabel:`未运行`,address:``,lastError:``},u;case`UpdateConfig`:{let e=t[0];if(u.running&&e.port!==void 0&&e.port!==u.config.port&&!e.restartPort)throw u={...u,lastError:`修改端口需要确认并重启服务`},Error(u.lastError);let n=ie(u.config,e);return u={...u,config:n,address:u.running?q(n.port):``,portRestartRequired:!1,lastError:``},u}case`ChooseDirectory`:return`/Users/me/Shared`;case`GetLogs`:return ve(f,t[0]??{});case`ClearLogs`:f=[];return;case`ExportLogs`:return`/Users/me/Downloads/onetiny-access.csv`;case`OpenConfigDir`:return;case`OpenShareAddress`:return;case`GetUpdateStatus`:return v;case`CheckUpdate`:return v={...v,latestVersion:`v0.12.0`,available:!0,state:`available`,message:`发现新版本: v0.12.0`,releaseURL:`https://github.com/tcp404/OneTiny/releases/tag/v0.12.0`},v;case`DownloadUpdate`:return v={...v,state:`downloaded`,message:`更新包已下载`,downloadedPath:`/tmp/onetiny-update/onetiny-gui-darwin-arm64.zip`},v;case`InstallUpdate`:return v={...v,state:`installing`,message:`更新安装已启动`,logPath:`/tmp/onetiny-update.log`},{started:!0,logPath:v.logPath,message:v.message};case`SetCredentials`:{let e=t[0];if(!e.username.trim())throw Error(`用户名不能为空`);if(!e.password.trim())throw Error(`密码不能为空`);if(e.password!==e.confirmPassword)throw Error(`两次输入的密码不一致`);return u={...u,config:{...u.config,isSecure:e.enableSecure?!0:u.config.isSecure},hasCredentials:!0,lastError:``},u}default:throw Error(`unknown mock method: ${e}`)}}function ie(e,t){let n={...e};return t.rootPath!=null&&(n.rootPath=t.rootPath),t.port!=null&&(n.port=t.port),t.maxLevel!=null&&(n.maxLevel=t.maxLevel),t.isAllowUpload!=null&&(n.isAllowUpload=t.isAllowUpload),t.isSecure!=null&&(n.isSecure=t.isSecure),t.scratchMaxItems!=null&&(n.scratchMaxItems=t.scratchMaxItems),t.scratchMaxItemSize!=null&&(n.scratchMaxItemSize=t.scratchMaxItemSize),n}async function S(){try{u=await b.GetStatus(),v=await b.GetUpdateStatus(),m===`logs`&&(d=await b.GetLogs(p)),h=W()}catch(e){h=$(e)}C()}async function ae(){if(!y){y=!0,v={...v,state:`checking`,message:G(`checking`)},m===`about`&&C();try{v=await b.CheckUpdate(),m===`about`&&C()}catch(e){v={...v,state:`error`,message:$(e)},m===`about`&&C()}}}function C(){let e=h||u.lastError;c.innerHTML=`
    <main class="shell">
      <section class="top-control" aria-label="共享状态">
        <div class="access-block">
          <div class="access-labels">
            <span class="label">访问地址</span>
            <span class="state ${u.running?`state-running`:``}">${Q(u.stateLabel)}</span>
          </div>
          <code>${Q(u.address||`服务未启动`)}</code>
        </div>
        <div class="top-actions">
          <button data-action="copy" ${u.address?``:`disabled`}>复制地址</button>
          <button data-action="open-browser" ${u.address?``:`disabled`}>浏览器打开</button>
          <button class="primary" data-action="${u.running?`stop`:`start`}">
            ${u.running?`停止共享`:`启动共享`}
          </button>
        </div>
      </section>

      ${e?`<p class="notice">${Q(e)}</p>`:``}

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
        ${w(`panel`,`控制面板`)}
        ${w(`security`,`安全设置`)}
        ${w(`logs`,`访问日志`)}
        ${w(`about`,`关于`)}
      </nav>

      <section class="content">
        ${T()}
      </section>
      ${ce()}
    </main>
  `,ue()}function w(e,t){return`<button class="tab ${m===e?`active`:``}" data-tab="${e}">${t}</button>`}function T(){switch(m){case`panel`:return E();case`security`:return oe();case`logs`:return O();case`about`:return se()}}function E(){return`
    <div class="control-list">
      <label class="control-row directory-row">
        <span>共享目录</span>
        <input class="readonly-input" type="text" value="${Q(u.config.rootPath)}" readonly>
        <button type="button" data-action="choose-dir">选择</button>
      </label>

      <div class="control-row">
        <span>允许上传</span>
        <label class="switch">
          <input type="checkbox" data-toggle="upload" ${u.config.isAllowUpload?`checked`:``}>
          <span></span>
        </label>
      </div>

      ${D()}

      <label class="control-row">
        <span>端口</span>
        <input class="number-input" type="number" min="1" max="65535" step="1" value="${u.config.port}" data-number="port">
      </label>

      <label class="control-row">
        <span>最大访问层级</span>
        <input class="number-input" type="number" min="0" max="255" step="1" value="${u.config.maxLevel}" data-number="maxLevel">
      </label>

      <label class="control-row">
        <span>临时列表容量</span>
        <input class="number-input" type="number" min="1" step="1" value="${u.config.scratchMaxItems}" data-number="scratchMaxItems">
      </label>

      <label class="control-row">
        <span>单条大小上限</span>
        <input class="number-input" type="text" value="${Q(u.config.scratchMaxItemSize)}" data-text-setting="scratchMaxItemSize">
      </label>
    </div>
  `}function oe(){return`
    <div class="control-list">
      ${D()}
      <div class="control-row">
        <span>账号状态</span>
        <strong class="value-pill ${u.hasCredentials?`ok`:``}">
          ${u.hasCredentials?`已配置`:`未配置`}
        </strong>
      </div>
      <div class="control-row">
        <span>登录保护</span>
        <strong class="value-pill ${u.config.isSecure?`ok`:``}">
          ${u.config.isSecure?`已开启`:`已关闭`}
        </strong>
      </div>
    </div>
  `}function D(){return`
    <div class="control-row">
      <span>登录保护</span>
      <div class="inline-actions">
        <label class="switch">
          <input type="checkbox" data-toggle="secure" ${u.config.isSecure?`checked`:``}>
          <span></span>
        </label>
        <button type="button" data-action="credentials">账号设置</button>
      </div>
    </div>
  `}function O(){return`
    <form class="log-filters" aria-label="访问日志筛选">
      <label>
        <span>事件</span>
        <select name="event">
          ${te.map(e=>`
                <option value="${Q(e.value)}" ${e.value===(p.event??``)?`selected`:``}>
                  ${Q(e.label)}
                </option>
              `).join(``)}
        </select>
      </label>
      <label>
        <span>开始时间</span>
        <input name="since" type="datetime-local" value="${Q(X(p.since))}">
      </label>
      <label>
        <span>结束时间</span>
        <input name="until" type="datetime-local" value="${Q(X(p.until))}">
      </label>
      <div class="toolbar">
        <button type="button" data-action="refresh-logs">刷新</button>
        <button type="button" data-action="export-logs">导出 CSV</button>
        <button type="button" class="danger" data-action="clear-logs">清空</button>
      </div>
    </form>
    <div class="log-table">
      ${le()}
    </div>
  `}function se(){return`
    <div class="about-panel">
      <dl class="about">
        <dt>版本</dt>
        <dd>OneTiny GUI ${Q(n(v.currentVersion))}</dd>
        <dt>模式</dt>
        <dd>${Q(pe(fe()))}</dd>
        <dt>配置文件</dt>
        <dd>${Q(u.configPath||`-`)}</dd>
        <dt>访问日志</dt>
        <dd>${Q(u.accessLogPath||`-`)}</dd>
      </dl>
      <section class="${t(v)}" aria-label="软件更新">
        <div class="update-copy">
          <strong>软件更新</strong>
          <span>${Q(v.message||G(v.state))}</span>
          ${v.downloadedPath?`<code>${Q(v.downloadedPath)}</code>`:``}
          ${v.logPath?`<code>${Q(v.logPath)}</code>`:``}
        </div>
        <div class="update-actions">
          <button type="button" data-action="check-update" ${K()?`disabled`:``}>检查更新</button>
          <button type="button" data-action="download-update" ${me()?``:`disabled`}>下载更新</button>
          <button class="primary" type="button" data-action="install-update" ${he()?``:`disabled`}>安装并重启</button>
        </div>
      </section>
      <button data-action="open-config">打开配置目录</button>
    </div>
  `}function ce(){return _?`
    <dialog class="credential-dialog" aria-labelledby="credential-title">
      <form class="credential-form" method="dialog">
        <div class="dialog-header">
          <h2 id="credential-title">账号设置</h2>
          <button class="icon-button" type="button" data-action="close-credentials" aria-label="关闭">×</button>
        </div>
        ${_.error?`<p class="dialog-error">${Q(_.error)}</p>`:``}
        <label>
          <span>用户名</span>
          <input name="username" autocomplete="username" value="${Q(_.username)}">
        </label>
        <label>
          <span>密码</span>
          <input name="password" type="password" autocomplete="new-password" value="${Q(_.password)}">
        </label>
        <label>
          <span>确认密码</span>
          <input name="confirmPassword" type="password" autocomplete="new-password" value="${Q(_.confirmPassword)}">
        </label>
        <div class="dialog-actions">
          <button type="button" data-action="close-credentials">取消</button>
          <button class="primary" type="submit">保存</button>
        </div>
      </form>
    </dialog>
  `:``}function le(){return d.length===0?`<p class="empty">暂无访问日志</p>`:`
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
        ${d.map(e=>`
              <tr>
                <td class="log-time">${Q(ge(e.time))}</td>
                <td>${Q(e.clientIP)}</td>
                <td>${Q(e.method||`-`)}</td>
                <td>${Q(e.event)}</td>
                <td class="log-path">${Q(e.path||`-`)}</td>
                <td>${Q(e.status?String(e.status):`-`)}</td>
                <td>${Q(e.result||`-`)}</td>
              </tr>
            `).join(``)}
      </tbody>
    </table>
  `}function ue(){c.querySelectorAll(`[data-tab]`).forEach(e=>{e.addEventListener(`click`,()=>{m=e.dataset.tab,S()})}),c.querySelector(`[data-action="start"]`)?.addEventListener(`click`,()=>{k(async()=>{u=await b.StartSharing(),h=W(),C()})}),c.querySelector(`[data-action="stop"]`)?.addEventListener(`click`,()=>{k(async()=>{u=await b.StopSharing(),h=W(),C()})}),c.querySelector(`[data-action="copy"]`)?.addEventListener(`click`,()=>{k(async()=>{u.address&&(await navigator.clipboard.writeText(u.address),h=`访问地址已复制`,C())})}),c.querySelector(`[data-action="open-browser"]`)?.addEventListener(`click`,()=>{k(async()=>{u.address&&(await b.OpenShareAddress(),h=`已在浏览器打开`,C())})}),c.querySelector(`[data-action="choose-dir"]`)?.addEventListener(`click`,()=>{k(async()=>{let e=await b.ChooseDirectory(u.config.rootPath);e&&(u=await b.UpdateConfig({rootPath:e}),h=W(),C())})}),c.querySelectorAll(`[data-toggle="upload"]`).forEach(e=>{e.addEventListener(`change`,()=>{k(async()=>{u=await b.UpdateConfig({isAllowUpload:e.checked}),h=W(),C()})})}),c.querySelectorAll(`[data-toggle="secure"]`).forEach(e=>{e.addEventListener(`change`,()=>{M(e.checked)})}),c.querySelectorAll(`[data-action="credentials"]`).forEach(e=>{e.addEventListener(`click`,()=>{R(u.config.isSecure)})}),c.querySelectorAll(`[data-number]`).forEach(e=>{e.addEventListener(`change`,()=>{e.dataset.number===`port`?N(e):e.dataset.number===`maxLevel`?P(e):e.dataset.number===`scratchMaxItems`&&F(e)})}),c.querySelectorAll(`[data-text-setting]`).forEach(e=>{e.addEventListener(`change`,()=>{e.dataset.textSetting===`scratchMaxItemSize`&&I(e)})}),c.querySelector(`[data-action="open-config"]`)?.addEventListener(`click`,()=>{k(async()=>{await b.OpenConfigDir(),h=W(),C()})}),c.querySelector(`[data-action="check-update"]`)?.addEventListener(`click`,()=>{A(async()=>{v={...v,state:`checking`,message:G(`checking`)},C(),v=await b.CheckUpdate(),h=W(),C()})}),c.querySelector(`[data-action="download-update"]`)?.addEventListener(`click`,()=>{A(async()=>{v={...v,state:`downloading`,message:G(`downloading`)},C(),v=await b.DownloadUpdate(),h=W(),C()})}),c.querySelector(`[data-action="install-update"]`)?.addEventListener(`click`,()=>{window.confirm(`安装更新会退出 OneTiny 并停止共享服务，是否继续？`)&&A(async()=>{v={...v,state:`installing`,message:G(`installing`)},C();let e=await b.InstallUpdate();v={...v,state:`installing`,message:e.message||G(`installing`),logPath:e.logPath},h=W(),C()})}),c.querySelector(`.log-filters`)?.addEventListener(`submit`,e=>{e.preventDefault(),k(async()=>{p=J(),d=await b.GetLogs(p),h=W(),C()})}),c.querySelector(`[data-action="refresh-logs"]`)?.addEventListener(`click`,()=>{k(async()=>{p=J(),d=await b.GetLogs(p),h=W(),C()})}),c.querySelector(`[data-action="export-logs"]`)?.addEventListener(`click`,()=>{k(async()=>{p=J();let e=await b.ExportLogs(p);h=e?`已导出到 ${e}`:W(),C()})}),c.querySelector(`[data-action="clear-logs"]`)?.addEventListener(`click`,()=>{window.confirm(`确定清空访问日志？`)&&k(async()=>{await b.ClearLogs(),d=[],h=W(),C()})}),c.querySelectorAll(`[data-action="close-credentials"]`).forEach(e=>{e.addEventListener(`click`,()=>{z()})}),c.querySelector(`.credential-form`)?.addEventListener(`submit`,e=>{e.preventDefault(),L()}),B()}function k(e){e().catch(e=>{h=$(e),C()})}function A(e){e().catch(e=>{j(e)})}async function j(e){let t=$(e);h=W();try{v=await b.GetUpdateStatus(),v.message||(v={...v,message:t})}catch{v={...v,state:`error`,message:t}}C()}async function M(e){if(e&&!u.hasCredentials){R(!0);return}k(async()=>{u=await b.UpdateConfig({isSecure:e}),h=W(),C()})}async function N(e){let t=U(e.value,1,65535,`端口`);if(t===null){C();return}if(t!==u.config.port){if(u.running&&!window.confirm(`修改端口需要重启共享服务，是否继续？`)){C();return}k(async()=>{u=await b.UpdateConfig({port:t,restartPort:u.running}),h=W(),C()})}}async function P(e){let t=U(e.value,0,255,`最大访问层级`);if(t===null){C();return}t!==u.config.maxLevel&&k(async()=>{u=await b.UpdateConfig({maxLevel:t}),h=W(),C()})}async function F(e){let t=de(e.value,`临时列表容量`);if(t===null){C();return}t!==u.config.scratchMaxItems&&k(async()=>{u=await b.UpdateConfig({scratchMaxItems:t}),h=W(),C()})}async function I(e){let t=e.value.trim();if(!/^[1-9][0-9]*\s*(B|KB|K|MB|M|GB|G)?$/i.test(t)){h=`单条大小上限格式无效`,C();return}t!==u.config.scratchMaxItemSize&&k(async()=>{u=await b.UpdateConfig({scratchMaxItemSize:t}),h=W(),C()})}async function L(){if(!_)return;let e=H(`username`).trim(),t=H(`password`),n=H(`confirmPassword`),r=_.targetSecure;_={..._,username:e,password:t,confirmPassword:n,error:``};let i=V(e,t,n);if(i){_.error=i,h=``,C();return}k(async()=>{u=await b.SetCredentials({username:e,password:t,confirmPassword:n,enableSecure:r}),_=null,h=W(),C()})}function R(e){_={targetSecure:e,username:``,password:``,confirmPassword:``,error:``},h=``,C()}function z(){_=null,h=W(),C()}function B(){let e=c.querySelector(`.credential-dialog`);if(!e)return;let t=()=>{_&&(_=null,h=W(),C())};e.addEventListener(`cancel`,e=>{e.preventDefault(),t()}),e.addEventListener(`close`,t),e.open||e.showModal(),e.querySelector(`input[name="username"]`)?.focus()}function V(e,t,n){return e?t.trim()?t===n?``:`两次输入的密码不一致`:`密码不能为空`:`用户名不能为空`}function H(e){return c.querySelector(`.credential-form [name="${e}"]`)?.value??``}function U(e,t,n,r){let i=Number(e);return!Number.isInteger(i)||i<t||i>n?(h=`${r}必须在 ${t}-${n} 之间`,null):i}function de(e,t){let n=Number(e);return!Number.isInteger(n)||n<1?(h=`${t}必须为正整数`,null):n}function W(){return g?`浏览器预览模式`:``}function fe(){return g?`browser-preview`:`wails-desktop`}function pe(e){return e===`browser-preview`?`浏览器预览模式`:`Wails 桌面运行时`}function G(e){switch(e){case`checking`:return`正在检查更新`;case`available`:return`有新版本可用`;case`downloading`:return`正在下载更新`;case`downloaded`:return`更新包已下载`;case`installing`:return`更新安装已启动`;case`current`:return`当前已是最新版本`;case`unknown`:return`无法判断更新状态`;case`error`:return`检查更新失败`;default:return`尚未检查`}}function K(){return v.state===`checking`||v.state===`downloading`||v.state===`installing`}function me(){return v.available&&v.state!==`downloaded`&&!K()}function he(){return v.state===`downloaded`}function q(e){return`http://127.0.0.1:${e}`}function J(){let e=c.querySelector(`.log-filters`);if(!e)return p;let t=new FormData(e),n=String(t.get(`event`)??``).trim(),r=Y(String(t.get(`since`)??``)),i=Y(String(t.get(`until`)??``)),a={};return n&&(a.event=n),r&&(a.since=r),i&&(a.until=i),a}function Y(e){if(!e)return null;let t=new Date(e);return Number.isNaN(t.getTime())?null:t.toISOString()}function X(e){if(!e)return``;let t=new Date(e);return Number.isNaN(t.getTime())?``:new Date(t.getTime()-t.getTimezoneOffset()*6e4).toISOString().slice(0,16)}function ge(e){let t=new Date(e);return Number.isNaN(t.getTime())?e||`-`:new Intl.DateTimeFormat(void 0,{year:`numeric`,month:`2-digit`,day:`2-digit`,hour:`2-digit`,minute:`2-digit`,second:`2-digit`}).format(t)}function _e(){let e=Date.now(),t=t=>new Date(e-t*6e4).toISOString();return[{time:t(4),clientIP:`192.168.31.18`,method:`GET`,event:`access`,path:`/`,status:200,result:`ok`},{time:t(16),clientIP:`192.168.31.42`,method:`GET`,event:`download`,path:`/photos/2026/spring-trip/very-long-file-name-that-should-wrap-in-the-log-table-without-breaking-layout.jpg`,status:200,result:`sent`},{time:t(28),clientIP:`192.168.31.42`,method:`POST`,event:`upload`,path:`/uploads/report-final.pdf`,status:201,result:`created`},{time:t(44),clientIP:`192.168.31.9`,method:`POST`,event:`login`,path:`/login`,status:200,result:`authenticated`},{time:t(63),clientIP:`192.168.31.77`,method:`GET`,event:`reject`,path:`/private/<script>alert(1)<\/script>.txt`,status:403,result:`blocked`},{time:t(87),clientIP:`192.168.31.51`,method:`GET`,event:`error`,path:`/archive.zip`,status:500,result:`read failed`}]}function ve(e,t){let n=t.event?.trim(),r=Z(t.since),i=Z(t.until);return e.filter(e=>{let t=Z(e.time);return!(n&&e.event!==n||r!==null&&t!==null&&t<r||i!==null&&t!==null&&t>i)})}function Z(e){if(!e)return null;let t=new Date(e).getTime();return Number.isNaN(t)?null:t}function Q(e){return e.replace(/[&<>"']/g,e=>({"&":`&amp;`,"<":`&lt;`,">":`&gt;`,'"':`&quot;`,"'":`&#39;`})[e])}function $(e){return e instanceof Error?e.message:String(e)}