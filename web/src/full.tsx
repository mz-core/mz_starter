import React, { useEffect, useState } from 'react';
import { CheckCircle2, AlertTriangle, Info, Download, RefreshCw, XCircle } from 'lucide-react';
import { api, token } from './api';
import type { messages } from './i18n';

type T = (typeof messages)[keyof typeof messages];

type Component = {
  id: string;
  type: string;
  version?: string;
  tag?: string;
  repository?: string;
  asset?: string;
  digest?: string;
  size?: number;
  destination?: string;
  dependencies?: string[];
  source_kind: string;
  source_detail?: string;
  status: string;
  reason?: string;
  protected?: boolean;
  optional?: boolean;
};

type ArtifactPlan = {
  mode: string;
  path?: string;
  status: string;
  version?: string;
  digest?: string;
  url?: string;
  reason?: string;
};

type FullPlan = {
  target: string;
  profile: string;
  local_root?: string;
  local_authorized: boolean;
  artifacts: ArtifactPlan;
  components: Component[];
  required: number;
  resolved: number;
  blockers: string[];
  warnings: string[];
  configuration_required: string[];
  files_ready: boolean;
  server_ready: boolean;
};

type Progress = {
  state: string;
  resource?: string;
  index?: number;
  total?: number;
  received?: number;
  size?: number;
  installed?: number;
  installed_resources?: string[];
  missing?: string[];
  artifacts: ArtifactPlan;
  plan?: FullPlan;
  destination?: string;
  launch_script?: string;
  txadmin_url?: string;
  detail?: string;
  files?: number;
  files_ready: boolean;
  server_ready: boolean;
};

type Job = { kind: string; report: Progress; done: boolean };

type Props = {
  t: T;
  label: (value: string) => string;
  server: string;
  planReady: boolean;
};

const statusLabel: Record<string, [string, string]> = {
  ready: ['Pronto', 'Ready'],
  system: ['Fornecido pelos artifacts', 'Provided by artifacts'],
  installed: ['Instalado', 'Installed'],
  blocked: ['Bloqueado', 'Blocked'],
  source_missing: ['Sem distribuiÃ§Ã£o', 'No distribution'],
};

const sourceLabel: Record<string, [string, string]> = {
  github_release: ['GitHub Release', 'GitHub Release'],
  external_release: ['Release externo', 'External release'],
  local_folder: ['Pasta local autorizada', 'Authorized local folder'],
  local_zip: ['Pacote ZIP local', 'Local ZIP package'],
  pinned_archive: ['Snapshot oficial fixado', 'Pinned official snapshot'],
  system_artifact: ['Artifact do FXServer', 'FXServer artifact'],
};

export function FullInstallPanel({ t, label, server, planReady }: Props) {
  const [localRoot, setLocalRoot] = useState('');
  const [authorized, setAuthorized] = useState(false);
  const [artifactsMode, setArtifactsMode] = useState('auto');
  const [allowIncomplete, setAllowIncomplete] = useState(false);
  const [artifactsPath, setArtifactsPath] = useState('');
  const [plan, setPlan] = useState<FullPlan | null>(null);
  const [resolvedKey, setResolvedKey] = useState('');
  const [resolving, setResolving] = useState(false);
  const [resolveError, setResolveError] = useState('');
  const [confirmed, setConfirmed] = useState(false);
  const [jobID, setJobID] = useState('');
  const [job, setJob] = useState<Job | null>(null);
  const [installError, setInstallError] = useState('');

  const currentKey = JSON.stringify({ server, localRoot, authorized, artifactsMode, artifactsPath });
  const planMatches = !!plan && resolvedKey === currentKey && plan.target === server;

  useEffect(() => {
    if (!jobID || job?.done) return;
    let stopped = false;
    const poll = async () => {
      try {
        const value = await api<Job>('/api/installations/' + jobID);
        if (!stopped) setJob(value);
      } catch {
        if (!stopped) setInstallError(t.failed);
      }
    };
    poll();
    const timer = window.setInterval(poll, 700);
    return () => {
      stopped = true;
      window.clearInterval(timer);
    };
  }, [jobID, job?.done]);

  const resolve = async () => {
    setResolving(true);
    setResolveError('');
    setPlan(null);
    try {
      const value = await api<FullPlan>('/api/installations/resolve', {
        server,
        local_root: localRoot,
        local_authorized: authorized,
        artifacts_mode: artifactsMode,
        artifacts_path: artifactsPath,
      });
      setPlan(value);
      setResolvedKey(currentKey);
      setConfirmed(false);
      setAllowIncomplete(false);
    } catch (error) {
      setResolveError(error instanceof Error ? error.message : 'network_error');
    } finally {
      setResolving(false);
    }
  };

  const start = async () => {
    setInstallError('');
    try {
      const value = await api<{ id: string }>('/api/installations', {
        kind: 'full',
        server,
        full_confirmed: true,
        allow_incomplete: allowIncomplete,
        local_root: localRoot,
        local_authorized: authorized,
        artifacts_mode: artifactsMode,
        artifacts_path: artifactsPath,
      });
      setJob(null);
      setJobID(value.id);
    } catch (error) {
      setInstallError(error instanceof Error ? error.message : 'network_error');
    }
  };

  const cancelOrClear = async () => {
    if (!jobID) return;
    try {
      const response = await fetch('/api/installations/' + jobID, { method: 'DELETE', headers: { 'X-MZ-Session': token } });
      if (!response.ok) throw new Error();
      if (job?.done) {
        setJobID('');
        setJob(null);
      }
    } catch {
      setInstallError(t.failed);
    }
  };

  const report = job?.report;
  const stateText = report?.state === 'completed' ? t.fullCompleted : report?.state === 'completed_files_only' ? t.fullCompletedFilesOnly : report?.state === 'incomplete' ? t.fullIncomplete : label(report?.state || '');
  const detailText = report?.detail === 'configuration_required' ? t.configurationRequired : report?.detail === 'server_artifacts_not_provided' ? t.artifactsNotProvided : report?.detail === 'full_installation_blocked' ? t.blockedInstall : report?.detail === 'missing_required_components' ? t.missingRequired : report?.detail?.startsWith('artifacts_failed:') ? `${t.artifactsNotProvided}: ${report.detail.slice('artifacts_failed:'.length)}` : label(report?.detail || '');
  const installed = new Set(report?.installed_resources || []);
  const components = report?.plan?.components || plan?.components || [];
  const blockers = report?.plan?.blockers || plan?.blockers || [];
  const warnings = report?.plan?.warnings || plan?.warnings || [];
  const configuration = report?.plan?.configuration_required || plan?.configuration_required || [];
  const total = report?.total || (plan ? plan.required : 0);
  const installedCount = report?.installed || 0;
  const percent = total > 0 ? Math.min(100, Math.round((installedCount / total) * 100)) : 0;
  const canInstall = planMatches && !!plan && (plan.files_ready || allowIncomplete) && confirmed && !jobID;
  const done = !!job?.done;

  return (
    <section className="full-install">
      <div className="section-head">
        <h3>{t.fullInstall}</h3>
        <Download size={18} />
      </div>
      <p>{t.fullNotice}</p>
      <div className="fields">
        <label>
          {t.localRoot}
          <input value={localRoot} onChange={(e) => setLocalRoot(e.target.value)} placeholder="D:\\distribuicao-mz" />
        </label>
        <small>{t.localRootHint}</small>
        <label className="check">
          <input type="checkbox" checked={authorized} onChange={(e) => setAuthorized(e.target.checked)} />
          {t.localAuthorized}
        </label>
        <label>
          {t.artifactsMode}
          <select value={artifactsMode} onChange={(e) => setArtifactsMode(e.target.value)}>
            <option value="auto">{t.artifactsAuto}</option>
            <option value="none">{t.artifactsNone}</option>
            <option value="existing">{t.artifactsExisting}</option>
            <option value="download">{t.artifactsDownload}</option>
          </select>
        </label>
        {(artifactsMode === 'existing' || artifactsMode === 'download') && (
          <label>
            {t.artifactsPath}
            <input value={artifactsPath} onChange={(e) => setArtifactsPath(e.target.value)} placeholder={t.artifactsHint} />
          </label>
        )}
      </div>
      <div className="wizard-actions">
        <button className="primary" disabled={!server.trim() || resolving} onClick={resolve}>
          <RefreshCw size={16} /> {resolving ? t.resolving : t.resolveComponents}
        </button>
      </div>
      {resolveError && <p className="error" role="alert">{label(resolveError)}</p>}
      {plan && (
        <>
          {!planMatches && <p className="error">{t.resolveRequiredAgain}</p>}
          <div className="summary-grid">
            <div><span>{t.requiredCount}</span><strong>{plan.required}</strong></div>
            <div><span>{t.resolvedCount}</span><strong>{plan.resolved}</strong></div>
            <div><span>{t.blockersTitle}</span><strong>{blockers.length}</strong></div>
            <div><span>{t.warningsTitle}</span><strong>{warnings.length}</strong></div>
          </div>
          {blockers.length > 0 && (
            <div className="issue">
              <AlertTriangle size={17} />
              <div>
                <strong>{t.blockersTitle}</strong>
                <ul>{blockers.map((value) => <li key={value}>{label(value)}</li>)}</ul>
              </div>
            </div>
          )}
          {warnings.length > 0 && (
            <details>
              <summary>{t.warningsTitle} ({warnings.length})</summary>
              <ul>{warnings.map((value) => <li key={value}>{label(value)}</li>)}</ul>
            </details>
          )}
          <div className="table-wrap component-table">
            <table>
              <thead><tr><th>{t.component}</th><th>{t.status}</th><th>{t.sourceLabel}</th><th>{t.available}</th></tr></thead>
              <tbody>
                {components.map((component) => {
                  const state = installed.has(component.id) ? 'installed' : component.status;
                  return (
                    <tr key={component.id}>
                      <td><strong>{component.id}</strong>{component.optional && <small>{t.optionalComponent}</small>}</td>
                      <td><span className="tag">{label(state)}</span>{component.reason && <small>{label(component.reason)}</small>}</td>
                      <td>{label(component.source_kind)}{component.source_detail && <small>{component.source_detail}</small>}</td>
                      <td>{component.version || 'â€”'}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          <p className="muted">{t.artifactsStatus}: {label(plan.artifacts.status)}{plan.artifacts.version ? ` Â· ${plan.artifacts.version}` : ''}{plan.artifacts.path ? ` Â· ${plan.artifacts.path}` : ''}</p>
          <label className="check">
            <input type="checkbox" checked={confirmed} onChange={(e) => setConfirmed(e.target.checked)} />
            {t.fullConfirm}
          </label>
          {!plan.files_ready && (
            <label className="check">
              <input type="checkbox" checked={allowIncomplete} onChange={(e) => setAllowIncomplete(e.target.checked)} />
              {t.allowIncomplete}
            </label>
          )}
          <button className="primary" disabled={!canInstall} onClick={start}>
            <Download size={16} /> {t.installButton}
          </button>
        </>
      )}
      {installError && <p className="error" role="alert">{installError}</p>}
      {report && (
        <div role="status">
          <div className="progress"><span style={{ width: `${percent}%` }} /></div>
          <p><strong>{stateText}</strong> {report.resource && `Â· ${report.resource}`} {report.index && report.total ? `Â· ${report.index}/${report.total}` : ''} {report.installed ? `Â· ${report.installed}/${report.total}` : ''}</p>
          <p>{t.installDone}: {installedCount} Â· {t.filesCreated}: {report.files || 0}{report.received ? ` Â· ${report.received}/${report.size || 0} ${t.installBytes}` : ''}</p>
          {report.detail && <p>{detailText}</p>}
          {report.destination && <p><strong>{t.serverDataDirectory}:</strong> <span className="hash">{report.destination}</span></p>}
          {report.artifacts && <p>{t.artifactsStatus}: {label(report.artifacts.status)}{report.artifacts.version ? ` Â· ${report.artifacts.version}` : ''}{report.artifacts.path ? ` Â· ${report.artifacts.path}` : ''}</p>}
          {report.artifacts?.path && <p><strong>{t.fxserverDirectory}:</strong> <span className="hash">{report.artifacts.path}</span></p>}
          {report.launch_script && (
            <div className="notice">
              <Info size={16} />
              <div>
                <strong>{t.txAdminTitle}</strong>
                <p>{t.txAdminRun} <code className="hash">{report.launch_script}</code></p>
                <p>{t.txAdminURL} <code>{report.txadmin_url || 'http://127.0.0.1:40120'}</code></p>
                <p>{t.txAdminExisting}</p>
                <p>{t.txAdminTunnel}</p>
                {report.state === 'incomplete' && <p className="error">{t.txAdminIncomplete}</p>}
              </div>
            </div>
          )}
          {done && !report.launch_script && <p className="muted">{t.txAdminUnavailable}</p>}
          {report.missing && report.missing.length > 0 && <p className="error">{t.blockersTitle}: {report.missing.join(', ')}</p>}
          {done && (
            <p className={report.files_ready ? 'notice' : 'error'}>
              {report.files_ready ? <CheckCircle2 size={16} /> : <XCircle size={16} />}
              {report.files_ready ? t.fullFilesReady : report.state === 'incomplete' ? t.fullIncomplete : t.fullFilesNotReady}
              {report.server_ready ? ` Â· ${t.serverReady}` : ` Â· ${t.notReady}`}
            </p>
          )}
          {done && configuration.length > 0 && (
            <details>
              <summary>{t.configurationRequired}</summary>
              <ul>{configuration.map((value) => <li key={value}>{label(value)}</li>)}</ul>
            </details>
          )}
          <div className="wizard-actions">
            <button onClick={cancelOrClear}>{done ? t.installClear : t.installCancel}</button>
          </div>
        </div>
      )}
      {!plan && !resolving && <p className="muted"><Info size={15} /> {t.fullResolveHint}</p>}
      {planReady && !plan && <small className="muted">{t.planWarning}</small>}
    </section>
  );
}
