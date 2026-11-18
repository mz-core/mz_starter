import React, { useEffect, useRef, useState } from 'react';
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
  installed?: boolean;
  installed_version?: string;
  fallback?: boolean;
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
  installed?: number;
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
  failures?: { id: string; reason: string }[];
  backups?: string[];
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
  const [resumeExisting, setResumeExisting] = useState(false);
  const [updateExisting, setUpdateExisting] = useState(false);
  const [updateID, setUpdateID] = useState('');
  const [excluded, setExcluded] = useState<string[]>([]);
  const [artifactsPath, setArtifactsPath] = useState('');
  const [plan, setPlan] = useState<FullPlan | null>(null);
  const [resolvedKey, setResolvedKey] = useState('');
  const [resolving, setResolving] = useState(false);
  const [resolveError, setResolveError] = useState('');
  const [confirmed, setConfirmed] = useState(false);
  const [jobID, setJobID] = useState('');
  const [job, setJob] = useState<Job | null>(null);
  const [installError, setInstallError] = useState('');

  const currentKey = JSON.stringify({ server, localRoot, authorized, artifactsMode, artifactsPath, excluded });
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
        excluded_ids: excluded,
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
        allow_incomplete: allowIncomplete || resumeExisting,
        resume_existing: resumeExisting,
        update_existing: resumeExisting && updateExisting,
        update_ids: resumeExisting && updateExisting && updateID ? [updateID] : [],
        excluded_ids: excluded,
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
  const detailText = report?.detail?.startsWith('existing_installation_incomplete:') ? `${t.resumeMissingFile} ${report.detail.slice('existing_installation_incomplete:'.length)}. ${t.resumeMissingHint}` : report?.detail === 'existing_installation_not_created_by_starter' ? t.resumeNotStarter : report?.detail === 'existing_installation_missing' ? t.resumeDirectoryMissing : report?.detail === 'configuration_required' ? t.configurationRequired : report?.detail === 'server_artifacts_not_provided' ? t.artifactsNotProvided : report?.detail === 'full_installation_blocked' ? t.blockedInstall : report?.detail === 'missing_required_components' ? t.missingRequired : report?.detail?.startsWith('artifacts_failed:') ? `${t.artifactsNotProvided}: ${report.detail.slice('artifacts_failed:'.length)}` : label(report?.detail || '');
  const installed = new Set(report?.installed_resources || []);
  const components = report?.plan?.components || plan?.components || [];
  const blockers = report?.plan?.blockers || plan?.blockers || [];
  const warnings = report?.plan?.warnings || plan?.warnings || [];
  const configuration = report?.plan?.configuration_required || plan?.configuration_required || [];
  const total = report?.total || (plan ? plan.required : 0);
  const installedCount = report?.installed || 0;
  const percent = total > 0 ? Math.min(100, Math.round((installedCount / total) * 100)) : 0;
  const canInstall = planMatches && !!plan && (plan.files_ready || allowIncomplete || resumeExisting) && confirmed && !jobID;
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
            <p>{t.selectComponentsHint}</p>
            <table>
              <thead><tr><th>{t.component}</th><th>{t.status}</th><th>{t.sourceLabel}</th><th>{t.available}</th></tr></thead>
              <tbody>
                {components.map((component) => {
                  const state = installed.has(component.id) ? 'installed' : component.installed ? 'installed' : component.status;
                  return (
                    <tr key={component.id}>
                      <td><label><input type="checkbox" aria-label={`${t.selectComponent} ${component.id}`} checked={!excluded.includes(component.id)} onChange={(e) => setExcluded(e.target.checked ? excluded.filter((id) => id !== component.id) : [...excluded, component.id])} /> <strong>{component.id}</strong></label>{component.optional && <small>{t.optionalComponent}</small>}</td>
                      <td><span className="tag">{label(state)}</span>{component.reason && <small>{label(component.reason)}</small>}</td>
                      <td>{label(component.source_kind)}{component.source_detail && <small>{component.source_detail}</small>}</td>
                      <td>
                        {component.installed_version && <small>{t.installed}: {component.installed_version}</small>}{component.version || '—'}
                        {component.tag && <small>{component.tag}</small>}
                        {component.fallback && <small className="error">{t.fallbackVersion}</small>}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          <p className="muted">{t.artifactsStatus}: {label(plan.artifacts.status)}{plan.artifacts.version ? ` Â· ${plan.artifacts.version}` : ''}{plan.artifacts.path ? ` Â· ${plan.artifacts.path}` : ''}</p>
          <label className="check">
            <input type="checkbox" checked={resumeExisting} onChange={(e) => setResumeExisting(e.target.checked)} />
            {t.resumeExisting}
          </label>
          <p className="muted">{t.resumeHint}</p>
          {resumeExisting && <label className="check">
            <input type="checkbox" checked={updateExisting} onChange={(e) => setUpdateExisting(e.target.checked)} />
            {t.updateExisting}
          </label>}
          {resumeExisting && updateExisting && <p className="muted">{t.updateExistingHint}</p>}
          {resumeExisting && updateExisting && <label>{t.updateSelection}
            <select value={updateID} onChange={(e) => setUpdateID(e.target.value)}>
              <option value="">{t.updateAll}</option>
              {components.filter((c) => c.installed && c.type === 'fivem_resource').map((c) => <option key={c.id} value={c.id}>{c.id}</option>)}
            </select>
          </label>}
          <label className="check">
            <input type="checkbox" checked={confirmed} onChange={(e) => setConfirmed(e.target.checked)} />
            {resumeExisting && updateExisting ? t.updateConfirm : t.fullConfirm}
          </label>
          {!plan.files_ready && (
            <label className="check">
              <input type="checkbox" checked={allowIncomplete} onChange={(e) => setAllowIncomplete(e.target.checked)} />
              {t.allowIncomplete}
            </label>
          )}
          <button className="primary" disabled={!canInstall} onClick={start}>
            <Download size={16} /> {resumeExisting ? t.completeInstallButton : t.installButton}
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
          {report.backups && report.backups.length > 0 && <details><summary>{t.updateBackups}</summary><ul>{report.backups.map((path) => <li key={path}><code>{path}</code></li>)}</ul></details>}
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
          {report.missing && report.missing.length > 0 && (
            <details>
              <summary>{t.missingReasons}</summary>
              <ul>
                {report.missing.map((id) => {
                  const failed = report.failures?.find((failure) => failure.id === id);
                  const component = components.find((item) => item.id === id);
                  return <li key={id}>{id}: {label(failed?.reason || component?.reason || 'missing')}</li>;
                })}
              </ul>
            </details>
          )}
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

function versionStatus(component: Component, t: T): string {
  const current = component.installed_version;
  const available = component.version;
  if (component.status !== 'ready' || component.fallback || !current || !available) return t.simpleUnknownVersion;
  if (current === available) return t.simpleUpToDate;
  const a = current.match(/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/);
  const b = available.match(/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/);
  if (!a || !b) return t.simpleUnknownVersion;
  for (let i = 1; i <= 3; i++) {
    if (a[i] === b[i]) continue;
    return a[i].length < b[i].length || (a[i].length === b[i].length && a[i] < b[i]) ? t.simpleUpdateAvailable : t.simpleNewerVersion;
  }
  return t.simpleUnknownVersion;
}

// Simple new-server flow; advanced resume/update remains separate.
export function SimpleInstallPanel({ t, label, server, resources, suggested, onSuggestion }: { t: T; label: (v: string) => string; server: string; resources: {id:string; type:string}[]; suggested: string; onSuggestion: (value:string) => void }) {
  const [artifacts, setArtifacts] = useState(true);
  const [mode, setMode] = useState<'new'|'existing'>('new');
  const [inspection, setInspection] = useState<FullPlan | null>(null);
  const [checking, setChecking] = useState(false);
  const [excluded, setExcluded] = useState<string[]>(['mz_banguard']);
  const [id, setID] = useState('');
  const [job, setJob] = useState<Job | null>(null);
  const [error, setError] = useState('');
  const resultDialog = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    if (job?.done && resultDialog.current && !resultDialog.current.open) resultDialog.current.showModal();
  }, [id, job?.done]);
  useEffect(() => {
    if (!id || job?.done) return;
    let active = true;
    const poll = async () => { try { const next = await api<Job>('/api/installations/' + id); if (active) setJob(next); } catch { if (active) setError(t.failed); } };
    poll(); const timer = window.setInterval(poll, 700);
    return () => { active = false; window.clearInterval(timer); };
  }, [id, job?.done]);
  const start = async () => {
    setError('');
    try {
      const next = await api<{id:string}>('/api/installations', mode === 'existing' ? {kind:'artifacts', server} : { kind:'full', server, full_confirmed:true, allow_incomplete:true, artifacts_mode:artifacts ? 'auto' : 'none', excluded_ids:excluded.filter(id => resources.some(r => r.id === id)) });
      setJob(null); setID(next.id);
    } catch (e) { setError(e instanceof Error ? e.message : t.failed); }
  };
  const inspect = async () => {
    setError(''); setChecking(true); setInspection(null);
    try { setInspection(await api<FullPlan>('/api/installations/resolve', {server, existing:true, artifacts_mode:'none'})); }
    catch (e) { setError(e instanceof Error ? e.message : t.failed); }
    finally { setChecking(false); }
  };
  const reset = async () => {
    if (!id) return;
    const result = await fetch('/api/installations/' + id, {method:'DELETE', headers:{'X-MZ-Session':token}});
    if (!result.ok) { setError(t.failed); return; }
    if (job?.done) { setID(''); setJob(null); }
  };
  return <section className="full-install">
    <h3>{t.simpleInstall}</h3>
    <label className="check"><input type="radio" name="simple-mode" checked={mode === 'new'} onChange={() => { setMode('new'); setInspection(null); }} />{t.simpleNewMode}</label>
    <label className="check"><input type="radio" name="simple-mode" checked={mode === 'existing'} onChange={() => { setMode('existing'); setInspection(null); }} />{t.simpleExistingMode}</label>
    <p>{mode === 'new' ? t.simpleIntro : t.simpleExistingHint}</p>
    {mode === 'new' && <><label className="check"><input type="checkbox" checked={artifacts} onChange={e => setArtifacts(e.target.checked)} />{t.simpleArtifacts}</label>
    <details><summary>{t.simpleChoose}</summary><p>{t.simpleChooseHint}</p>{resources.filter(r => r.type !== 'artifact').map(r => <label className="check" key={r.id}><input type="checkbox" checked={!excluded.includes(r.id)} onChange={e => setExcluded(e.target.checked ? excluded.filter(id => id !== r.id) : [...excluded,r.id])} />{r.id}</label>)}</details></>}
    {mode === 'existing' && <><button disabled={!server.trim() || checking || !!id} onClick={inspect}>{checking ? t.resolving : t.simpleInspect}</button>
      {inspection && inspection.target === server && <details open><summary>{t.simpleDetected}: {inspection.components.filter(c => c.installed).length}</summary>
        {inspection.components.filter(c => c.installed).map(c => <p key={c.id}>{c.id}: {t.installed} {c.installed_version || '—'} · {t.available} {c.status === 'ready' ? c.version || '—' : '—'}{c.fallback ? ` (${t.fallbackVersion})` : ''} · {versionStatus(c, t)}</p>)}
        <small>{t.simpleInspectionHint}</small></details>}</>}
    <button className="primary" disabled={!server.trim() || !!id} onClick={start}><Download size={16}/>{mode === 'existing' ? t.simpleArtifactStart : t.simpleStart}</button>
    {error && <p className="error" role="alert">{label(error)}</p>}
    {job && <div role="status"><p><strong>{label(job.report.state)}</strong> {job.report.resource || ''}</p>
      {job.report.total ? <progress value={job.report.installed || 0} max={job.report.total}/> : null}
      {job.report.detail && <p>{job.report.detail === 'artifacts_only_complete' ? t.simpleArtifactsDone : job.report.detail.includes('untrusted writable principal') ? t.simpleUnsafeFolder : label(job.report.detail)}</p>}
      {job.done && job.report.detail?.includes('untrusted writable principal') && suggested && <button onClick={() => { onSuggestion(suggested); reset(); }}>{t.simpleUseSuggested}: {suggested}</button>}
      {job.report.missing && job.report.missing.length > 0 && <p>{t.simpleMissing}: {job.report.missing.join(', ')}</p>}
      {job.report.destination && <p>{t.serverDataDirectory}: {job.report.destination}</p>}
      {job.report.artifacts?.path && <p>{t.fxserverDirectory}: {job.report.artifacts.path}</p>}
      {job.report.launch_script && <p>{t.txAdminRun} <code>{job.report.launch_script}</code></p>}
      {job.done && <p>{job.report.state === 'failed' ? t.fullFilesNotReady : job.kind === 'artifacts' ? t.simpleArtifactsDone : t.simpleAfter}</p>}
      <button onClick={reset}>{job.done ? t.installClear : t.installCancel}</button>
    </div>}
    <dialog ref={resultDialog} className="install-result" aria-labelledby="install-result-title">
      <h2 id="install-result-title">{job?.report.state === 'cancelled' ? t.resultCancelled : job?.report.state === 'failed' ? t.resultFailed : job?.report.state === 'incomplete' || job?.report.state === 'completed_files_only' ? t.resultIncomplete : t.resultComplete}</h2>
      <p>{job?.report.state === 'failed' ? t.resultFailedHint : job?.report.state === 'cancelled' ? t.resultCancelledHint : job?.kind === 'artifacts' ? t.simpleArtifactsDone : t.resultCompleteHint}</p>
      {job?.report.missing && job.report.missing.length > 0 && <p>{t.simpleMissing}: {job.report.missing.join(', ')}</p>}
      {job?.report.detail && job.report.state === 'failed' && <p>{job.report.detail.includes('untrusted writable principal') ? t.simpleUnsafeFolder : label(job.report.detail)}</p>}
      {job?.report.detail?.includes('artifacts_failed:') && <p>{t.artifactsNotProvided}: {job.report.detail.split('artifacts_failed:')[1]}</p>}
      {job?.report.destination && <p>{t.serverDataDirectory}: <code className="hash">{job.report.destination}</code></p>}
      <button className="primary" onClick={() => resultDialog.current?.close()} autoFocus>{t.resultClose}</button>
    </dialog>
  </section>;
}

export function ResourceManagerPanel({t, label, server, onServer, resources, os}: {t:T; label:(v:string)=>string; server:string; onServer:(v:string)=>void; resources:{id:string;type:string}[]; os:string}) {
  const [plan, setPlan] = useState<FullPlan | null>(null);
  const [checking, setChecking] = useState(false);
  const [stopped, setStopped] = useState(false);
  const [jobID, setJobID] = useState('');
  const [job, setJob] = useState<Job | null>(null);
  const [selected, setSelected] = useState('');
  const [error, setError] = useState('');
  const ids = resources.map(r => r.id);
  const mz = resources.filter(r => r.type === 'fivem_resource');
  const inspect = async () => {
    setChecking(true); setError(''); setPlan(null);
    try { setPlan(await api<FullPlan>('/api/installations/resolve', {server, existing:true, artifacts_mode:'none', excluded_ids:[...ids, 'baseevents']})); }
    catch (e) { setError(e instanceof Error ? e.message : t.failed); }
    finally { setChecking(false); }
  };
  useEffect(() => {
    if (!jobID || job?.done) return;
    let active = true;
    const poll = async () => { try { const next = await api<Job>('/api/installations/' + jobID); if (active) setJob(next); } catch { if (active) setError(t.failed); } };
    poll(); const timer = window.setInterval(poll, 700); return () => { active = false; window.clearInterval(timer); };
  }, [jobID, job?.done]);
  const start = async (resource: string) => {
    setError('');
    try {
      const next = await api<{id:string}>('/api/installations', {kind:'full', server, full_confirmed:true, resume_existing:true, update_existing:true, update_ids:[resource], excluded_ids:[...ids.filter(id => id !== resource), 'baseevents'], artifacts_mode:'none'});
      setSelected(resource); setJob(null); setJobID(next.id);
    } catch (e) { setError(e instanceof Error ? e.message : t.failed); }
  };
  const clear = async () => {
    const response = await fetch('/api/installations/' + jobID, {method:'DELETE', headers:{'X-MZ-Session':token}});
    if (!response.ok) { setError(t.failed); return; }
    if (job?.done) { setJobID(''); setJob(null); await inspect(); }
  };
  const components = new Map((plan?.target === server ? plan.components : []).map(c => [c.id, c]));
  const reason = job?.report.failures?.find(x => x.id === selected)?.reason || job?.report.plan?.components.find(c => c.id === selected)?.reason || job?.report.detail || '';
  const humanReason = reason.includes('rate_limited') ? t.resourceRateLimited : reason.includes('update_baseline_missing') ? t.resourceNeedsReview : reason.includes('core_migration_review_required') ? t.resourceCoreReview : label(reason);
  return <section className="full-install">
    <h3>{t.resourceManageTitle}</h3><p>{t.resourceManageHint}</p>
    <label className="fields">{t.server}<input value={server} onChange={e => {onServer(e.target.value); setPlan(null);}} placeholder={os === 'windows' ? 'C:\\FXServer\\server-data' : '/srv/fxserver/server-data'} /></label>
    <button disabled={!server.trim() || checking || !!jobID} onClick={inspect}>{checking ? t.resolving : t.resourceScan}</button>
    {error && <p className="error" role="alert">{label(error)}</p>}
    {plan && plan.target === server && <>
      <p>{t.resourceListHint}</p>
      <label className="check"><input type="checkbox" checked={stopped} onChange={e => setStopped(e.target.checked)}/>{t.resourceStopped}</label>
      <div className="table-wrap"><table><thead><tr><th>{t.component}</th><th>{t.installed}</th><th>{t.status}</th><th>{t.resourceAction}</th></tr></thead><tbody>
        {mz.map(r => {const c=components.get(r.id); return <tr key={r.id}><td>{r.id}</td><td>{c?.installed ? (c.installed_version || t.unknown) : '—'}</td><td>{c?.installed ? t.resourcePresent : t.resourceAbsent}</td><td><button disabled={!!jobID || (c?.installed && !stopped)} onClick={() => start(r.id)}>{c?.installed ? t.resourceUpdate : t.resourceDownload}</button></td></tr>})}
      </tbody></table></div>
    </>}
    {job && <div role="status"><p>{selected}: {label(job.report.state)} {job.report.resource || ''}</p>
      {job.done && job.report.failures && job.report.failures.length > 0 && <p className="error">{humanReason}</p>}
      {job.done && !job.report.failures?.length && job.report.missing?.includes(selected) && <p className="error">{humanReason || t.resourceUnavailable}</p>}
      {job.done && job.report.state === 'failed' && <p className="error">{humanReason}</p>}
      {job.done && job.report.installed_resources?.includes(selected) && <p>{t.resourceDone}</p>}
      {job.done && !job.report.installed_resources?.includes(selected) && !job.report.missing?.includes(selected) && job.report.state !== 'failed' && <p>{t.resourceNoChange}</p>}
      {job.report.backups && job.report.backups.length > 0 && <p>{t.updateBackups}: {job.report.backups.join(', ')}</p>}
      <button onClick={clear}>{job.done ? t.installClear : t.installCancel}</button>
    </div>}
  </section>;
}
