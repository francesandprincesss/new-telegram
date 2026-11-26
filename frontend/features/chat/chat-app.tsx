"use client";

import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { api, ApiError, errorText } from "../../lib/api";
import { mergeDialogs, statusLabels, type ComposerDraft, type Dialog, type Status, type User } from "../../lib/chat-data";
import { AuthForm } from "./auth-form";
import { Thread } from "./thread";
import { Avatar, Icon, IconButton, Timestamp, TypingIndicator } from "./ui";
import { useEvents } from "./use-events";

type Session = { user?: User; loading?: boolean; error?: string };

export function ChatApp() {
  const [session, setSession] = useState<Session>({ loading: true });
  const [attempt, setAttempt] = useState(0);
  const expire = useCallback(() => setSession({}), []);

  useEffect(() => {
    const controller = new AbortController();
    api.me(controller.signal).then(
      user => setSession({ user }),
      error => {
        if (controller.signal.aborted) return;
        if (error instanceof ApiError && error.status === 401) setSession({});
        else setSession({ error: errorText(error) });
      },
    );
    return () => controller.abort();
  }, [attempt]);

  if (session.user) return <Workspace key={session.user.id} user={session.user} onExpired={expire} />;
  if (session.loading) return <main className="form-shell"><p role="status">Подключение…</p></main>;
  if (session.error) {
    return (
      <main className="form-shell gap-4">
        <p className="error-text" role="alert">{session.error}</p>
        <button type="button" className="action justify-self-start"
          onClick={() => { setSession({ loading: true }); setAttempt(value => value + 1); }}>
          Повторить
        </button>
      </main>
    );
  }
  return <AuthForm onSignIn={user => setSession({ user })} />;
}

const emptyDraft: ComposerDraft = { body: "" };

function Workspace({ user, onExpired }: { user: User; onExpired: () => void }) {
  const [dialogs, setDialogs] = useState<Dialog[]>([]);
  const [token, setToken] = useState<string | null>(null);
  const [moreBusy, setMoreBusy] = useState(false);
  const [drafts, setDrafts] = useState<ReadonlyMap<string, ComposerDraft>>(() => new Map());
  const [sending, setSending] = useState<ReadonlySet<string>>(() => new Set());
  const [activeId, setActiveId] = useState<string>();
  const [query, setQuery] = useState("");
  const [threadOpen, setThreadOpen] = useState(false);
  const [details, setDetails] = useState(() => window.matchMedia("(min-width: 851px)").matches);
  const [revision, setRevision] = useState(0);
  const [listError, setListError] = useState("");
  const [accountError, setAccountError] = useState("");
  const [status, setStatus] = useState<Status>(user.status);
  const [statusBusy, setStatusBusy] = useState(false);
  const [logoutBusy, setLogoutBusy] = useState(false);
  const [finding, setFinding] = useState(false);
  const search = useRef<HTMLInputElement>(null);
  const refresh = useCallback(() => setRevision(value => value + 1), []);
  const { connected, typingDialogs } = useEvents(user.id, refresh, onExpired);
  const active = dialogs.find(dialog => dialog.id === activeId);

  function updateDraft(id: string, update: (draft: ComposerDraft) => ComposerDraft) {
    setDrafts(current => {
      const previous = current.get(id) ?? emptyDraft;
      const next = update(previous);
      if (previous === next) return current;
      const updated = new Map(current);
      if (next.body || next.replyTo) updated.set(id, next);
      else updated.delete(id);
      return updated;
    });
  }

  function trackSend(id: string, pending: boolean) {
    setSending(current => {
      const next = new Set(current);
      if (pending) next.add(id);
      else next.delete(id);
      return next;
    });
  }

  const errorMessage = useCallback((error: unknown) => {
    if (error instanceof ApiError && error.status === 401) onExpired();
    return errorText(error);
  }, [onExpired]);

  useEffect(() => {
    const controller = new AbortController();
    api.dialogs("", controller.signal).then(
      result => {
        if (controller.signal.aborted) return;
        setDialogs(value => mergeDialogs(value, result.dialogs));
        setToken(value => value ?? result.page_info.next_page_token);
        setListError("");
      },
      error => { if (!controller.signal.aborted) setListError(errorMessage(error)); },
    );
    return () => controller.abort();
  }, [revision, errorMessage]);

  useEffect(() => {
    const controller = new AbortController();
    if (connected) {
      api.me(controller.signal).then(result => {
        if (!controller.signal.aborted) setStatus(result.status);
      }, error => { if (!controller.signal.aborted) errorMessage(error); });
    }
    return () => controller.abort();
  }, [connected, errorMessage]);

  useEffect(() => {
    const wide = window.matchMedia("(min-width: 851px)");
    const resize = (event: MediaQueryListEvent) => setDetails(event.matches);
    function focusSearch(event: KeyboardEvent) {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey ||
        (event.target instanceof HTMLElement && event.target.closest("input, textarea, select, [contenteditable], dialog"))) return;
      event.preventDefault();
      setThreadOpen(false);
      requestAnimationFrame(() => search.current?.focus());
    }
    wide.addEventListener("change", resize);
    document.addEventListener("keydown", focusSearch);
    return () => {
      wide.removeEventListener("change", resize);
      document.removeEventListener("keydown", focusSearch);
    };
  }, []);

  async function loadMore() {
    if (!token || moreBusy) return;
    setMoreBusy(true);
    try {
      const result = await api.dialogs(token);
      setDialogs(value => mergeDialogs(result.dialogs, value));
      setToken(result.page_info.next_page_token);
      setListError("");
    } catch (error) {
      setListError(errorMessage(error));
    } finally {
      setMoreBusy(false);
    }
  }

  async function changeStatus(value: Exclude<Status, "OFFLINE">) {
    setStatusBusy(true);
    setAccountError("");
    try {
      setStatus((await api.status(value)).status);
    } catch (error) {
      setAccountError(errorMessage(error));
    } finally {
      setStatusBusy(false);
    }
  }

  async function logout() {
    setLogoutBusy(true);
    setAccountError("");
    try {
      await api.logout();
      onExpired();
    } catch (error) {
      setAccountError(errorMessage(error));
      setLogoutBusy(false);
    }
  }

  function openDialog(id: string) {
    setActiveId(id);
    setThreadOpen(true);
    if (window.matchMedia("(max-width: 620px)").matches) {
      requestAnimationFrame(() => document.querySelector<HTMLButtonElement>(".back")?.focus());
    }
  }

  function back() {
    setThreadOpen(false);
    setDetails(false);
    requestAnimationFrame(() => document.querySelector<HTMLButtonElement>('.chat[aria-current="true"]')?.focus());
  }

  const normalizedQuery = query.trim().replace(/^@/, "").toLowerCase();
  const visible = dialogs.filter(dialog => dialog.peer.username.toLowerCase().includes(normalizedQuery));

  return (
    <main className={`workspace ${details && active ? "" : "details-hidden"} ${threadOpen && active ? "" : "list-open"}`}>
      <aside className="sidebar" aria-label="Список чатов">
        <div className="sidebar-head">
          <div className="flex h-16 items-center justify-between gap-2">
            <h1>Чаты</h1>
            <IconButton name="plus" label="Новый чат" onClick={() => setFinding(true)} />
          </div>
          <label className="search">
            <Icon name="search" />
            <input type="search" ref={search} value={query} onChange={event => setQuery(event.target.value)}
              placeholder="Поиск по чатам" aria-label="Поиск по чатам" />
            <kbd className="keycap" aria-hidden="true">/</kbd>
          </label>
        </div>
        <div className="chat-list">
          {visible.map(dialog => {
            const draft = drafts.get(dialog.id)?.body.trim();
            const peerTyping = typingDialogs.has(dialog.id);
            return (
              <button key={dialog.id} type="button" className="chat" aria-current={active?.id === dialog.id}
                onClick={() => openDialog(dialog.id)}>
                <Avatar username={dialog.peer.username} />
                <span>
                  <span className="chat-top">
                    <span className="chat-name">{dialog.peer.username}</span>
                    {dialog.last_message && <Timestamp value={dialog.last_message.created_at} compact />}
                  </span>
                  <span className="chat-bottom">
                    <span className="chat-preview">{peerTyping
                      ? <TypingIndicator />
                      : draft
                        ? <><strong className="draft-label">Черновик</strong> {draft}</>
                        : dialog.last_message
                          ? (dialog.last_message.sender_id === user.id ? "Вы: " : "") + dialog.last_message.body
                          : "Сообщений пока нет"}</span>
                    {dialog.unread_count > 0 && <span className="badge" aria-label={`Непрочитанных: ${dialog.unread_count}`}>{dialog.unread_count}</span>}
                  </span>
                </span>
              </button>
            );
          })}
          {!visible.length && !listError && <p className="empty subtle" role="status">
            {token === null ? "Загрузка чатов…" : query.trim() ? "Чаты не найдены" : "Чатов пока нет"}
          </p>}
          {token && <button type="button" className="quiet-action justify-self-center" disabled={moreBusy} onClick={loadMore}>
            {moreBusy ? "Загрузка…" : "Показать ещё"}
          </button>}
          {listError && <div className="empty">
            <p role="alert" className="error-text">{listError}</p>
            <button type="button" className="quiet-action mt-2" onClick={refresh}>Повторить</button>
          </div>}
        </div>
        {accountError && <p className="error-text mx-5 mb-2" role="alert">{accountError}</p>}
        <div className="self">
          <Avatar username={user.username} />
          <div className="min-w-0">
            <strong className="block truncate" title={user.username}>{user.username}</strong>
            {!connected && <span className="subtle" role="status">Подключение…</span>}
          </div>
          <button type="button" className="quiet-action" disabled={logoutBusy} onClick={logout}>Выйти</button>
          <label className="sr-only" htmlFor="status">Ваш статус</label>
          <select className="status-select" id="status" value={status} disabled={statusBusy || !connected}
            onChange={event => void changeStatus(event.target.value as Exclude<Status, "OFFLINE">)}>
            {status === "OFFLINE" && <option value="OFFLINE" disabled>{statusLabels.OFFLINE}</option>}
            {(["ONLINE", "IDLE", "DND"] as const).map(value => <option key={value} value={value}>{statusLabels[value]}</option>)}
          </select>
        </div>
      </aside>
      {active
        ? <Thread key={active.id} dialog={active} userId={user.id} revision={revision} details={details}
            setDetails={setDetails} back={back} onChange={refresh} errorMessage={errorMessage}
            visibleOnMobile={threadOpen} peerTyping={typingDialogs.has(active.id)} draft={drafts.get(active.id) ?? emptyDraft}
            sending={sending.has(active.id)} updateDraft={updateDraft} trackSend={trackSend} />
        : <section className="conversation" aria-label="Переписка">
          <div className="empty-thread">
            <p className="subtle">{dialogs.length ? "Выберите чат" : "Начните новый чат"}</p>
            <button type="button" className="quiet-action justify-self-center" onClick={() => setFinding(true)}>Новый чат</button>
          </div>
        </section>}
      {finding && <NewChat userId={user.id} errorMessage={errorMessage}
        onClose={() => setFinding(false)} onOpen={dialog => {
          setDialogs(value => mergeDialogs(value, [dialog]));
          setQuery("");
          setFinding(false);
          openDialog(dialog.id);
          refresh();
        }} />}
    </main>
  );
}

function NewChat({ userId, onClose, onOpen, errorMessage }: {
  userId: string;
  onClose: () => void;
  onOpen: (dialog: Dialog) => void;
  errorMessage: (error: unknown) => string;
}) {
  const modal = useRef<HTMLDialogElement>(null);
  const pending = useRef<AbortController | null>(null);
  const [query, setQuery] = useState("");
  const [result, setResult] = useState<User>();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const dialog = modal.current;
    dialog?.showModal();
    return () => { pending.current?.abort(); dialog?.close(); };
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const username = query.trim().replace(/^@/, "");
    if (!result && !/^[a-z0-9_]{3,32}$/i.test(username)) {
      setError("Введите точное имя пользователя");
      return;
    }
    const controller = new AbortController();
    pending.current = controller;
    setBusy(true);
    setError("");
    try {
      if (result) {
        const dialog = await api.createDialog(result.id, controller.signal);
        if (!controller.signal.aborted) onOpen(dialog);
      } else {
        const found = await api.find(username, controller.signal);
        if (controller.signal.aborted) return;
        if (found.id === userId) setError("Это ваш аккаунт");
        else setResult(found);
      }
    } catch (error) {
      if (!controller.signal.aborted) setError(errorMessage(error));
    } finally {
      if (!controller.signal.aborted) setBusy(false);
    }
  }

  return (
    <dialog ref={modal} aria-labelledby="new-chat-title" onCancel={event => { event.preventDefault(); onClose(); }}>
      <div className="mb-3 flex items-center justify-between gap-2">
        <h2 id="new-chat-title" className="text-base font-medium">Новый чат</h2>
        <IconButton name="close" label="Закрыть поиск" onClick={onClose} />
      </div>
      <form className="grid gap-4" onSubmit={submit}>
        <label className="form-label">Имя пользователя
          <input className="field" placeholder="@username" value={query} disabled={busy}
            autoCapitalize="none" spellCheck={false} maxLength={33} required
            onChange={event => { setQuery(event.target.value); setResult(undefined); setError(""); }} />
        </label>
        {result && <div className="flex items-center gap-3" role="status"><Avatar username={result.username} /><span>@{result.username}</span></div>}
        {error && <p className="error-text" role="alert">{error}</p>}
        <button type="submit" className="action" disabled={busy}>{busy ? "Подождите…" : result ? "Открыть чат" : "Найти"}</button>
      </form>
    </dialog>
  );
}
