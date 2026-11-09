import { useState } from "react";
import type { FormEvent } from "react";
import {
  ArrowRight,
  Webhook,
  Terminal,
  ShieldCheck,
  Activity,
  Braces,
} from "lucide-react";
import { api, post, setCSRF } from "./api";
import type { User } from "./api";
import { Logo, Spinner } from "./ui";
export default function Auth({ onAuth }: { onAuth: (u: User) => void }) {
  const [register, setRegister] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const data = await api<{ user: User; csrf: string }>(
        "/auth/" + (register ? "register" : "login"),
        post({ email, password }),
      );
      setCSRF(data.csrf);
      onAuth(data.user);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="auth-page">
      <aside className="auth-story">
        <Logo />
        <div className="auth-pitch">
          <span className="eyebrow">
            <span className="live-dot" /> BUILT FOR YOUR NEXT BREAKTHROUGH
          </span>
          <h1>
            每一个事件，
            <br />
            都有迹可循<span className="accent">.</span>
          </h1>
          <p>
            捕获、检查、重放。
            <br />让 Webhook 调试回到你的掌控之中。
          </p>
          <div className="terminal-card">
            <div className="terminal-top">
              <span />
              <span />
              <span />
              <small>terminal — hooklab</small>
              <Terminal size={14} />
            </div>
            <pre>
              <span className="muted">$</span> curl -X POST{" "}
              <span className="accent">$HOOKLAB_URL</span>
              {" \\\n"} -H{" "}
              <span className="json-string">
                'Content-Type: application/json'
              </span>
              {" \\\n"} -d{" "}
              <span className="json-string">'{`{"event":"hello.world"}`}'</span>
              {"\n\n"}
              <span className="accent">↳ 202 Accepted</span>
              {"\n"}
              <span className="muted"> 事件已捕获。每个细节，清晰可见。</span>
            </pre>
            <span className="terminal-signal">
              <Activity size={14} /> Listening for possibilities
            </span>
          </div>
          <div className="auth-features">
            <span>
              <ShieldCheck size={16} />
              安全隔离
            </span>
            <span>
              <Braces size={16} />
              原始数据
            </span>
            <span>
              <Webhook size={16} />
              任意 HTTP 方法
            </span>
          </div>
        </div>
        <footer>
          THE SMALL TOOL FOR THE BIG PICTURE.<span>HookLab / v0.1</span>
        </footer>
      </aside>
      <main className="auth-form-panel">
        <div className="auth-mobile-logo">
          <Logo />
        </div>
        <div className="auth-form">
          <div className="section-icon">
            <Webhook size={24} />
          </div>
          <span className="eyebrow">YOUR WEBHOOK COMMAND CENTER</span>
          <h2>{register ? "开始你的调试之旅" : "欢迎回来"}</h2>
          <p className="muted">
            {register
              ? "创建账户，连接你的第一个 Webhook。"
              : "登录你的账户，继续探索事件背后的细节。"}
          </p>
          <form onSubmit={submit}>
            <label htmlFor="email">邮箱地址</label>
            <input
              id="email"
              type="email"
              placeholder="you@company.com"
              autoComplete="email"
              required
              maxLength={254}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <div className="label-row">
              <label htmlFor="password">密码</label>
              <small>至少 12 个字符</small>
            </div>
            <input
              id="password"
              type="password"
              placeholder="输入你的密码"
              autoComplete={register ? "new-password" : "current-password"}
              required
              minLength={12}
              maxLength={128}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            {error && (
              <div className="error-banner" role="alert">
                {error}
              </div>
            )}
            <button className="primary auth-submit" disabled={busy}>
              {busy ? (
                <Spinner />
              ) : (
                <>
                  {register ? "注册并进入工作台" : "登录工作台"}
                  <ArrowRight size={17} />
                </>
              )}
            </button>
          </form>
          <p className="auth-switch">
            {register ? "已经有账户？" : "还没有账户？"}
            <button
              onClick={() => {
                setRegister(!register);
                setError("");
              }}
            >
              {register ? "返回登录" : "创建账户"}
            </button>
          </p>
          <div className="auth-note">
            <ShieldCheck size={14} /> Argon2id 密码加密 · 安全会话
          </div>
        </div>
        <span className="auth-corner">少一点猜测，多一点确定。</span>
      </main>
    </div>
  );
}
