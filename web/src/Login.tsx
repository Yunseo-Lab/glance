import SourceLink from "./workspace/SourceLink";
import { useState } from "react";
import { LockKeyhole, LoaderCircle } from "lucide-react";
import { request } from "./types";

export default function Login({ onSuccess }: { onSuccess: () => void }) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  async function login(event: React.SubmitEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      await request("/api/v1/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ password }),
      });
      setPassword("");
      onSuccess();
    } catch (err) {
      setPassword("");
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <main className="login-page">
      <form className="login-card" onSubmit={login}>
        <div className="brand">
          <div className="brand-mark">
            g<span>·</span>
          </div>
          <span>glance.</span>
        </div>
        <span className="login-symbol">
          <LockKeyhole size={25} />
        </span>
        <h1>나의 공간에 로그인</h1>
        <p>설정한 비밀번호로 대시보드를 열어보세요.</p>
        <label htmlFor="owner-password">비밀번호</label>
        <input
          id="owner-password"
          type="password"
          autoComplete="current-password"
          required
          maxLength={4096}
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          autoFocus
          disabled={busy}
        />
        {error && (
          <p className="error-text" role="alert">
            {error}
          </p>
        )}
        <button className="button primary" disabled={busy} type="submit">
          {busy ? (
            <LoaderCircle size={16} className="spin" />
          ) : (
            <LockKeyhole size={16} />
          )}{" "}
          로그인
        </button>
        <small>로그인 세션은 12시간 후 만료됩니다.</small>
        <small>
          <SourceLink />
        </small>
      </form>
    </main>
  );
}
