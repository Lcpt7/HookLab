import { useEffect, useState } from "react";
import { setCSRF } from "./api";
import type { User } from "./api";
import Auth from "./Auth";
import Workspace from "./Workspace";
import { Logo, Spinner } from "./ui";
export default function App() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  useEffect(() => {
    const expired = () => {
      setUser(null);
      setCSRF("");
    };
    window.addEventListener("session-expired", expired);
    fetch("/api/auth/me", { credentials: "same-origin" })
      .then(async (res) => {
        if (res.status === 401) return;
        if (!res.ok) throw new Error("无法连接服务，请确认 API 已启动");
        const data = await res.json();
        setCSRF(data.csrf);
        setUser(data.user);
      })
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
    return () => window.removeEventListener("session-expired", expired);
  }, []);
  if (loading)
    return (
      <div className="loading-screen">
        <Logo />
        <Spinner />
      </div>
    );
  if (error)
    return (
      <div className="loading-screen">
        <Logo />
        <p role="alert">{error}</p>
        <button className="primary" onClick={() => location.reload()}>
          重新连接
        </button>
      </div>
    );
  return user ? (
    <Workspace
      user={user}
      onLogout={() => {
        setUser(null);
        setCSRF("");
      }}
    />
  ) : (
    <Auth onAuth={setUser} />
  );
}
