import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Segmented,
  Select,
  Switch,
  Tag,
} from "antd";
import type { ProxyConfig, ProxyType, RoutingSettingsData, RuntimeGroupItem } from "../api/types";
import { buildProxyUrl, resolveProxyClientHost } from "../api/settings";
import {
  useProxySettings,
  useUpdateProxySettings,
  useApplyProxySettings,
  useRoutingSettings,
  useUpdateRoutingSettings,
  useForwardingPolicy,
  useUpdateForwardingPolicy,
} from "../hooks/useProxySettings";
import { useRuntimeGroups, useSelectRuntimeGroup } from "../hooks/useRuntime";
import { useToast } from "../components/common/ToastContext";
import { useI18n } from "../i18n/context";

const selectPopupClassNames = { popup: { root: "bp-ant-overlay-fix" } } as const;
const getOverlayContainer = (triggerNode: HTMLElement) =>
  triggerNode.parentElement ?? document.body;

interface ProxyCardProps {
  title: string;
  proxyType: ProxyType;
  data?: ProxyConfig;
  onSaved?: () => void;
}

type SettingsSection = "access" | "routing" | "runtime";

export default function Settings() {
  const { tr } = useI18n();
  const [section, setSection] = useState<SettingsSection>("access");
  const [pendingApply, setPendingApply] = useState(false);
  const {
    data,
    isLoading,
    isFetching: proxyFetching,
    refetch: refetchProxySettings,
  } = useProxySettings();
  const {
    data: routingData,
    isLoading: routingLoading,
    isFetching: routingFetching,
    refetch: refetchRoutingSettings,
  } = useRoutingSettings();
  const {
    data: forwardingPolicy,
    isLoading: forwardingPolicyLoading,
    isFetching: forwardingPolicyFetching,
    refetch: refetchForwardingPolicy,
  } = useForwardingPolicy();
  const {
    data: runtimeGroups,
    isLoading: runtimeGroupsLoading,
    isFetching: runtimeGroupsFetching,
    isError: runtimeGroupsIsError,
    error: runtimeGroupsError,
    refetch: refetchRuntimeGroups,
  } = useRuntimeGroups();
  const applyAll = useApplyProxySettings();

  const refreshing =
    proxyFetching || routingFetching || forwardingPolicyFetching || runtimeGroupsFetching;
  const runtimeGroupsErrorMessage = runtimeGroupsIsError
    ? formatQueryError(runtimeGroupsError, tr("toast.unknown", "Unknown error"))
    : "";
  const showSectionLoading =
    section === "access"
      ? isLoading
      : section === "routing"
        ? routingLoading || forwardingPolicyLoading
        : false;

  const onRefreshAll = async () =>
    Promise.all([
      refetchProxySettings(),
      refetchRoutingSettings(),
      refetchForwardingPolicy(),
      refetchRuntimeGroups(),
    ]);

  const onApplyAll = async () => {
    try {
      await applyAll.mutateAsync();
      setPendingApply(false);
    } finally {
      await onRefreshAll();
    }
  };

  const markPendingApply = () => setPendingApply(true);

  return (
    <div className="bp-page">
      <div className="bp-page-header">
        <div>
          <h1 className="bp-page-title">{tr("settings.title", "Settings")}</h1>
          <p className="bp-page-subtitle">
            {tr("settings.subtitle", "Configure global HTTP and SOCKS5 forwarding behavior.")}
          </p>
        </div>
        <div className="bp-page-actions bp-page-actions--header">
          {pendingApply ? (
            <Tag color="warning">
              {tr("settings.pending_apply", "Pending restart to take effect")}
            </Tag>
          ) : null}
          <Button
            className="bp-btn-fixed"
            type="primary"
            loading={applyAll.isPending}
            onClick={() => void onApplyAll()}
          >
            {tr("settings.apply_all", "Apply Changes / Restart")}
          </Button>
          <Button className="bp-btn-fixed" loading={refreshing} onClick={() => void onRefreshAll()}>
            {tr("common.refresh", "Refresh")}
          </Button>
        </div>
      </div>
      {pendingApply ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          title={tr("settings.pending_apply_title", "Configuration saved")}
          description={tr(
            "settings.pending_apply_desc",
            "Some changes are saved to DB and will take effect after applying runtime.",
          )}
        />
      ) : null}
      <Segmented
        style={{ marginBottom: 16 }}
        value={section}
        onChange={(value) => setSection(value as SettingsSection)}
        options={[
          { label: tr("settings.section.access", "Access"), value: "access" },
          { label: tr("settings.section.routing", "Routing"), value: "routing" },
          { label: tr("settings.section.runtime", "Runtime"), value: "runtime" },
        ]}
      />
      {section === "access" ? (
        <div className="bp-settings-grid">
          <ProxySettingsCard
            title={tr("settings.http.title", "HTTP Proxy")}
            proxyType="http"
            data={data?.http}
            onSaved={markPendingApply}
          />
          <ProxySettingsCard
            title={tr("settings.socks.title", "SOCKS5 Proxy")}
            proxyType="socks"
            data={data?.socks}
            onSaved={markPendingApply}
          />
        </div>
      ) : null}
      {section === "routing" ? (
        <>
          <div>
            <ForwardingPolicyCard data={forwardingPolicy} onSaved={markPendingApply} />
          </div>
          <div style={{ marginTop: 16 }}>
            <RoutingSettingsCard data={routingData} onSaved={markPendingApply} />
          </div>
        </>
      ) : null}
      {section === "runtime" ? (
        <div>
          <RuntimeGroupsCard
            items={runtimeGroups?.items}
            autoIntervalSec={forwardingPolicy?.biz_auto_interval_sec}
            isLoading={runtimeGroupsLoading && !runtimeGroups}
            isFetching={runtimeGroupsFetching}
            errorMessage={runtimeGroupsErrorMessage}
            onGoPolicy={() => setSection("routing")}
          />
        </div>
      ) : null}
      {showSectionLoading && (
        <p className="bp-muted" style={{ marginTop: 12 }}>
          {tr("common.loading", "Loading...")}
        </p>
      )}
    </div>
  );
}

interface RuntimeGroupsCardProps {
  items?: RuntimeGroupItem[];
  autoIntervalSec?: number;
  isLoading?: boolean;
  isFetching?: boolean;
  errorMessage?: string;
  onGoPolicy?: () => void;
}

function RuntimeGroupsCard({
  items,
  autoIntervalSec,
  isLoading,
  isFetching,
  errorMessage,
  onGoPolicy,
}: RuntimeGroupsCardProps) {
  const { tr } = useI18n();
  const updateGroup = useSelectRuntimeGroup();
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [dirtyTags, setDirtyTags] = useState<Record<string, boolean>>({});
  const [applyingTag, setApplyingTag] = useState("");

  const groups = useMemo(
    () => (items ?? []).filter((item) => item.tag === "manual" || item.tag.startsWith("biz-")),
    [items],
  );
  const manualGroup = useMemo(() => groups.find((item) => item.tag === "manual"), [groups]);
  const businessGroups = useMemo(
    () => groups.filter((item) => item.tag.startsWith("biz-")),
    [groups],
  );

  useEffect(() => {
    setDrafts((prev) => {
      const next: Record<string, string> = {};
      for (const item of groups) {
        const upstreamValue = item.runtime_selected_outbound ?? item.default;
        const hasDraft = Object.prototype.hasOwnProperty.call(prev, item.tag);
        if (dirtyTags[item.tag] && hasDraft) {
          next[item.tag] = prev[item.tag];
          continue;
        }
        next[item.tag] = upstreamValue;
      }
      const prevKeys = Object.keys(prev);
      const nextKeys = Object.keys(next);
      if (prevKeys.length === nextKeys.length) {
        let changed = false;
        for (const key of nextKeys) {
          if (prev[key] !== next[key]) {
            changed = true;
            break;
          }
        }
        if (!changed) {
          return prev;
        }
      }
      return next;
    });
  }, [groups, dirtyTags]);

  useEffect(() => {
    setDirtyTags((prev) => {
      const next: Record<string, boolean> = {};
      for (const item of groups) {
        if (prev[item.tag]) {
          next[item.tag] = true;
        }
      }
      const prevKeys = Object.keys(prev);
      const nextKeys = Object.keys(next);
      if (prevKeys.length === nextKeys.length) {
        let changed = false;
        for (const key of nextKeys) {
          if (prev[key] !== next[key]) {
            changed = true;
            break;
          }
        }
        if (!changed) {
          return prev;
        }
      }
      return next;
    });
  }, [groups]);

  const updateDraft = (tag: string, value: string) => {
    setDrafts((prev) => ({ ...prev, [tag]: value }));
    setDirtyTags((prev) => ({ ...prev, [tag]: true }));
  };

  const applyGroupChoice = async (groupTag: string, selectedOutbound: string) => {
    setApplyingTag(groupTag);
    try {
      await updateGroup.mutateAsync({
        group_tag: groupTag,
        selected_outbound: selectedOutbound,
      });
      setDirtyTags((prev) => {
        const next = { ...prev };
        delete next[groupTag];
        return next;
      });
      setDrafts((prev) => ({ ...prev, [groupTag]: selectedOutbound }));
    } finally {
      setApplyingTag("");
    }
  };

  return (
    <Card className="bp-settings-card">
      <div className="bp-card-header">
        <div>
          <p className="bp-card-kicker">{tr("settings.groups.kicker", "Runtime Groups")}</p>
          <h2 className="bp-card-title">
            {tr("settings.groups.title", "Business Routing Groups")}
          </h2>
        </div>
      </div>
      <p className="bp-muted" style={{ marginTop: 0 }}>
        {tr(
          "settings.groups.desc",
          "Manual fallback and each business group support auto toggle: on = trigger urltest now and then by configured interval, off = manual node pick in that group pool.",
        )}
      </p>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 12,
          marginBottom: 8,
        }}
      >
        <span className="bp-muted">
          {tr("settings.groups.auto_interval_value", "Current auto interval: {value}", {
            value: formatSecondsInterval(autoIntervalSec),
          })}
        </span>
        <Button size="small" onClick={onGoPolicy}>
          {tr("settings.groups.go_policy", "Go to Forwarding Policy")}
        </Button>
      </div>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        title={tr(
          "settings.groups.mapping_note",
          "Business candidates come from explicit members in subscription business groups. Generic groups like manual/proxy are not expanded.",
        )}
        description={tr(
          "settings.groups.refresh_note",
          "If subscription rules changed, refresh subscription first, then re-open this page.",
        )}
      />
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        title={tr(
          "settings.groups.auto_probe_note",
          "When Auto is enabled and applied, runtime triggers a delay test on the auto group with https://www.gstatic.com/generate_204.",
        )}
        description={tr(
          "settings.groups.auto_interval_note",
          "Recurring auto test interval is configured in Forwarding Policy.",
        )}
      />
      {isFetching && !isLoading ? (
        <p className="bp-muted" style={{ margin: "0 0 12px" }}>
          {tr("settings.groups.refreshing", "Refreshing runtime groups...")}
        </p>
      ) : null}
      {errorMessage ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          title={tr("settings.groups.load_error", "Failed to load runtime groups")}
          description={errorMessage}
        />
      ) : null}
      {isLoading ? (
        <p className="bp-muted">{tr("settings.groups.loading", "Loading runtime groups...")}</p>
      ) : null}
      {!isLoading && groups.length === 0 ? (
        <p className="bp-muted">
          {tr("settings.groups.empty", "No runtime groups available yet.")}
        </p>
      ) : (
        <div style={{ display: "grid", gap: 12 }}>
          {manualGroup ? (
            <div
              key={manualGroup.tag}
              style={{
                border: "1px solid var(--bp-border)",
                borderRadius: 12,
                padding: 12,
                display: "grid",
                gap: 8,
              }}
            >
              {(() => {
                const draftValue =
                  drafts[manualGroup.tag] ??
                  manualGroup.runtime_selected_outbound ??
                  manualGroup.default;
                const currentValue = manualGroup.runtime_selected_outbound ?? manualGroup.default;
                const autoOutbound = manualGroup.auto_outbound;
                const autoEnabled = Boolean(autoOutbound && draftValue === autoOutbound);
                const manualNodeCandidates = dedupeOutbounds(
                  manualGroup.node_candidates ?? manualGroup.outbounds.filter((v) => v !== autoOutbound),
                );
                const manualNodeOptions = manualNodeCandidates.map((value) => ({
                  value,
                  label: formatOutboundOptionLabel(value, tr),
                }));
                const safeDraftValue = manualNodeCandidates.includes(draftValue)
                  ? draftValue
                  : (manualNodeCandidates[0] ?? "");
                const manualExitValue = resolveManualExitValue(
                  manualNodeCandidates,
                  manualGroup.runtime_effective_outbound,
                );
                const effectiveDraftValue = autoEnabled ? draftValue : safeDraftValue;
                return (
                  <>
                    <div style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
                      <strong>{tr("settings.groups.manual_title", "Manual Fallback Group")}</strong>
                      <Tag>{manualGroup.tag}</Tag>
                    </div>
                    <p className="bp-muted" style={{ margin: 0 }}>
                      {tr(
                        "settings.groups.manual_desc",
                        "Shared manual fallback. Business groups can choose manual to follow this selection.",
                      )}
                    </p>
                    {manualGroup.persisted_selected_outbound ? (
                      <p className="bp-muted" style={{ margin: 0 }}>
                        {tr("settings.groups.persisted", "Saved: {outbound}", {
                          outbound: manualGroup.persisted_selected_outbound,
                        })}
                        {manualGroup.persisted_updated_at
                          ? ` · ${tr("settings.groups.persisted_at", "Updated {time}", { time: manualGroup.persisted_updated_at })}`
                          : ""}
                      </p>
                    ) : null}
                    {manualGroup.runtime_selected_outbound ? (
                      <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
                        <Tag color="processing">
                          {tr("settings.groups.runtime_selected", "Runtime selected: {outbound}", {
                            outbound: manualGroup.runtime_selected_outbound,
                          })}
                        </Tag>
                        {manualGroup.runtime_effective_outbound ? (
                          <Tag
                            color={
                              manualGroup.runtime_effective_outbound ===
                              manualGroup.runtime_selected_outbound
                                ? "blue"
                                : "success"
                            }
                          >
                            {tr("settings.groups.runtime_effective", "Effective: {outbound}", {
                              outbound: manualGroup.runtime_effective_outbound,
                            })}
                          </Tag>
                        ) : null}
                      </div>
                    ) : null}
                    {manualGroup.auto_candidates && manualGroup.auto_candidates.length > 0 ? (
                      <details>
                        <summary className="bp-muted" style={{ cursor: "pointer" }}>
                          {tr("settings.groups.auto_candidates_toggle", "View auto candidates ({count})", {
                            count: String(manualGroup.auto_candidates.length),
                          })}
                        </summary>
                        <div style={{ display: "flex", flexWrap: "wrap", gap: 6, marginTop: 8 }}>
                          {manualGroup.auto_candidates.map((nodeTag) => (
                            <Tag key={`${manualGroup.tag}-${nodeTag}`}>{nodeTag}</Tag>
                          ))}
                        </div>
                      </details>
                    ) : null}
                    <div
                      style={{
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "space-between",
                        gap: 12,
                      }}
                    >
                      <span className="bp-muted">
                        {tr("settings.groups.auto_toggle", "Auto Best Node")}
                      </span>
                      <Switch
                        checked={autoEnabled}
                        disabled={!autoOutbound}
                        onChange={(checked) => {
                          if (checked && autoOutbound) {
                            updateDraft(manualGroup.tag, autoOutbound);
                            return;
                          }
                          if (manualExitValue) {
                            updateDraft(manualGroup.tag, manualExitValue);
                          }
                        }}
                      />
                    </div>
                    {autoEnabled ? (
                      <div className="bp-runtime-choice-panel">
                        <div className="bp-runtime-choice-panel-head">
                          <strong>{tr("settings.groups.auto_active_title", "Auto mode is active")}</strong>
                          {autoOutbound ? (
                            <Tag color="processing">{formatOutboundOptionLabel(autoOutbound, tr)}</Tag>
                          ) : null}
                        </div>
                        <p className="bp-muted" style={{ margin: 0 }}>
                          {tr(
                            "settings.groups.manual_auto_active_desc",
                            "Manual fallback now follows the auto-tested best node until you switch back to manual selection.",
                          )}
                        </p>
                        <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
                          {manualGroup.runtime_selected_outbound ? (
                            <Tag color="processing">
                              {tr("settings.groups.runtime_selected", "Runtime selected: {outbound}", {
                                outbound: manualGroup.runtime_selected_outbound,
                              })}
                            </Tag>
                          ) : null}
                          {manualGroup.runtime_effective_outbound ? (
                            <Tag color="success">
                              {tr("settings.groups.runtime_effective", "Effective: {outbound}", {
                                outbound: manualGroup.runtime_effective_outbound,
                              })}
                            </Tag>
                          ) : null}
                        </div>
                        {manualExitValue ? (
                          <div
                            className="bp-page-actions bp-settings-actions"
                            style={{ marginTop: 0, justifyContent: "flex-start" }}
                          >
                            <Button size="small" onClick={() => updateDraft(manualGroup.tag, manualExitValue)}>
                              {tr("settings.groups.switch_manual", "Switch to Manual Selection")}
                            </Button>
                          </div>
                        ) : null}
                      </div>
                    ) : manualNodeOptions.length > 0 ? (
                      <select
                        className="bp-native-select"
                        value={safeDraftValue}
                        onChange={(e) => updateDraft(manualGroup.tag, e.target.value)}
                      >
                        {manualNodeOptions.map((option) => (
                          <option key={`${manualGroup.tag}-${option.value}`} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <p className="bp-muted" style={{ margin: 0 }}>
                        {tr("settings.groups.manual_node_empty", "No nodes available for manual selection.")}
                      </p>
                    )}
                    <div className="bp-page-actions bp-settings-actions">
                      <Button
                        className="bp-btn-fixed"
                        type="primary"
                        loading={applyingTag === manualGroup.tag}
                        disabled={
                          (applyingTag.length > 0 && applyingTag !== manualGroup.tag) ||
                          !effectiveDraftValue ||
                          effectiveDraftValue === currentValue
                        }
                        onClick={() => void applyGroupChoice(manualGroup.tag, effectiveDraftValue)}
                      >
                        {tr("settings.groups.apply", "Apply Group Choice")}
                      </Button>
                    </div>
                  </>
                );
              })()}
            </div>
          ) : null}
          {businessGroups.length > 0 ? (
            <div style={{ display: "grid", gap: 8 }}>
              {businessGroups.map((group) => {
                const draftValue =
                  drafts[group.tag] ?? group.runtime_selected_outbound ?? group.default;
                const currentValue = group.runtime_selected_outbound ?? group.default;
                const autoOutbound = group.auto_outbound;
                const autoEnabled = Boolean(autoOutbound && draftValue === autoOutbound);
                const nodeCandidates = dedupeOutbounds(
                  group.node_candidates ?? group.outbounds.filter((v) => v !== autoOutbound),
                );
                const nodeOptions = nodeCandidates.map((value) => ({
                  value,
                  label: formatOutboundOptionLabel(value, tr),
                }));
                const safeNodeDraftValue = nodeCandidates.includes(draftValue)
                  ? draftValue
                  : (nodeCandidates[0] ?? "");
                const effectiveDraftValue = autoEnabled ? draftValue : safeNodeDraftValue;
                const hasBusinessNodes = nodeCandidates.some((value) => value !== "manual");
                const manualExitValue = resolveManualExitValue(
                  nodeCandidates,
                  group.runtime_effective_outbound,
                );
                return (
                  <details key={group.tag}>
                    <summary
                      style={{
                        cursor: "pointer",
                        border: "1px solid var(--bp-border)",
                        borderRadius: 12,
                        padding: 10,
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                        gap: 8,
                      }}
                    >
                      <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
                        <strong>{formatGroupLabel(group.tag, tr)}</strong>
                        <Tag>{group.tag}</Tag>
                        <Tag color={autoEnabled ? "success" : "default"}>
                          {autoEnabled
                            ? tr("settings.groups.mode_auto", "AUTO")
                            : tr("settings.groups.mode_manual", "MANUAL")}
                        </Tag>
                      </span>
                      {group.runtime_effective_outbound ? (
                        <Tag color="processing">
                          {tr("settings.groups.runtime_effective", "Effective: {outbound}", {
                            outbound: group.runtime_effective_outbound,
                          })}
                        </Tag>
                      ) : null}
                    </summary>
                    <div
                      style={{
                        marginTop: 8,
                        border: "1px solid var(--bp-border)",
                        borderRadius: 12,
                        padding: 12,
                        display: "grid",
                        gap: 8,
                      }}
                    >
                      {group.persisted_selected_outbound ? (
                        <p className="bp-muted" style={{ margin: 0 }}>
                          {tr("settings.groups.persisted", "Saved: {outbound}", {
                            outbound: group.persisted_selected_outbound,
                          })}
                          {group.persisted_updated_at
                            ? ` · ${tr("settings.groups.persisted_at", "Updated {time}", { time: group.persisted_updated_at })}`
                            : ""}
                        </p>
                      ) : null}
                      {group.runtime_selected_outbound ? (
                        <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
                          <Tag color="processing">
                            {tr(
                              "settings.groups.runtime_selected",
                              "Runtime selected: {outbound}",
                              {
                                outbound: group.runtime_selected_outbound,
                              },
                            )}
                          </Tag>
                          {group.runtime_effective_outbound ? (
                            <Tag
                              color={
                                group.runtime_effective_outbound === group.runtime_selected_outbound
                                  ? "blue"
                                  : "success"
                              }
                            >
                              {tr("settings.groups.runtime_effective", "Effective: {outbound}", {
                                outbound: group.runtime_effective_outbound,
                              })}
                            </Tag>
                          ) : null}
                        </div>
                      ) : (
                        <p className="bp-muted" style={{ margin: 0 }}>
                          {tr(
                            "settings.groups.runtime_unavailable",
                            "Runtime state unavailable (Clash API unreachable).",
                          )}
                        </p>
                      )}
                      {group.auto_candidates && group.auto_candidates.length > 0 ? (
                        <details>
                          <summary className="bp-muted" style={{ cursor: "pointer" }}>
                            {tr(
                              "settings.groups.auto_candidates_toggle",
                              "View auto candidates ({count})",
                              {
                                count: String(group.auto_candidates.length),
                              },
                            )}
                          </summary>
                          <div style={{ display: "flex", flexWrap: "wrap", gap: 6, marginTop: 8 }}>
                            {group.auto_candidates.map((nodeTag) => (
                              <Tag key={`${group.tag}-${nodeTag}`}>{nodeTag}</Tag>
                            ))}
                          </div>
                        </details>
                      ) : (
                        <p className="bp-muted" style={{ margin: 0 }}>
                          {tr(
                            "settings.groups.auto_empty",
                            "No business node pool was parsed for this group, so only manual is available.",
                          )}
                        </p>
                      )}
                      <div style={{ display: "grid", gap: 8 }}>
                        <div
                          style={{
                            display: "flex",
                            alignItems: "center",
                            justifyContent: "space-between",
                            gap: 12,
                          }}
                        >
                          <span className="bp-muted">
                            {tr("settings.groups.auto_toggle", "Auto Best Node")}
                          </span>
                          <Switch
                            checked={autoEnabled}
                            disabled={!autoOutbound}
                            onChange={(checked) => {
                              if (checked && autoOutbound) {
                                updateDraft(group.tag, autoOutbound);
                                return;
                              }
                              if (manualExitValue) {
                                updateDraft(group.tag, manualExitValue);
                                return;
                              }
                            }}
                          />
                        </div>
                        {autoEnabled ? (
                          <div className="bp-runtime-choice-panel">
                            <div className="bp-runtime-choice-panel-head">
                              <strong>
                                {tr("settings.groups.auto_active_title", "Auto mode is active")}
                              </strong>
                              {autoOutbound ? <Tag color="processing">{autoOutbound}</Tag> : null}
                            </div>
                            <p className="bp-muted" style={{ margin: 0 }}>
                              {tr(
                                "settings.groups.auto_active_desc",
                                "Runtime will keep following the auto-tested best node until you switch back to manual selection.",
                              )}
                            </p>
                            <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
                              {group.runtime_selected_outbound ? (
                                <Tag color="processing">
                                  {tr(
                                    "settings.groups.runtime_selected",
                                    "Runtime selected: {outbound}",
                                    {
                                      outbound: group.runtime_selected_outbound,
                                    },
                                  )}
                                </Tag>
                              ) : null}
                              {group.runtime_effective_outbound ? (
                                <Tag color="success">
                                  {tr(
                                    "settings.groups.runtime_effective",
                                    "Effective: {outbound}",
                                    {
                                      outbound: group.runtime_effective_outbound,
                                    },
                                  )}
                                </Tag>
                              ) : null}
                            </div>
                            {manualExitValue ? (
                              <div
                                className="bp-page-actions bp-settings-actions"
                                style={{ marginTop: 0, justifyContent: "flex-start" }}
                              >
                                <Button
                                  size="small"
                                  onClick={() => updateDraft(group.tag, manualExitValue)}
                                >
                                  {tr(
                                    "settings.groups.switch_manual",
                                    "Switch to Manual Selection",
                                  )}
                                </Button>
                              </div>
                            ) : null}
                          </div>
                        ) : nodeOptions.length > 0 ? (
                          <select
                            className="bp-native-select"
                            value={safeNodeDraftValue}
                            onChange={(e) => updateDraft(group.tag, e.target.value)}
                          >
                            {nodeOptions.map((option) => (
                              <option key={`${group.tag}-${option.value}`} value={option.value}>
                                {option.label}
                              </option>
                            ))}
                          </select>
                        ) : (
                          <p className="bp-muted" style={{ margin: 0 }}>
                            {tr(
                              "settings.groups.node_empty",
                              "No business nodes available for manual selection.",
                            )}
                          </p>
                        )}
                        {!hasBusinessNodes ? (
                          <p className="bp-muted" style={{ margin: 0 }}>
                            {tr(
                              "settings.groups.manual_only",
                              "This group currently resolves to the shared manual fallback only.",
                            )}
                          </p>
                        ) : null}
                      </div>
                      <div className="bp-page-actions bp-settings-actions">
                        <Button
                          className="bp-btn-fixed"
                          type="primary"
                          loading={applyingTag === group.tag}
                          disabled={
                            (applyingTag.length > 0 && applyingTag !== group.tag) ||
                            !effectiveDraftValue ||
                            effectiveDraftValue === currentValue
                          }
                          onClick={() => void applyGroupChoice(group.tag, effectiveDraftValue)}
                        >
                          {tr("settings.groups.apply", "Apply Group Choice")}
                        </Button>
                      </div>
                    </div>
                  </details>
                );
              })}
            </div>
          ) : null}
        </div>
      )}
    </Card>
  );
}

function formatGroupLabel(
  tag: string,
  tr: (key: string, fallback: string, vars?: Record<string, string>) => string,
): string {
  if (tag === "manual") {
    return tr("settings.groups.manual_title", "Manual Fallback Group");
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

function formatSecondsInterval(value?: number): string {
  const sec = typeof value === "number" && value > 0 ? value : 1800;
  if (sec % 3600 === 0) {
    return `${sec / 3600}h`;
  }
  if (sec % 60 === 0) {
    return `${sec / 60}m`;
  }
  return `${sec}s`;
}

function dedupeOutbounds(values: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of values) {
    const value = raw.trim();
    if (!value || seen.has(value)) {
      continue;
    }
    seen.add(value);
    out.push(value);
  }
  return out;
}

function resolveManualExitValue(
  nodeCandidates: string[],
  runtimeEffectiveOutbound?: string | null,
): string | undefined {
  if (nodeCandidates.includes("manual")) {
    return "manual";
  }
  if (runtimeEffectiveOutbound && nodeCandidates.includes(runtimeEffectiveOutbound)) {
    return runtimeEffectiveOutbound;
  }
  return nodeCandidates[0];
}

function formatOutboundOptionLabel(
  value: string,
  tr: (key: string, fallback: string, vars?: Record<string, string>) => string,
): string {
  if (value === "manual") {
    return tr("settings.groups.follow_manual", "Follow Manual Fallback");
  }
  return value;
}

function formatQueryError(error: unknown, fallback: string): string {
  const anyErr = error as any;
  return (
    anyErr?.appError?.message ||
    anyErr?.response?.data?.error?.message ||
    anyErr?.message ||
    fallback
  );
}

function ProxySettingsCard({ title, proxyType, data, onSaved }: ProxyCardProps) {
  const { tr } = useI18n();
  const [form] = Form.useForm();
  const { addToast } = useToast();
  const update = useUpdateProxySettings();
  const authMode = Form.useWatch("auth_mode", form);
  const enabledWatch = Form.useWatch("enabled", form);
  const listenAddressWatch = Form.useWatch("listen_address", form);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!data) return;
    form.setFieldsValue({
      enabled: data.enabled,
      listen_address: data.listen_address,
      port: data.port,
      auth_mode: data.auth_mode,
      username: data.username || "",
      password: data.password || "",
    });
  }, [data, form]);

  const onSave = async () => {
    const values = await form.validateFields();
    if (isPublicNoAuth(values.enabled, values.listen_address, values.auth_mode)) {
      const ok = await confirmPublicNoAuth(tr);
      if (!ok) {
        return;
      }
    }
    await update.mutateAsync({
      proxy_type: proxyType,
      enabled: values.enabled,
      listen_address: values.listen_address,
      port: values.port,
      auth_mode: values.auth_mode,
      username: values.username,
      password: values.password,
    });
    onSaved?.();
  };

  const onCopy = async () => {
    if (!data) return;
    const preferredHost = window.location.hostname || undefined;
    const clientHost = resolveProxyClientHost(data.listen_address, preferredHost);
    const url = buildProxyUrl(data, preferredHost);
    try {
      await navigator.clipboard.writeText(url);
      addToast(
        "success",
        tr("settings.copy.success", "Connection string copied ({host}:{port})", {
          host: clientHost,
          port: data.port,
        }),
      );
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      addToast("error", tr("settings.copy.failed", "Copy failed"));
    }
  };

  const statusTag = data?.status ? (
    <Tag
      color={data.status === "running" ? "success" : data.status === "error" ? "error" : "default"}
    >
      {data.status === "running"
        ? tr("settings.status.running", "Running")
        : data.status === "error"
          ? tr("settings.status.error", "Error")
          : tr("settings.status.stopped", "Stopped")}
    </Tag>
  ) : null;

  return (
    <Card className="bp-settings-card">
      <div className="bp-card-header">
        <div>
          <p className="bp-card-kicker">{tr("settings.proxy.kicker", "Global Forwarding")}</p>
          <h2 className="bp-card-title">{title}</h2>
        </div>
        {statusTag}
      </div>
      <div className="bp-settings-status-row">
        <span className="bp-muted">{tr("settings.proxy.binding", "Current binding")}</span>
        <span className="bp-table-mono">
          {data?.listen_address ?? "0.0.0.0"}:{data?.port ?? "-"}
        </span>
      </div>
      <div className="bp-settings-status-row">
        <span className="bp-muted">{tr("settings.proxy.copy_host", "Copy URL host")}</span>
        <span className="bp-table-mono">
          {resolveProxyClientHost(
            data?.listen_address ?? "0.0.0.0",
            window.location.hostname || undefined,
          )}
        </span>
      </div>
      {data?.error_message && (
        <p className="bp-text-danger" style={{ marginBottom: 12 }}>
          {data.error_message}
        </p>
      )}
      {isPublicNoAuth(
        enabledWatch ?? data?.enabled ?? true,
        listenAddressWatch ?? data?.listen_address ?? "0.0.0.0",
        authMode ?? data?.auth_mode ?? "none",
      ) && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          title={tr("settings.security.warning_title", "Public exposure without authentication")}
          description={tr(
            "settings.security.warning_desc",
            "Current config listens on 0.0.0.0 with auth_mode=none. Anyone who can access this port may use your proxy.",
          )}
        />
      )}
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          enabled: true,
          listen_address: "0.0.0.0",
          port: proxyType === "http" ? 7890 : 7891,
          auth_mode: "none",
          username: "",
          password: "",
        }}
      >
        <Form.Item
          name="enabled"
          label={tr("settings.status.enabled", "Enabled")}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
        <Form.Item name="listen_address" label={tr("settings.proxy.listen", "Listen Address")}>
          <Select
            options={[
              { value: "127.0.0.1", label: "127.0.0.1 (Localhost)" },
              { value: "0.0.0.0", label: "0.0.0.0 (All Interfaces)" },
            ]}
            getPopupContainer={getOverlayContainer}
            classNames={selectPopupClassNames}
          />
        </Form.Item>
        <Form.Item
          name="port"
          label={tr("settings.proxy.port", "Port")}
          rules={[
            { required: true, message: tr("settings.proxy.port.required", "Port is required") },
            {
              type: "number",
              min: 1,
              max: 65535,
              message: tr("settings.proxy.port.range", "Port must be 1-65535"),
            },
          ]}
        >
          <InputNumber min={1} max={65535} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="auth_mode" label={tr("settings.proxy.auth_mode", "Auth Mode")}>
          <Select
            options={[
              { value: "none", label: tr("settings.proxy.auth.none", "None") },
              { value: "basic", label: tr("settings.proxy.auth.basic", "Basic") },
            ]}
            getPopupContainer={getOverlayContainer}
            classNames={selectPopupClassNames}
          />
        </Form.Item>
        {authMode === "basic" && (
          <>
            <Form.Item
              name="username"
              label={tr("settings.proxy.username", "Username")}
              rules={[
                {
                  required: true,
                  message: tr(
                    "settings.proxy.username.required",
                    "Username is required for Basic auth",
                  ),
                },
              ]}
            >
              <Input />
            </Form.Item>
            <Form.Item
              name="password"
              label={tr("settings.proxy.password", "Password")}
              rules={[
                {
                  required: true,
                  message: tr(
                    "settings.proxy.password.required",
                    "Password is required for Basic auth",
                  ),
                },
              ]}
            >
              <Input.Password />
            </Form.Item>
          </>
        )}
      </Form>
      <div className="bp-page-actions bp-settings-actions">
        <Button className="bp-btn-fixed" onClick={onSave} type="primary" loading={update.isPending}>
          {tr("common.save", "Save")}
        </Button>
        <Button className="bp-btn-fixed" onClick={onCopy} disabled={!data}>
          {copied ? tr("settings.copy.done", "Copied") : tr("settings.copy.url", "Copy URL")}
        </Button>
      </div>
    </Card>
  );
}

interface RoutingCardProps {
  data?: RoutingSettingsData;
  onSaved?: () => void;
}

function RoutingSettingsCard({ data, onSaved }: RoutingCardProps) {
  const { tr } = useI18n();
  const [form] = Form.useForm();
  const update = useUpdateRoutingSettings();

  useEffect(() => {
    if (!data) return;
    form.setFieldsValue({
      bypass_private_enabled: data.bypass_private_enabled,
      bypass_domains_text: (data.bypass_domains || []).join("\n"),
      bypass_cidrs_text: (data.bypass_cidrs || []).join("\n"),
    });
  }, [data, form]);

  const onSave = async () => {
    const values = await form.validateFields();
    await update.mutateAsync({
      bypass_private_enabled: values.bypass_private_enabled,
      bypass_domains: splitLines(values.bypass_domains_text),
      bypass_cidrs: splitLines(values.bypass_cidrs_text),
    });
    onSaved?.();
  };

  return (
    <Card className="bp-settings-card">
      <div className="bp-card-header">
        <div>
          <p className="bp-card-kicker">{tr("settings.routing.kicker", "Route Rules")}</p>
          <h2 className="bp-card-title">{tr("settings.routing.title", "Routing Bypass")}</h2>
        </div>
        {data?.updated_at ? (
          <span className="bp-muted">
            {tr("settings.routing.updated", "Updated {time}", { time: data.updated_at })}
          </span>
        ) : null}
      </div>
      <p className="bp-muted" style={{ marginTop: 0 }}>
        {tr("settings.routing.desc", "Matched domains and CIDRs will go direct instead of proxy.")}
      </p>
      <div style={{ display: "flex", flexWrap: "wrap", gap: 8, marginBottom: 12 }}>
        <Tag color={data?.bypass_private_enabled ? "success" : "default"}>
          {tr("settings.routing.cn_bundle", "CN Direct Bundle")}:{" "}
          {data?.bypass_private_enabled
            ? tr("nodes.status.enabled", "Enabled")
            : tr("nodes.status.disabled", "Disabled")}
        </Tag>
      </div>
      <p className="bp-muted" style={{ marginTop: -4, marginBottom: 12 }}>
        {tr("settings.routing.cn_bundle_desc", "Includes: private/LAN, geosite-cn, geoip-cn")}
      </p>
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          bypass_private_enabled: true,
          bypass_domains_text: "localhost\nlocal",
          bypass_cidrs_text:
            "127.0.0.0/8\n10.0.0.0/8\n172.16.0.0/12\n192.168.0.0/16\n169.254.0.0/16\n::1/128\nfc00::/7\nfe80::/10",
        }}
      >
        <Form.Item
          name="bypass_private_enabled"
          label={tr("settings.routing.enable", "Enable bypass rules")}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
        <Form.Item
          name="bypass_domains_text"
          label={tr("settings.routing.domains", "Bypass domains (one per line)")}
        >
          <Input.TextArea
            rows={4}
            placeholder="localhost&#10;local"
          />
        </Form.Item>
        <Form.Item
          name="bypass_cidrs_text"
          label={tr("settings.routing.cidrs", "Bypass CIDRs (one per line)")}
        >
          <Input.TextArea
            rows={6}
            placeholder="192.168.0.0/16&#10;10.0.0.0/8"
          />
        </Form.Item>
      </Form>
      <div className="bp-page-actions bp-settings-actions">
        <Button className="bp-btn-fixed" onClick={onSave} type="primary" loading={update.isPending}>
          {tr("common.save", "Save")}
        </Button>
      </div>
    </Card>
  );
}

function splitLines(raw: string): string[] {
  return raw
    .split(/\r?\n|,/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}

interface ForwardingPolicyCardProps {
  data?: {
    healthy_only_enabled: boolean;
    max_latency_ms: number;
    allow_untested: boolean;
    node_test_timeout_ms: number;
    node_test_concurrency: number;
    biz_auto_interval_sec: number;
    updated_at?: string;
  };
  onSaved?: () => void;
}

function ForwardingPolicyCard({ data, onSaved }: ForwardingPolicyCardProps) {
  const { tr } = useI18n();
  const [form] = Form.useForm();
  const update = useUpdateForwardingPolicy();

  useEffect(() => {
    if (!data) {
      return;
    }
    form.setFieldsValue({
      healthy_only_enabled: data.healthy_only_enabled,
      max_latency_ms: data.max_latency_ms,
      allow_untested: data.allow_untested,
      node_test_timeout_ms: data.node_test_timeout_ms,
      node_test_concurrency: data.node_test_concurrency,
      biz_auto_interval_sec: data.biz_auto_interval_sec,
    });
  }, [data, form]);

  const onSave = async () => {
    const values = await form.validateFields();
    await update.mutateAsync({
      healthy_only_enabled: values.healthy_only_enabled,
      max_latency_ms: values.max_latency_ms,
      allow_untested: values.allow_untested,
      node_test_timeout_ms: values.node_test_timeout_ms,
      node_test_concurrency: values.node_test_concurrency,
      biz_auto_interval_sec: values.biz_auto_interval_sec,
    });
    onSaved?.();
  };

  return (
    <Card id="forwarding-policy-card" className="bp-settings-card">
      <div className="bp-card-header">
        <div>
          <p className="bp-card-kicker">{tr("settings.forwarding.kicker", "Forwarding Policy")}</p>
          <h2 className="bp-card-title">
            {tr("settings.forwarding.title", "Node Eligibility Gate")}
          </h2>
        </div>
        {data?.updated_at ? (
          <span className="bp-muted">
            {tr("settings.forwarding.updated", "Updated {time}", { time: data.updated_at })}
          </span>
        ) : null}
      </div>
      <p className="bp-muted" style={{ marginTop: 0, marginBottom: 12 }}>
        {tr(
          "settings.forwarding.desc",
          "When enabled, only healthy nodes are included in runtime forwarding config.",
        )}
      </p>
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          healthy_only_enabled: true,
          max_latency_ms: 1200,
          allow_untested: false,
          node_test_timeout_ms: 3000,
          node_test_concurrency: 8,
          biz_auto_interval_sec: 1800,
        }}
      >
        <Form.Item
          name="healthy_only_enabled"
          label={tr("settings.forwarding.healthy_only", "Healthy nodes only")}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
        <Form.Item
          name="max_latency_ms"
          label={tr("settings.forwarding.max_latency", "Max latency (ms)")}
          rules={[
            {
              required: true,
              message: tr("settings.forwarding.max_latency.required", "Please enter max latency"),
            },
            {
              type: "number",
              min: 1,
              max: 10000,
              message: tr(
                "settings.forwarding.max_latency.range",
                "Max latency must be between 1 and 10000",
              ),
            },
          ]}
        >
          <InputNumber min={1} max={10000} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item
          name="allow_untested"
          label={tr("settings.forwarding.allow_untested", "Allow untested nodes")}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
        <Form.Item
          name="node_test_timeout_ms"
          label={tr("settings.forwarding.test_timeout", "Node test timeout (ms)")}
          rules={[
            {
              required: true,
              message: tr(
                "settings.forwarding.test_timeout.required",
                "Please enter node test timeout",
              ),
            },
            {
              type: "number",
              min: 500,
              max: 10000,
              message: tr(
                "settings.forwarding.test_timeout.range",
                "Timeout must be between 500 and 10000 ms",
              ),
            },
          ]}
        >
          <InputNumber min={500} max={10000} step={100} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item
          name="node_test_concurrency"
          label={tr("settings.forwarding.test_concurrency", "Node test concurrency")}
          rules={[
            {
              required: true,
              message: tr(
                "settings.forwarding.test_concurrency.required",
                "Please enter node test concurrency",
              ),
            },
            {
              type: "number",
              min: 1,
              max: 64,
              message: tr(
                "settings.forwarding.test_concurrency.range",
                "Concurrency must be between 1 and 64",
              ),
            },
          ]}
        >
          <InputNumber min={1} max={64} step={1} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item
          name="biz_auto_interval_sec"
          label={tr("settings.forwarding.biz_auto_interval", "Business auto test interval (sec)")}
          rules={[
            {
              required: true,
              message: tr(
                "settings.forwarding.biz_auto_interval.required",
                "Please enter auto test interval",
              ),
            },
            {
              type: "number",
              min: 60,
              max: 86400,
              message: tr(
                "settings.forwarding.biz_auto_interval.range",
                "Interval must be between 60 and 86400 sec",
              ),
            },
          ]}
        >
          <InputNumber min={60} max={86400} step={60} style={{ width: "100%" }} />
        </Form.Item>
      </Form>
      <div className="bp-page-actions bp-settings-actions">
        <Button className="bp-btn-fixed" onClick={onSave} type="primary" loading={update.isPending}>
          {tr("common.save", "Save")}
        </Button>
      </div>
    </Card>
  );
}

function isPublicNoAuth(enabled: boolean, listenAddress: string, authMode: string): boolean {
  return enabled && listenAddress === "0.0.0.0" && authMode === "none";
}

function confirmPublicNoAuth(
  tr: (
    key: string,
    fallback?: string,
    params?: Record<string, string | number | boolean | null | undefined>,
  ) => string,
): Promise<boolean> {
  return new Promise((resolve) => {
    Modal.confirm({
      title: tr("settings.security.confirm_title", "Confirm public unauthenticated proxy"),
      content: tr(
        "settings.security.confirm_desc",
        "This setting exposes proxy service on all interfaces without authentication. Continue only if this is intended.",
      ),
      okText: tr("common.save", "Save"),
      cancelText: tr("common.cancel", "Cancel"),
      onOk: () => resolve(true),
      onCancel: () => resolve(false),
    });
  });
}
