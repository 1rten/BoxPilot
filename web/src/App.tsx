import { useState } from "react";
import { BrowserRouter, Routes, Route, NavLink } from "react-router-dom";
import Dashboard from "./pages/Dashboard";
import Subscriptions from "./pages/Subscriptions";
import Nodes from "./pages/Nodes";
import Settings from "./pages/Settings";
import type { RuntimeGroupItem, RuntimeProxyCheckItem } from "./api/types";
import {
  useForwardingSummary,
  useStartForwardingRuntime,
  useStopForwardingRuntime,
} from "./hooks/useProxySettings";
import { Button, Input, Popover, Tag } from "antd";
import { useI18n } from "./i18n/context";
import { useRuntimeGroups, useRuntimeProxyCheck } from "./hooks/useRuntime";
import { formatDateTime } from "./utils/datetime";

export default function App() {
  const { locale, setLocale, tr } = useI18n();
  const [localeOpen, setLocaleOpen] = useState(false);
  const { data: summary, refetch: refetchForwardingSummary } = useForwardingSummary();
  const { data: runtimeGroups, refetch: refetchRuntimeGroups } = useRuntimeGroups();
  const startForwarding = useStartForwardingRuntime();
  const stopForwarding = useStopForwardingRuntime();
  const proxyCheck = useRuntimeProxyCheck();
  const [proxyCheckTarget, setProxyCheckTarget] = useState("https://www.gstatic.com/generate_204");
  const toggling = startForwarding.isPending || stopForwarding.isPending;
  const isRunning = !!summary?.running;
  const runtimeStatus = summary?.status ?? "stopped";
  const runtimeStatusLabel = tr(`app.proxy.runtime.${runtimeStatus}`, runtimeStatus.toUpperCase());
  const groups = (runtimeGroups?.items ?? []).filter((item) => item.tag === "manual" || item.tag.startsWith("biz-"));
  const proxyCheckResult = proxyCheck.data;

  return (
    <BrowserRouter>
      <div className="bp-shell">
        <nav className="bp-nav">
          <div className="bp-nav-left">
            <div className="bp-brand">
              <div className="bp-brand-mark" aria-hidden="true">
                <img src="/favicon.svg" alt="" />
              </div>
              <div className="bp-brand-name">BoxPilot</div>
            </div>
            <div className="bp-tabs">
              <NavLink
                to="/"
                end
                className={({ isActive }) =>
                  isActive ? "bp-tab bp-tab-active" : "bp-tab"
                }
              >
                {tr("nav.dashboard", "Dashboard")}
              </NavLink>
              <NavLink
                to="/subscriptions"
                className={({ isActive }) =>
                  isActive ? "bp-tab bp-tab-active" : "bp-tab"
                }
              >
                {tr("nav.subscriptions", "Subscriptions")}
              </NavLink>
              <NavLink
                to="/nodes"
                className={({ isActive }) =>
                  isActive ? "bp-tab bp-tab-active" : "bp-tab"
                }
              >
                {tr("nav.nodes", "Nodes")}
              </NavLink>
            </div>
          </div>
          <div className="bp-nav-right">
            <Popover
              placement="bottomRight"
              trigger={["click"]}
              open={localeOpen}
              onOpenChange={setLocaleOpen}
              getPopupContainer={() => document.body}
              overlayClassName="bp-ant-overlay-fix"
              content={
                <div className="bp-locale-menu">
                  <button
                    type="button"
                    className={locale === "zh" ? "bp-locale-option bp-locale-option-active" : "bp-locale-option"}
                    onClick={() => {
                      setLocale("zh");
                      setLocaleOpen(false);
                    }}
                  >
                    <span>{tr("nav.language.zh", "中文")}</span>
                    {locale === "zh" ? <span>✓</span> : null}
                  </button>
                  <button
                    type="button"
                    className={locale === "en" ? "bp-locale-option bp-locale-option-active" : "bp-locale-option"}
                    onClick={() => {
                      setLocale("en");
                      setLocaleOpen(false);
                    }}
                  >
                    <span>{tr("nav.language.en", "English")}</span>
                    {locale === "en" ? <span>✓</span> : null}
                  </button>
                </div>
              }
            >
              <button
                type="button"
                className="bp-locale-trigger"
                aria-label={tr("nav.language", "Language")}
                title={tr("nav.language", "Language")}
                aria-haspopup="menu"
                aria-expanded={localeOpen}
              >
                <svg viewBox="0 0 24 24" aria-hidden="true">
                  <path d="M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20Zm6.94 9h-3.03a15.3 15.3 0 0 0-1.2-5.03A8.02 8.02 0 0 1 18.94 11Zm-6.94-7a13.3 13.3 0 0 1 1.95 7h-3.9A13.3 13.3 0 0 1 12 4ZM4.06 13h3.03c.14 1.78.55 3.51 1.2 5.03A8.02 8.02 0 0 1 4.06 13Zm3.03-2H4.06a8.02 8.02 0 0 1 4.23-5.03A15.3 15.3 0 0 0 7.09 11Zm4.91 9a13.3 13.3 0 0 1-1.95-7h3.9A13.3 13.3 0 0 1 12 20Zm.71-2.97A15.3 15.3 0 0 0 15.91 13h3.03a8.02 8.02 0 0 1-4.23 4.03Z" />
                </svg>
                <span>{locale.toUpperCase()}</span>
              </button>
            </Popover>
            <Popover
              placement="bottomRight"
              trigger={["hover", "click"]}
              getPopupContainer={() => document.body}
              overlayClassName="bp-ant-overlay-fix"
              onOpenChange={(open) => {
                if (open) {
                  void refetchForwardingSummary();
                  void refetchRuntimeGroups();
                }
              }}
              content={
                <div className="bp-forwarding-popover">
                  <div className="bp-forwarding-popover-head">
                    <span className="bp-forwarding-popover-title">{tr("nav.proxy", "Proxy")}</span>
                    <span className={`bp-runtime-dot bp-runtime-dot-${runtimeStatus}`} />
                  </div>
                  <p className="bp-forwarding-popover-meta">
                    {tr("app.proxy.status", "Status")}: {runtimeStatusLabel} · {tr("app.proxy.selected", "Selected")}: {summary?.selected_nodes_count ?? 0}
                  </p>
                  {summary?.error_message && (
                    <p className="bp-forwarding-popover-error">{summary.error_message}</p>
                  )}
                  {groups.length > 0 ? (
                    <div className="bp-forwarding-popover-list">
                      {groups.map((group: RuntimeGroupItem) => {
                        const selected = group.runtime_selected_outbound ?? group.default;
                        const effective = group.runtime_effective_outbound;
                        const isAuto = !!group.auto_outbound && selected === group.auto_outbound;
                        return (
                          <div key={group.tag} className="bp-forwarding-popover-item">
                            <span className="bp-forwarding-popover-name">{formatRuntimeGroupLabel(group.tag)}</span>
                            <div className="bp-forwarding-popover-side">
                              <Tag color={isAuto ? "processing" : "default"}>
                                {isAuto
                                  ? tr("app.proxy.group.mode_auto", "AUTO")
                                  : tr("app.proxy.group.mode_manual", "MANUAL")}
                              </Tag>
                              <span className="bp-table-mono">
                                {tr("app.proxy.group.selected", "Selected")}: {selected}
                              </span>
                              {effective && effective !== selected ? (
                                <span className="bp-table-mono">
                                  {tr("app.proxy.group.effective", "Effective")}: {effective}
                                </span>
                              ) : null}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  ) : (
                    <p className="bp-forwarding-popover-empty">{tr("app.proxy.groups.empty", "No business routing groups.")}</p>
                  )}
                  <div className="bp-forwarding-popover-section">
                    <div className="bp-forwarding-popover-head">
                      <span className="bp-forwarding-popover-title">
                        {tr("app.proxy.check.title", "Proxy Chain Check")}
                      </span>
                      {proxyCheckResult?.checked_at ? (
                        <span className="bp-forwarding-popover-time">{formatDateTime(proxyCheckResult.checked_at)}</span>
                      ) : null}
                    </div>
                    <div className="bp-forwarding-popover-toolbar">
                      <Input
                        value={proxyCheckTarget}
                        onChange={(e) => setProxyCheckTarget(e.target.value)}
                        placeholder={tr("app.proxy.check.target.placeholder", "https://www.gstatic.com/generate_204")}
                        className="bp-input"
                      />
                      <Button
                        type="primary"
                        size="small"
                        loading={proxyCheck.isPending}
                        onClick={() =>
                          proxyCheck.mutate({
                            target_url: proxyCheckTarget.trim() || undefined,
                          })
                        }
                      >
                        {tr("app.proxy.check.run", "Check")}
                      </Button>
                    </div>
                    <div className="bp-forwarding-popover-check-grid">
                      <ProxyMiniResult
                        label="HTTP"
                        data={proxyCheckResult?.http}
                        labels={{
                          pass: tr("app.proxy.check.pass", "PASS"),
                          fail: tr("app.proxy.check.fail", "FAIL"),
                          disabled: tr("app.proxy.check.disabled", "DISABLED"),
                          tls_ok: tr("app.proxy.check.tls_ok", "TLS OK"),
                          tls_na: tr("app.proxy.check.tls_na", "TLS -"),
                        }}
                      />
                      <ProxyMiniResult
                        label="SOCKS5"
                        data={proxyCheckResult?.socks}
                        labels={{
                          pass: tr("app.proxy.check.pass", "PASS"),
                          fail: tr("app.proxy.check.fail", "FAIL"),
                          disabled: tr("app.proxy.check.disabled", "DISABLED"),
                          tls_ok: tr("app.proxy.check.tls_ok", "TLS OK"),
                          tls_na: tr("app.proxy.check.tls_na", "TLS -"),
                        }}
                      />
                    </div>
                  </div>
                </div>
              }
            >
              <button
                type="button"
                className={
                  runtimeStatus === "running"
                    ? "bp-forwarding-toggle bp-forwarding-toggle-running"
                    : runtimeStatus === "error"
                    ? "bp-forwarding-toggle bp-forwarding-toggle-error"
                    : "bp-forwarding-toggle"
                }
                disabled={toggling}
                onClick={() => {
                  if (isRunning) {
                    stopForwarding.mutate();
                  } else {
                    startForwarding.mutate();
                  }
                }}
                title={summary?.error_message || tr("app.proxy.control", "Forwarding runtime control")}
              >
                <span className={`bp-runtime-dot bp-runtime-dot-${runtimeStatus}`} />
                {toggling ? tr("nav.proxy.applying", "Applying...") : tr("nav.proxy", "Proxy")}
              </button>
            </Popover>
            <NavLink
              to="/settings"
              className={({ isActive }) =>
                isActive ? "bp-settings-link bp-settings-link-active" : "bp-settings-link"
              }
              aria-label={tr("nav.settings", "Settings")}
              title={tr("nav.settings", "Settings")}
            >
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <path d="M19.14 12.94a7.8 7.8 0 0 0 .05-.94 7.8 7.8 0 0 0-.05-.94l2.03-1.58a.5.5 0 0 0 .12-.65l-1.92-3.32a.5.5 0 0 0-.61-.22l-2.39.96a7.4 7.4 0 0 0-1.62-.94l-.36-2.54a.5.5 0 0 0-.5-.43h-3.84a.5.5 0 0 0-.5.43l-.36 2.54a7.4 7.4 0 0 0-1.62.94l-2.39-.96a.5.5 0 0 0-.61.22L2.71 8.83a.5.5 0 0 0 .12.65l2.03 1.58a7.8 7.8 0 0 0-.05.94 7.8 7.8 0 0 0 .05.94l-2.03 1.58a.5.5 0 0 0-.12.65l1.92 3.32a.5.5 0 0 0 .61.22l2.39-.96c.5.39 1.05.71 1.62.94l.36 2.54a.5.5 0 0 0 .5.43h3.84a.5.5 0 0 0 .5-.43l.36-2.54c.57-.23 1.12-.55 1.62-.94l2.39.96a.5.5 0 0 0 .61-.22l1.92-3.32a.5.5 0 0 0-.12-.65zM12 15.2a3.2 3.2 0 1 1 0-6.4 3.2 3.2 0 0 1 0 6.4z" />
              </svg>
            </NavLink>
          </div>
        </nav>
        <main className="bp-main">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/subscriptions" element={<Subscriptions />} />
            <Route path="/nodes" element={<Nodes />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}

function formatRuntimeGroupLabel(tag: string): string {
  if (tag === "manual") {
    return "Manual";
  }
  if (tag.startsWith("biz-")) {
    const body = tag.slice(4).replace(/-/g, " ").trim();
    if (!body) {
      return tag;
    }
    return body.replace(/\b\w/g, (c) => c.toUpperCase());
  }
  return tag;
}

function ProxyMiniResult({
  label,
  data,
  labels,
}: {
  label: string;
  data?: RuntimeProxyCheckItem;
  labels: {
    pass: string;
    fail: string;
    disabled: string;
    tls_ok: string;
    tls_na: string;
  };
}) {
  if (!data) {
    return (
      <div className="bp-forwarding-popover-check-item">
        <div className="bp-forwarding-popover-check-head">
          <span className="bp-forwarding-popover-check-label">{label}</span>
          <Tag bordered={false}>-</Tag>
        </div>
        <p className="bp-forwarding-popover-empty">-</p>
      </div>
    );
  }

  const tone = !data.enabled ? undefined : data.connected ? "success" : "error";
  const statusText = data.enabled ? (data.connected ? labels.pass : labels.fail) : labels.disabled;
  return (
    <div className="bp-forwarding-popover-check-item">
      <div className="bp-forwarding-popover-check-head">
        <span className="bp-forwarding-popover-check-label">{label}</span>
        <Tag color={tone}>{statusText}</Tag>
      </div>
      <div className="bp-forwarding-popover-check-meta">
        <span className="bp-table-mono">{data.proxy_url}</span>
        <span>{data.status_code ?? "-"}</span>
        <span>{data.latency_ms != null ? `${data.latency_ms}ms` : "-"}</span>
        <span>{data.tls_ok ? labels.tls_ok : labels.tls_na}</span>
      </div>
      {data.error ? (
        <p className="bp-forwarding-popover-error" title={data.error}>
          {data.error}
        </p>
      ) : null}
    </div>
  );
}
