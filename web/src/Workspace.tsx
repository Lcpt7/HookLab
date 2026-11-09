import { useCallback, useEffect, useState } from "react";
import {
  Webhook,
  LayoutDashboard,
  Radio,
  Plus,
  ChevronDown,
  ArrowUpRight,
  ArrowDownLeft,
  ArrowRight,
  Search,
  SlidersHorizontal,
  RefreshCw,
  MoreHorizontal,
  Terminal,
  BookOpen,
  LogOut,
  Command,
  Check,
  Box,
  Activity,
  Layers,
  ChevronRight,
  Menu,
  X,
  ShieldCheck,
  ExternalLink,
} from "lucide-react";
import { api, size } from "./api";
import type { Endpoint, Event, User } from "./api";
import { Logo, CopyButton, Method, Spinner } from "./ui";
import EndpointModal from "./EndpointModal";
import Detail from "./Detail";
type View = "workspace" | "events" | "endpoints" | "guide";
export default function Workspace({
  user,
  onLogout,
}: {
  user: User;
  onLogout: () => void;
}) {
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [selected, setSelected] = useState<number | null>(null);
  const [view, setView] = useState<View>("workspace");
  const [events, setEvents] = useState<Event[]>([]);
  const [loading, setLoading] = useState(true);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState("");
  const [connected, setConnected] = useState(false);
  const [epoch, setEpoch] = useState(0);
  const [q, setQ] = useState("");
  const [method, setMethod] = useState("");
  const [before, setBefore] = useState(0);
  const [cursors, setCursors] = useState<number[]>([]);
  const [next, setNext] = useState(0);
  const [detail, setDetail] = useState<number | null>(null);
  const [modal, setModal] = useState<"new" | Endpoint | null>(null);
  const [toast, setToast] = useState("");
  const [mobile, setMobile] = useState(false);
  const notify = useCallback((s: string) => setToast(s), []);
  const closeDetail = useCallback(() => setDetail(null), []);
  const ep = endpoints.find((e) => e.id === selected);
  const url = ep ? location.origin + "/h/" + ep.token : "";
  const all = view === "events";
  const loadEndpoints = useCallback(async (id?: number) => {
    const data = await api<{ endpoints: Endpoint[] }>("/endpoints");
    setEndpoints(data.endpoints);
    setSelected((prev) =>
      id && data.endpoints.some((e) => e.id === id)
        ? id
        : data.endpoints.some((e) => e.id === prev)
          ? prev
          : (data.endpoints[0]?.id ?? null),
    );
    setLoaded(true);
  }, []);
  useEffect(() => {
    loadEndpoints().catch((e) => {
      setError(e.message);
      setLoaded(true);
    });
  }, [loadEndpoints, epoch]);
  useEffect(() => {
    const tick = setInterval(() => setEpoch((v) => v + 1), 5000);
    return () => clearInterval(tick);
  }, []);
  useEffect(() => {
    if (!loaded) return;
    let live = true;
    const timer = setTimeout(
      () => {
        const params = new URLSearchParams();
        if (!all && selected) params.set("endpoint_id", String(selected));
        if (q) params.set("q", q);
        if (method) params.set("method", method);
        if (before) params.set("before", String(before));
        setLoading(true);
        api<{ events: Event[]; next_cursor: number }>("/events?" + params)
          .then((data) => {
            if (live) {
              setEvents(data.events);
              setNext(data.next_cursor);
              setError("");
              setConnected(true);
            }
          })
          .catch((e) => {
            if (live) {
              setError(e.message);
              setConnected(false);
            }
          })
          .finally(() => {
            if (live) setLoading(false);
          });
      },
      q ? 250 : 0,
    );
    return () => {
      live = false;
      clearTimeout(timer);
    };
  }, [loaded, selected, all, q, method, before, epoch]);
  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => setToast(""), 3500);
    return () => clearTimeout(t);
  }, [toast]);
  function navigate(v: View) {
    setView(v);
    setMobile(false);
    setQ("");
    setMethod("");
    setBefore(0);
    setCursors([]);
  }
  function choose(id: number) {
    setSelected(id);
    setBefore(0);
    setCursors([]);
    setQ("");
    setMethod("");
    setView("workspace");
  }
  const total = endpoints.reduce((a, e) => a + e.event_count, 0);
  const active = endpoints.filter((e) => e.active).length;
  return (
    <div className="app-shell">
      {mobile && (
        <div className="sidebar-scrim" onClick={() => setMobile(false)} />
      )}
      <aside className={"sidebar " + (mobile ? "open" : "")}>
        <Logo />
        <button
          className="workspace-picker"
          onClick={() => navigate("workspace")}
        >
          <span className="workspace-avatar">P</span>
          <span>
            个人工作空间<small>Personal workspace</small>
          </span>
          <ChevronDown size={15} />
        </button>
        <span className="nav-caption">WORKSPACE</span>
        <nav aria-label="主导航">
          {(
            [
              { id: "workspace", icon: LayoutDashboard, name: "工作台" },
              { id: "events", icon: Radio, name: "全部事件" },
              { id: "endpoints", icon: Webhook, name: "Endpoint 管理" },
            ] as const
          ).map((v) => (
            <button
              key={v.id}
              className={view === v.id ? "nav-item active" : "nav-item"}
              onClick={() => navigate(v.id)}
            >
              <v.icon size={18} />
              <span>{v.name}</span>
              {v.id === "events" && <small>{total}</small>}
              {view === v.id && <span className="nav-indicator" />}
            </button>
          ))}
        </nav>
        <div className="sidebar-endpoints">
          <div className="sidebar-label">
            <span>YOUR ENDPOINTS</span>
            <button aria-label="添加 Endpoint" onClick={() => setModal("new")}>
              <Plus size={15} />
            </button>
          </div>
          {endpoints.slice(0, 6).map((e) => (
            <button
              key={e.id}
              className={
                "sidebar-endpoint " +
                (e.id === selected && view === "workspace" ? "selected" : "")
              }
              onClick={() => choose(e.id)}
            >
              <span
                className={e.active ? "endpoint-dot" : "endpoint-dot paused"}
              />
              <span>{e.name}</span>
              <small>{e.event_count}</small>
            </button>
          ))}
          {!endpoints.length && (
            <span className="sidebar-empty">你的连接，将从这里开始。</span>
          )}
        </div>
        <div className="sidebar-bottom">
          <div className="build-card">
            <Terminal size={19} />
            <strong>从第一个请求开始</strong>
            <p>一个 curl 命令，即刻打通你的事件链路。</p>
            <button onClick={() => navigate("guide")}>
              查看快速开始
              <ArrowUpRight size={14} />
            </button>
          </div>
          <button
            className={"nav-item " + (view === "guide" ? "active" : "")}
            onClick={() => navigate("guide")}
          >
            <BookOpen size={18} />
            使用指南
            <ArrowUpRight size={13} />
          </button>
          <div className="profile">
            <span className="avatar">{user.email[0].toUpperCase()}</span>
            <div>
              <strong>{user.email.split("@")[0]}</strong>
              <small>个人账户</small>
            </div>
            <button
              className="icon-button"
              aria-label="退出登录"
              title="退出登录"
              onClick={async () => {
                try {
                  await api("/auth/logout", { method: "POST" });
                  onLogout();
                } catch (e) {
                  notify((e as Error).message);
                }
              }}
            >
              <LogOut size={17} />
            </button>
          </div>
        </div>
      </aside>
      <main className="main-shell">
        <header className="topbar">
          <button
            className="icon-button mobile-menu"
            onClick={() => setMobile(true)}
            aria-label="打开导航"
          >
            <Menu size={19} />
          </button>
          <div className="breadcrumb">
            <span>个人工作空间</span>
            <ChevronRight size={13} />
            <strong>
              {view === "events"
                ? "全部事件"
                : view === "endpoints"
                  ? "Endpoint 管理"
                  : view === "guide"
                    ? "使用指南"
                    : "工作台"}
            </strong>
          </div>
          <div className="topbar-right">
            <span className={"connection " + (!connected ? "offline" : "")}>
              <span className="live-dot" />
              {connected ? "已连接 API" : "正在连接"}
            </span>
            <span className="topbar-divider" />
            <span className="version">v0.1.0</span>
            <span className="avatar mini">{user.email[0].toUpperCase()}</span>
          </div>
        </header>
        <div className="main-content">
          <section className="page-heading">
            <div>
              <span className="eyebrow">OBSERVE. INSPECT. ITERATE.</span>
              <h1>
                {view === "events"
                  ? "全部事件"
                  : view === "endpoints"
                    ? "Endpoint 管理"
                    : view === "guide"
                      ? "快速开始"
                      : "Webhook 工作台"}
                <span className="heading-dot" aria-hidden="true">
                  .
                </span>
              </h1>
              <p>
                {view === "events"
                  ? "所有连接的事件，都在这里汇聚。"
                  : view === "endpoints"
                    ? "为每个服务，建立一个清晰的事件入口。"
                    : view === "guide"
                      ? "从接收第一个事件，到验证你的处理逻辑。"
                      : "连接你的服务，让每一个 Webhook 清晰可见。"}
              </p>
            </div>
            <button className="primary" onClick={() => setModal("new")}>
              <Plus size={17} />
              新建 Endpoint
            </button>
          </section>
          {view !== "guide" && (
            <section className="stats-grid">
              <div className="stat-card">
                <div className="stat-label">
                  <span>已捕获事件</span>
                  <Activity size={17} />
                </div>
                <div className="stat-value">
                  {total.toLocaleString()}
                  <span className="stat-pill">
                    <ArrowDownLeft size={12} />
                    已持久化
                  </span>
                </div>
                <div className="stat-bottom">
                  <span>来自你的所有 Endpoint</span>
                  <svg viewBox="0 0 100 24" aria-hidden="true">
                    <path d="M0 20H17L23 12L30 19H42L50 4L57 18H75L82 10L90 14H100" />
                  </svg>
                </div>
              </div>
              <div className="stat-card">
                <div className="stat-label">
                  <span>Endpoint</span>
                  <Layers size={17} />
                </div>
                <div className="stat-value">
                  {endpoints.length.toString().padStart(2, "0")}
                  <span className="stat-subvalue">/ 100</span>
                </div>
                <div className="stat-bottom">
                  <span>独立 Token，独立事件流</span>
                  <span className="mini-bars">▂ ▄ ▃ ▆ ▄ ▇ ▅</span>
                </div>
              </div>
              <div className="stat-card">
                <div className="stat-label">
                  <span>正在接收</span>
                  <Radio size={17} />
                </div>
                <div className="stat-value">
                  {active.toString().padStart(2, "0")}
                  <span className="status-text">
                    <span className="live-dot" /> Listening
                  </span>
                </div>
                <div className="stat-bottom">
                  <span>{endpoints.length - active} 个 Endpoint 已暂停</span>
                  <span className="signal-bars">
                    <i />
                    <i />
                    <i />
                    <i />
                  </span>
                </div>
              </div>
            </section>
          )}
          {error && (
            <div className="error-banner" role="alert">
              {error}
              <button onClick={() => setEpoch((v) => v + 1)}>重试</button>
            </div>
          )}
          {view === "guide" ? (
            <Guide url={url} onNew={() => setModal("new")} />
          ) : view === "endpoints" ? (
            <div className="endpoint-grid">
              {endpoints.map((e) => (
                <article key={e.id} className="endpoint-overview">
                  <div>
                    <span className="endpoint-card-icon">
                      <Webhook size={21} />
                    </span>
                    <span
                      className={"status-label " + (!e.active ? "paused" : "")}
                    >
                      {e.active ? "ACTIVE" : "PAUSED"}
                    </span>
                  </div>
                  <h2>{e.name}</h2>
                  <p>
                    {e.event_count} 个事件 ·{" "}
                    {new Date(e.created_at).toLocaleDateString("zh-CN")}
                  </p>
                  <code>/h/{e.token.slice(0, 18)}…</code>
                  <footer>
                    <button className="quiet" onClick={() => choose(e.id)}>
                      查看事件
                      <ArrowRight size={14} />
                    </button>
                    <button
                      className="icon-button"
                      aria-label={"管理 " + e.name}
                      onClick={() => setModal(e)}
                    >
                      <MoreHorizontal size={19} />
                    </button>
                  </footer>
                </article>
              ))}
              <button
                className="new-endpoint-card"
                onClick={() => setModal("new")}
              >
                <Plus size={23} />
                <strong>添加一个连接</strong>
                <span>为你的下一个服务建立 Endpoint</span>
              </button>
            </div>
          ) : !loaded ? (
            <div className="panel-loading">
              <Spinner />
              正在加载你的工作空间…
            </div>
          ) : !endpoints.length ? (
            <div className="first-endpoint">
              <div className="empty-orbit">
                <span />
                <span />
                <Webhook size={36} />
                <i>
                  <Plus size={14} />
                </i>
              </div>
              <span className="eyebrow">READY WHEN YOU ARE</span>
              <h2>创建你的第一个 Endpoint</h2>
              <p>
                无需配置服务器。创建专属接收地址，
                <br />
                捕获每个请求，检查每个细节。
              </p>
              <button className="primary" onClick={() => setModal("new")}>
                <Plus size={16} />
                新建 Endpoint
              </button>
              <div className="empty-benefits">
                <span>
                  <Check size={13} />
                  支持所有 HTTP 方法
                </span>
                <span>
                  <Check size={13} />
                  原始请求完整保留
                </span>
                <span>
                  <Check size={13} />
                  敏感头自动脱敏
                </span>
              </div>
            </div>
          ) : (
            <div className={"workbench " + (all ? "all-events" : "")}>
              {!all && (
                <section className="endpoint-rail">
                  <div className="rail-header">
                    <span>ENDPOINTS</span>
                    <span>{endpoints.length.toString().padStart(2, "0")}</span>
                    <button
                      className="icon-button"
                      aria-label="添加 Endpoint"
                      onClick={() => setModal("new")}
                    >
                      <Plus size={15} />
                    </button>
                  </div>
                  <div className="rail-list">
                    {endpoints.map((e) => (
                      <button
                        key={e.id}
                        onClick={() => choose(e.id)}
                        className={
                          "endpoint-item " + (e.id === selected ? "active" : "")
                        }
                      >
                        <div className="endpoint-item-top">
                          <span className="rail-icon">
                            <Webhook size={16} />
                          </span>
                          <span
                            className={
                              e.active ? "endpoint-dot" : "endpoint-dot paused"
                            }
                          />
                        </div>
                        <strong>{e.name}</strong>
                        <span className="endpoint-item-bottom">
                          <code>{e.token.slice(0, 10)}…</code>
                          <span>{e.event_count} 事件</span>
                        </span>
                      </button>
                    ))}
                  </div>
                  <button className="rail-add" onClick={() => setModal("new")}>
                    <Plus size={14} />
                    添加 Endpoint
                  </button>
                  <div className="rail-note">
                    <ShieldCheck size={15} />
                    <p>
                      你的 Endpoint 和事件
                      <br />
                      仅对你可见。
                    </p>
                  </div>
                </section>
              )}
              <div className="events-workspace">
                {!all && ep && (
                  <section className="endpoint-banner">
                    <div className="endpoint-banner-top">
                      <div>
                        <h2>{ep.name}</h2>
                        <span
                          className={
                            "status-label " + (!ep.active ? "paused" : "")
                          }
                        >
                          <span className="live-dot" />
                          {ep.active ? "接收中" : "已暂停"}
                        </span>
                      </div>
                      <button
                        className="icon-button"
                        aria-label="管理当前 Endpoint"
                        title="管理 Endpoint"
                        onClick={() => setModal(ep)}
                      >
                        <MoreHorizontal size={20} />
                      </button>
                    </div>
                    <p>将 Webhook 发送到此地址，即可实时查看请求。</p>
                    <div className="url-box">
                      <span className="url-icon">
                        <Webhook size={16} />
                      </span>
                      <code data-testid="endpoint-url">{url}</code>
                      <CopyButton
                        value={url}
                        label="复制 URL"
                        onError={notify}
                      />
                    </div>
                    <div className="endpoint-banner-bottom">
                      <span>
                        <Command size={12} /> 支持所有 HTTP 方法
                      </span>
                      <span>
                        256-bit token <ShieldCheck size={12} />
                      </span>
                    </div>
                  </section>
                )}
                <section className="event-section">
                  <div className="section-heading">
                    <h2>
                      请求历史 <span>{all ? total : ep?.event_count || 0}</span>
                    </h2>
                    <span className="auto-refresh">
                      <span className="live-dot" />每 5 秒刷新
                    </span>
                    <button
                      className={"icon-button " + (loading ? "spin-icon" : "")}
                      aria-label="刷新事件"
                      title="刷新事件"
                      onClick={() => setEpoch((v) => v + 1)}
                    >
                      <RefreshCw size={15} />
                    </button>
                  </div>
                  <div className="table-toolbar">
                    <div className="search-box">
                      <Search size={15} />
                      <input
                        aria-label="搜索事件"
                        placeholder="搜索路径或请求内容…"
                        maxLength={200}
                        value={q}
                        onChange={(e) => {
                          setQ(e.target.value);
                          setBefore(0);
                          setCursors([]);
                        }}
                      />
                      {q && (
                        <button aria-label="清空搜索" onClick={() => setQ("")}>
                          <X size={13} />
                        </button>
                      )}
                    </div>
                    <div className="method-select">
                      <SlidersHorizontal size={14} />
                      <select
                        aria-label="HTTP 方法"
                        value={method}
                        onChange={(e) => {
                          setMethod(e.target.value);
                          setBefore(0);
                          setCursors([]);
                        }}
                      >
                        <option value="">所有方法</option>
                        {Array.from(
                          new Set([
                            "GET",
                            "POST",
                            "PUT",
                            "PATCH",
                            "DELETE",
                            "HEAD",
                            "OPTIONS",
                            ...events.map((e) => e.method),
                            ...(method ? [method] : []),
                          ]),
                        ).map((m) => (
                          <option key={m}>{m}</option>
                        ))}
                      </select>
                      <ChevronDown size={13} />
                    </div>
                  </div>
                  <div className="table-scroll">
                    <table className="event-table">
                      <thead>
                        <tr>
                          <th>方法</th>
                          <th>请求路径</th>
                          {all && <th>ENDPOINT</th>}
                          <th>来源 IP</th>
                          <th>大小</th>
                          <th>接收时间</th>
                          <th />
                        </tr>
                      </thead>
                      <tbody>
                        {events.map((e) => (
                          <tr
                            key={e.id}
                            className={detail === e.id ? "selected" : ""}
                          >
                            <td>
                              <Method value={e.method} />
                            </td>
                            <td>
                              <button
                                className="event-path"
                                aria-label={
                                  "查看事件 " + e.method + " " + e.path
                                }
                                onClick={() => setDetail(e.id)}
                              >
                                {e.path}
                                {e.raw_query && <span>?{e.raw_query}</span>}
                              </button>
                              <small className="event-id">
                                evt_{e.id.toString().padStart(6, "0")}
                              </small>
                            </td>
                            {all && <td>{e.endpoint_name}</td>}
                            <td className="mono muted">{e.source_ip}</td>
                            <td className="mono muted">{size(e.size)}</td>
                            <td>
                              <span className="event-time">
                                {new Date(e.received_at).toLocaleTimeString(
                                  "zh-CN",
                                  { hour12: false },
                                )}
                              </span>
                              <small className="event-id">
                                {new Date(e.received_at).toLocaleDateString(
                                  "zh-CN",
                                )}
                              </small>
                            </td>
                            <td>
                              <button
                                className="icon-button"
                                aria-label={"打开事件 " + e.id}
                                onClick={() => setDetail(e.id)}
                              >
                                <ArrowUpRight size={15} />
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                  {!events.length && (
                    <div className="empty-events">
                      <span className="empty-event-icon">
                        <Radio size={27} />
                      </span>
                      <h3>
                        {q || method ? "没有匹配的事件" : "正在等待第一个事件"}
                      </h3>
                      <p>
                        {q || method
                          ? "试试其他关键词，或调整 HTTP 方法筛选。"
                          : "向上方 Endpoint 发送请求，它会出现在这里。"}
                      </p>
                      {q || method ? (
                        <button
                          className="quiet"
                          onClick={() => {
                            setQ("");
                            setMethod("");
                          }}
                        >
                          清除筛选
                          <ArrowRight size={13} />
                        </button>
                      ) : (
                        ep && (
                          <div className="curl-example">
                            <code>
                              <span>$</span> curl -X POST '{url}' <span>…</span>
                            </code>
                            <CopyButton
                              value={`curl -X POST '${url}' -H 'Content-Type: application/json' -d '{"event":"hello.world"}'`}
                              label="复制命令"
                              onError={notify}
                            />
                          </div>
                        )
                      )}
                    </div>
                  )}
                  <footer className="table-footer">
                    <span>
                      {loading ? (
                        <>
                          <Spinner />
                          正在同步…
                        </>
                      ) : (
                        <>
                          <span className="small-check">
                            <Check size={11} />
                          </span>
                          已同步 · {events.length} 条事件
                          {before > 0 ? " · 历史页" : ""}
                        </>
                      )}
                    </span>
                    <div>
                      <button
                        disabled={!cursors.length}
                        aria-label="上一页"
                        onClick={() => {
                          setBefore(cursors[cursors.length - 1] || 0);
                          setCursors((c) => c.slice(0, -1));
                        }}
                      >
                        上一页
                      </button>
                      <button
                        disabled={!next}
                        aria-label="下一页"
                        onClick={() => {
                          setCursors((c) => [...c, before]);
                          setBefore(next);
                        }}
                      >
                        下一页
                        <ChevronRight size={13} />
                      </button>
                    </div>
                  </footer>
                </section>
              </div>
            </div>
          )}
          <footer className="page-footer">
            <span>
              <span className="live-dot" />
              HookLab <span className="muted">— 为开发者而造</span>
            </span>
            <span>
              <ShieldCheck size={12} />
              敏感请求头默认脱敏 <i /> 时间按浏览器本地时区显示
            </span>
          </footer>
        </div>
      </main>
      {modal && (
        <EndpointModal
          endpoint={modal === "new" ? undefined : modal}
          onClose={() => setModal(null)}
          onSaved={async (id) => {
            await loadEndpoints(id);
            setEpoch((v) => v + 1);
            setView("workspace");
            setBefore(0);
            setCursors([]);
            notify(id ? "Endpoint 已保存" : "Endpoint 已删除");
          }}
        />
      )}
      {detail !== null && (
        <Detail id={detail} onClose={closeDetail} notify={notify} />
      )}{" "}
      {toast && (
        <div className="toast" role="status">
          <Check size={16} />
          {toast}
          <button onClick={() => setToast("")} aria-label="关闭通知">
            <X size={14} />
          </button>
        </div>
      )}
    </div>
  );
}
function Guide({ url, onNew }: { url: string; onNew: () => void }) {
  const example = url || "http://localhost:8080/h/YOUR_TOKEN";
  return (
    <div className="guide">
      <section>
        <span className="guide-number">01</span>
        <div>
          <h2>建立你的事件入口</h2>
          <p>
            为 Stripe、GitHub 或自己的服务创建独立
            Endpoint。每个地址包含一个随机
            Token，请像密钥一样保管，不要放入公开仓库。
          </p>
          <button className="secondary" onClick={onNew}>
            <Plus size={14} />
            新建 Endpoint
          </button>
        </div>
      </section>
      <section>
        <span className="guide-number">02</span>
        <div>
          <h2>发送第一个 Webhook</h2>
          <p>把 Endpoint URL 配置到服务提供商，或在终端运行以下命令。</p>
          <pre>
            curl -X POST '{example}' {"\\\n"} -H 'Content-Type:
            application/json' {"\\\n"} -d '
            {`{"event":"hello.world","source":"terminal"}`}'
          </pre>
          <CopyButton
            value={`curl -X POST '${example}' -H 'Content-Type: application/json' -d '{"event":"hello.world","source":"terminal"}'`}
            label="复制命令"
          />
        </div>
      </section>
      <section>
        <span className="guide-number">03</span>
        <div>
          <h2>检查每一个细节</h2>
          <p>
            点击事件查看格式化 JSON、原始 Body、Headers 和 Query。可按 HTTP
            方法筛选、搜索路径与内容，或下载完整二进制 Body。
          </p>
          <div className="info-note">
            <Box size={16} />
            默认请求体上限为 1 MiB，敏感头在落库前脱敏。Query 和 Body
            会原样保存，请避免发送生产凭证。
          </div>
        </div>
      </section>
      <section>
        <span className="guide-number">04</span>
        <div>
          <h2>重放并验证</h2>
          <p>
            在事件详情中点击「重放事件」，填写公网 HTTP/HTTPS
            地址。重放不会转发凭证，不会自动附加原始
            Query，不会跟随重定向。结果保存在重放记录中。
          </p>
          <a className="quiet" href="/healthz" target="_blank" rel="noreferrer">
            检查服务健康状态
            <ExternalLink size={13} />
          </a>
        </div>
      </section>
    </div>
  );
}
