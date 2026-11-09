import { useEffect, useState } from "react";
import {
  X,
  ArrowUpRight,
  RotateCcw,
  Download,
  Braces,
  Clock,
  ShieldCheck,
} from "lucide-react";
import { api, post, textBody, size, stamp } from "./api";
import type { Event, Replay } from "./api";
import { Method, Code, CopyButton, Modal, Spinner } from "./ui";
export default function Detail({
  id,
  onClose,
  notify,
}: {
  id: number;
  onClose: () => void;
  notify: (s: string) => void;
}) {
  const [event, setEvent] = useState<Event>();
  const [error, setError] = useState("");
  const [tab, setTab] = useState("Body");
  const [raw, setRaw] = useState(false);
  const [replay, setReplay] = useState(false);
  const [history, setHistory] = useState<Replay[]>([]);
  const [historyError, setHistoryError] = useState("");
  useEffect(() => {
    let live = true;
    setEvent(undefined);
    setError("");
    setTab("Body");
    api<Event>("/events/" + id)
      .then((e) => {
        if (live) setEvent(e);
      })
      .catch((e) => {
        if (live) setError(e.message);
      });
    const key = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !document.querySelector("dialog[open]"))
        onClose();
    };
    window.addEventListener("keydown", key);
    return () => {
      live = false;
      window.removeEventListener("keydown", key);
    };
  }, [id, onClose]);
  useEffect(() => {
    if (tab !== "重放记录") return;
    let live = true;
    api<{ replays: Replay[] }>("/events/" + id + "/replays")
      .then((r) => {
        if (live) {
          setHistory(r.replays);
          setHistoryError("");
        }
      })
      .catch((e) => {
        if (live) setHistoryError(e.message);
      });
    return () => {
      live = false;
    };
  }, [id, tab, replay]);
  const text = event ? textBody(event) : "";
  let formatted = text,
    isJSON = false;
  try {
    if (text.length > 24000) throw new Error("large preview");
    formatted = JSON.stringify(JSON.parse(text), null, 2);
    isJSON = true;
  } catch {
    /* arbitrary payload */
  }
  const content = raw ? text : formatted;
  const previewLines = content.slice(0, 24000).split("\n", 201);
  const truncated = content.length > 24000 || previewLines.length > 200;
  const preview = previewLines.slice(0, 200).join("\n");
  function download() {
    if (!event) return;
    const b = Uint8Array.from(atob(event.body_base64 || ""), (c) =>
      c.charCodeAt(0),
    );
    const url = URL.createObjectURL(new Blob([b]));
    const a = document.createElement("a");
    a.href = url;
    a.download = `hooklab-event-${id}.bin`;
    a.click();
    URL.revokeObjectURL(url);
  }
  return (
    <>
      <div className="detail-scrim" onClick={onClose} />
      <aside
        className="detail-panel"
        role="dialog"
        aria-modal="true"
        aria-label="事件详情"
      >
        <header className="detail-header">
          <span className="eyebrow">EVENT INSPECTOR</span>
          <button
            className="icon-button"
            aria-label="关闭详情"
            onClick={onClose}
          >
            <X size={19} />
          </button>
        </header>
        <div className="detail-title">
          <h2>
            事件详情 <span>#{id.toString().padStart(6, "0")}</span>
          </h2>
          {event && (
            <div className="request-path">
              <Method value={event.method} />
              <strong>{event.path}</strong>
            </div>
          )}
        </div>
        {error ? (
          <div role="alert" className="error-banner">
            {error}
          </div>
        ) : !event ? (
          <div className="panel-loading">
            <Spinner />
            正在加载事件…
          </div>
        ) : (
          <>
            <div className="detail-meta">
              <div>
                <span>接收时间</span>
                <strong>{stamp(event.received_at)}</strong>
              </div>
              <div>
                <span>来源 IP</span>
                <strong>{event.source_ip || "unknown"}</strong>
              </div>
              <div>
                <span>所属 Endpoint</span>
                <strong>{event.endpoint_name}</strong>
              </div>
              <div>
                <span>请求大小</span>
                <strong>{size(event.size)}</strong>
              </div>
            </div>
            <div className="detail-actions">
              <button className="secondary" onClick={() => setReplay(true)}>
                <RotateCcw size={14} />
                重放事件
                <ArrowUpRight size={13} />
              </button>
              <button className="quiet" onClick={download}>
                <Download size={14} />
                下载原始 Body
              </button>
            </div>
            <div className="tabs">
              {["Body", "Headers", "Query", "重放记录"].map((t) => (
                <button
                  key={t}
                  className={tab === t ? "active" : ""}
                  onClick={() => setTab(t)}
                >
                  {t}
                  {t === "Headers" && (
                    <small aria-hidden="true">
                      {Object.keys(event.headers || {}).length}
                    </small>
                  )}
                </button>
              ))}
            </div>
            <div className="detail-content">
              {tab === "Body" ? (
                <>
                  <div className="viewer-toolbar">
                    <span>
                      <Braces size={14} />
                      {isJSON ? "application/json" : "原始请求内容"}
                      <small>{size(event.size)}</small>
                    </span>
                    <div className="segmented">
                      <button
                        className={!raw ? "active" : ""}
                        onClick={() => setRaw(false)}
                      >
                        格式化
                      </button>
                      <button
                        className={raw ? "active" : ""}
                        onClick={() => setRaw(true)}
                      >
                        Raw
                      </button>
                    </div>
                  </div>
                  {!isJSON && !raw && text.length <= 24000 && (
                    <p className="viewer-hint">
                      非 JSON 内容，按原始文本显示。二进制数据请下载查看。
                    </p>
                  )}
                  {truncated && (
                    <p className="viewer-hint" role="status">
                      已限制预览为最多 200 行 / 24,000
                      字符；下载可获取完整原始数据。
                    </p>
                  )}
                  <div className="code-view" data-testid="body-viewer">
                    <div className="line-numbers" aria-hidden="true">
                      {preview.split("\n").map((_, i) => (
                        <span key={i}>{i + 1}</span>
                      ))}
                    </div>
                    <pre>
                      {!event.size ? (
                        <span className="muted">此请求没有 Body</span>
                      ) : raw || preview.length > 12000 ? (
                        <code>{preview}</code>
                      ) : (
                        <Code value={preview} />
                      )}
                    </pre>
                  </div>
                  <div className="viewer-footer">
                    <span>UTF-8 · {isJSON ? "JSON" : "TEXT / BINARY"}</span>
                    <CopyButton
                      value={raw ? text : formatted}
                      label="复制内容"
                      onError={notify}
                    />
                  </div>
                </>
              ) : tab === "Headers" ? (
                <>
                  <div className="privacy-note">
                    <ShieldCheck size={15} />
                    敏感请求头已在存储前脱敏
                  </div>
                  <div data-testid="headers-viewer" className="key-values">
                    {Object.entries(event.headers || {}).map(([k, v]) => (
                      <div key={k}>
                        <span>{k}</span>
                        <code>{v.join(", ")}</code>
                      </div>
                    ))}
                  </div>
                </>
              ) : tab === "Query" ? (
                <>
                  <p className="muted small">原始 Query String</p>
                  <pre className="query-raw">
                    {event.raw_query || "（无查询参数）"}
                  </pre>
                  <div className="key-values">
                    {Array.from(new URLSearchParams(event.raw_query)).map(
                      ([k, v], i) => (
                        <div key={i}>
                          <span>{k}</span>
                          <code>{v}</code>
                        </div>
                      ),
                    )}
                  </div>
                </>
              ) : (
                <>
                  {historyError ? (
                    <div role="alert" className="error-banner">
                      {historyError}
                    </div>
                  ) : history.length ? (
                    history.map((v) => (
                      <div className="replay-entry" key={v.id}>
                        <div>
                          <span
                            className={
                              "status-label " + (v.status_code ? "" : "failed")
                            }
                          >
                            {v.status_code || "FAILED"}
                          </span>
                          <span className="muted">{v.duration_ms} ms</span>
                        </div>
                        <code>{v.target}</code>
                        {v.error && <p className="text-danger">{v.error}</p>}
                        <small className="muted">{stamp(v.created_at)}</small>
                      </div>
                    ))
                  ) : (
                    <div className="empty-small">
                      <Clock size={24} />
                      <h3>还没有重放记录</h3>
                      <p>将此事件发送到一个公网地址，验证你的处理逻辑。</p>
                      <button
                        className="secondary"
                        onClick={() => setReplay(true)}
                      >
                        重放此事件
                      </button>
                    </div>
                  )}
                </>
              )}
            </div>
            <footer className="detail-bottom">
              <span className="live-dot" /> 已持久化到 MySQL
              <span>EVENT / {id}</span>
            </footer>
          </>
        )}
      </aside>
      {replay && event && (
        <ReplayModal
          id={id}
          onClose={() => setReplay(false)}
          onDone={() => {
            setTab("重放记录");
            setReplay(false);
            notify("重放已完成");
          }}
        />
      )}
    </>
  );
}
function ReplayModal({
  id,
  onClose,
  onDone,
}: {
  id: number;
  onClose: () => void;
  onDone: () => void;
}) {
  const [target, setTarget] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return (
    <Modal
      title="重放事件"
      onClose={() => {
        if (!busy) onClose();
      }}
    >
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          setError("");
          try {
            await api("/events/" + id + "/replay", post({ target }));
            onDone();
          } catch (e) {
            setError((e as Error).message);
          } finally {
            setBusy(false);
          }
        }}
      >
        <div className="modal-body">
          <p className="muted">
            向目标地址发送此事件的方法、Body
            和非敏感请求头。此操作会产生真实网络请求。
          </p>
          <label htmlFor="target-url">目标 URL</label>
          <input
            id="target-url"
            type="url"
            placeholder="https://api.example.com/webhook"
            autoFocus
            required
            value={target}
            onChange={(e) => setTarget(e.target.value)}
          />
          <div className="info-note">
            <ShieldCheck size={16} />
            仅允许公网 HTTP/HTTPS（80/443）；不跟随重定向，不转发凭证。原事件的
            Query 不会自动附加，请在目标 URL 中填写。
          </div>
          {error && (
            <div className="error-banner" role="alert">
              {error}
            </div>
          )}
        </div>
        <footer className="modal-footer">
          <button
            type="button"
            className="secondary"
            onClick={onClose}
            disabled={busy}
          >
            取消
          </button>
          <button className="primary" disabled={busy}>
            {busy ? (
              <Spinner />
            ) : (
              <>
                <RotateCcw size={15} />
                发送重放
              </>
            )}
          </button>
        </footer>
      </form>
    </Modal>
  );
}
