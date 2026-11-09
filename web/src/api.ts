export type User = { id: number; email: string };
export type Endpoint = {
  id: number;
  name: string;
  token: string;
  active: boolean;
  created_at: string;
  event_count: number;
};
export type Event = {
  id: number;
  endpoint_id: number;
  endpoint_name: string;
  method: string;
  path: string;
  raw_query: string;
  size: number;
  source_ip: string;
  received_at: string;
  headers?: Record<string, string[]>;
  body_base64?: string;
};
export type Replay = {
  id: number;
  target: string;
  status_code: number;
  duration_ms: number;
  error: string;
  created_at: string;
};
let csrf = "";
export function setCSRF(value: string) {
  csrf = value;
}
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch("/api" + path, {
    ...init,
    credentials: "same-origin",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": csrf,
      ...init.headers,
    },
  });
  if (!res.ok) {
    let message = "请求失败，请稍后重试";
    try {
      message = (await res.json()).error || message;
    } catch {
      /* proxy error */
    }
    if (res.status === 401)
      window.dispatchEvent(new CustomEvent("session-expired"));
    throw new Error(message);
  }
  return res.status === 204 ? (undefined as T) : res.json();
}
export const post = (data: unknown): RequestInit => ({
  method: "POST",
  body: JSON.stringify(data),
});
export function textBody(event: Event) {
  return new TextDecoder("utf-8").decode(
    Uint8Array.from(atob(event.body_base64 || ""), (c) => c.charCodeAt(0)),
  );
}
export function size(n: number) {
  return n < 1024 ? n + " B" : (n / 1024).toFixed(1) + " KB";
}
export function stamp(s: string) {
  return new Date(s).toLocaleString("zh-CN", { hour12: false });
}
