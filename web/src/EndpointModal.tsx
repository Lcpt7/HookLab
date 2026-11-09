import { useState } from "react";
import type { FormEvent } from "react";
import { Trash2, AlertTriangle, ArrowRight } from "lucide-react";
import { api, post } from "./api";
import type { Endpoint } from "./api";
import { Modal, Spinner } from "./ui";
export default function EndpointModal({
  endpoint,
  onClose,
  onSaved,
}: {
  endpoint?: Endpoint;
  onClose: () => void;
  onSaved: (id?: number) => Promise<void>;
}) {
  const [name, setName] = useState(endpoint?.name || "");
  const [active, setActive] = useState(endpoint?.active ?? true);
  const [busy, setBusy] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const [error, setError] = useState("");
  async function save(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const v = endpoint
        ? await api<Endpoint>("/endpoints/" + endpoint.id, {
            method: "PATCH",
            body: JSON.stringify({ name, active }),
          })
        : await api<Endpoint>("/endpoints", post({ name }));
      await onSaved(v.id);
      onClose();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function remove() {
    if (!endpoint) return;
    setBusy(true);
    setError("");
    try {
      await api("/endpoints/" + endpoint.id, { method: "DELETE" });
      await onSaved();
      onClose();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      title={endpoint ? "管理 Endpoint" : "新建 Endpoint"}
      onClose={() => {
        if (!busy) onClose();
      }}
    >
      <form onSubmit={save}>
        <div className="modal-body">
          {confirm ? (
            <div className="delete-confirm">
              <AlertTriangle size={30} />
              <h3>永久删除 {endpoint?.name}？</h3>
              <p>
                该 Endpoint 及其全部事件和重放记录将被永久删除。此操作无法撤销。
              </p>
            </div>
          ) : (
            <>
              <p className="muted">
                {endpoint
                  ? "更新名称或暂停接收。已有事件会保留。"
                  : "一个独立的接收地址，连接你的每一次事件。"}
              </p>
              <label htmlFor="endpoint-name">Endpoint 名称</label>
              <input
                id="endpoint-name"
                autoFocus
                required
                maxLength={80}
                placeholder="例如 Stripe production"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
              <small className="muted">
                使用清晰的服务名称，方便快速识别。
              </small>
              {endpoint ? (
                <label className="toggle-row">
                  <span>
                    <strong>接收 Webhook</strong>
                    <small>关闭后，新的请求将返回 404。</small>
                  </span>
                  <input
                    type="checkbox"
                    checked={active}
                    onChange={(e) => setActive(e.target.checked)}
                  />
                </label>
              ) : (
                <div className="info-note">
                  自动生成 256 位随机 Token，支持所有 HTTP 方法。
                </div>
              )}
            </>
          )}
          {error && (
            <div role="alert" className="error-banner">
              {error}
            </div>
          )}
        </div>
        <footer className="modal-footer">
          {confirm ? (
            <>
              <button
                type="button"
                className="secondary"
                onClick={() => setConfirm(false)}
                disabled={busy}
              >
                返回
              </button>
              <button
                type="button"
                className="danger"
                onClick={remove}
                disabled={busy}
              >
                {busy ? <Spinner /> : "确认永久删除"}
              </button>
            </>
          ) : (
            <>
              {endpoint && (
                <button
                  type="button"
                  className="text-danger"
                  onClick={() => setConfirm(true)}
                >
                  <Trash2 size={15} />
                  删除 Endpoint
                </button>
              )}
              <div className="spacer" />
              <button
                type="button"
                className="secondary"
                onClick={onClose}
                disabled={busy}
              >
                取消
              </button>
              <button className="primary" disabled={busy || !name.trim()}>
                {busy ? (
                  <Spinner />
                ) : (
                  <>
                    {endpoint ? "保存修改" : "创建 Endpoint"}
                    <ArrowRight size={16} />
                  </>
                )}
              </button>
            </>
          )}
        </footer>
      </form>
    </Modal>
  );
}
