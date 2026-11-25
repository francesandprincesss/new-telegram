"use client";

import { useState, type FormEvent } from "react";
import { api, errorText } from "../../lib/api";
import { validateCredentials, type User } from "../../lib/chat-data";

export function AuthForm({ onSignIn }: { onSignIn: (user: User) => void }) {
  const [mode, setMode] = useState<"login" | "register">("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const registering = mode === "register";

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const name = username.trim();
    const validation = validateCredentials(name, password, registering);
    setError(validation);
    if (validation) return;
    setBusy(true);
    try {
      await api.authenticate(mode, name, password);
      setPassword("");
      onSignIn(await api.me());
    } catch (error) {
      setError(errorText(error));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="form-shell">
      <h1 className="mb-4 !h-auto">{registering ? "Создать аккаунт" : "Вход"}</h1>
      <form className="grid gap-4" onSubmit={submit}>
        <label className="form-label">Имя пользователя
          <input className="field" value={username} onChange={e => setUsername(e.target.value)}
            autoComplete="username" autoCapitalize="none" spellCheck={false} maxLength={32} required />
        </label>
        <label className="form-label">Пароль
          <input className="field" type="password" value={password} onChange={e => setPassword(e.target.value)}
            autoComplete={registering ? "new-password" : "current-password"}
            aria-describedby={registering ? "registration-password-hint" : undefined} required />
          {registering && <small id="registration-password-hint">От 8 до 72 символов</small>}
        </label>
        {error && <p className="error-text" role="alert">{error}</p>}
        <button type="submit" className="action" disabled={busy}>
          {busy ? "Подождите…" : registering ? "Зарегистрироваться" : "Войти"}
        </button>
      </form>
      <button type="button" className="quiet-action mt-3 justify-self-start" disabled={busy}
        onClick={() => { setError(""); setMode(registering ? "login" : "register"); }}>
        {registering ? "Уже есть аккаунт" : "Создать аккаунт"}
      </button>
    </main>
  );
}
