"use client";

import { Fragment, useEffect, useLayoutEffect, useRef, useState, type FormEvent } from "react";
import { api } from "../../lib/api";
import {
  mergeMessages,
  reactions,
  validateMessage,
  type Dialog,
  type Message,
  type ReactionKind,
} from "../../lib/chat-data";
import { Avatar, dayLabel, Icon, IconButton, Presence, Timestamp } from "./ui";

type ThreadProps = {
  dialog: Dialog;
  userId: string;
  revision: number;
  details: boolean;
  setDetails: (open: boolean) => void;
  back: () => void;
  onChange: () => void;
  errorMessage: (error: unknown) => string;
  visibleOnMobile: boolean;
};

export function Thread({
  dialog, userId, revision, details, setDetails, back, onChange, errorMessage, visibleOnMobile,
}: ThreadProps) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [token, setToken] = useState<string | null>(null);
  const [historyError, setHistoryError] = useState("");
  const [moreBusy, setMoreBusy] = useState(false);
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const [actionError, setActionError] = useState("");
  const [openReaction, setOpenReaction] = useState<string>();
  const [updatingReaction, setUpdatingReaction] = useState<string>();
  const [atEnd, setAtEnd] = useState(true);
  const [viewing, setViewing] = useState(false);
  const [retry, setRetry] = useState(0);
  const pane = useRef<HTMLDivElement>(null);
  const input = useRef<HTMLTextAreaElement>(null);
  const info = useRef<HTMLButtonElement>(null);
  const pages = useRef(1);
  const loadedMessages = useRef<Message[]>([]);
  const latestRead = useRef("");
  const scroll = useRef<{ bottom?: true; height?: number; top?: number } | null>(null);
  const { id, peer } = dialog;

  useEffect(() => {
    const controller = new AbortController();
    const pageCount = pages.current;

    async function load() {
      let next = "";
      const current: Message[] = [];
      try {
        for (let page = 0; page < pageCount; page++) {
          const result = await api.messages(id, next, controller.signal);
          current.push(...result.messages);
          next = result.page_info.next_page_token;
          if (!next) break;
        }
        if (controller.signal.aborted) return;
        const list = pane.current;
        if (list && list.scrollHeight - list.scrollTop - list.clientHeight <= 48) scroll.current = { bottom: true };
        if (pages.current === pageCount) {
          // Keep the loaded tail while the first pages shift on a live message.
          // An empty overlap after reconnect means the old and fresh windows have a gap.
          const fresh = new Set(current.map(message => message.id));
          const overlaps = loadedMessages.current.some(message => fresh.has(message.id));
          setMessages(value => mergeMessages(overlaps ? value : [], current));
          if (!overlaps || !next) setToken(next);
          else setToken(value => value ?? next);
        } else {
          setMessages(value => mergeMessages(value, current));
        }
        setHistoryError("");
      } catch (error) {
        if (!controller.signal.aborted) setHistoryError(errorMessage(error));
      }
    }

    void load();
    return () => controller.abort();
  }, [id, revision, retry, errorMessage]);

  useLayoutEffect(() => {
    loadedMessages.current = messages;
    const list = pane.current;
    if (!list || !list.clientHeight) return;
    const position = scroll.current;
    scroll.current = null;
    if (position?.bottom) list.scrollTop = list.scrollHeight;
    if (position?.height !== undefined && position.top !== undefined) {
      list.scrollTop = position.top + list.scrollHeight - position.height;
    }
  }, [messages, visibleOnMobile]);

  useEffect(() => {
    const compact = matchMedia("(max-width: 620px)");
    function update() {
      setViewing(!document.hidden && document.hasFocus() && (!compact.matches || visibleOnMobile));
    }
    // Evaluate from browser events and once after layout.
    const frame = requestAnimationFrame(update);
    compact.addEventListener("change", update);
    document.addEventListener("visibilitychange", update);
    window.addEventListener("focus", update);
    window.addEventListener("blur", update);
    return () => {
      cancelAnimationFrame(frame);
      compact.removeEventListener("change", update);
      document.removeEventListener("visibilitychange", update);
      window.removeEventListener("focus", update);
      window.removeEventListener("blur", update);
    };
  }, [visibleOnMobile]);

  const unread = messages.findLast(message => message.sender_id !== userId && !message.read_at)?.id;
  useEffect(() => {
    if (!viewing || !atEnd || !unread || latestRead.current === unread) return;
    latestRead.current = unread;
    // The request may finish even if the reader moves to a different thread.
    api.read(id, unread).then(onChange, error => {
      if (latestRead.current === unread) latestRead.current = "";
      setActionError(errorMessage(error));
    });
  }, [viewing, atEnd, unread, id, onChange, errorMessage]);

  async function older() {
    if (!token || moreBusy) return;
    setMoreBusy(true);
    try {
      const result = await api.messages(id, token);
      const list = pane.current;
      if (list) scroll.current = { height: list.scrollHeight, top: list.scrollTop };
      pages.current++;
      setMessages(value => mergeMessages(result.messages, value));
      setToken(result.page_info.next_page_token);
      setHistoryError("");
    } catch (error) {
      setHistoryError(errorMessage(error));
    } finally {
      setMoreBusy(false);
    }
  }

  async function send(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!draft.trim() || sending) return;
    const validation = validateMessage(draft);
    setActionError(validation);
    if (validation) return;
    const body = draft.trim();
    setSending(true);
    try {
      const message = await api.send(id, body);
      scroll.current = { bottom: true };
      setMessages(value => mergeMessages(value, [message]));
      setDraft(value => value.trim() === body ? "" : value);
      onChange();
    } catch (error) {
      setActionError(errorMessage(error));
    } finally {
      setSending(false);
      input.current?.focus();
    }
  }

  async function changeReaction(message: Message, reaction: ReactionKind) {
    if (updatingReaction) return;
    const kind = message.reactions.some(item => item.user_id === userId && item.reaction === reaction) ? null : reaction;
    setActionError("");
    setUpdatingReaction(message.id);
    try {
      const result = await api.react(message.id, kind);
      const applied = "reaction" in result ? [result] : [];
      setMessages(value => value.map(item => item.id === message.id
        ? { ...item, reactions: [...item.reactions.filter(entry => entry.user_id !== userId), ...applied] }
        : item));
      setOpenReaction(undefined);
      onChange();
    } catch (error) {
      setActionError(errorMessage(error));
    } finally {
      setUpdatingReaction(undefined);
    }
  }

  return (
    <>
      <section className="conversation" aria-label="Переписка">
        <header className="thread-head">
          <IconButton name="back" label="К списку чатов" className="back" onClick={back} />
          <Avatar username={peer.username} />
          <div className="thread-user">
            <h2>{peer.username}</h2>
            <div className="subtle"><Presence status={peer.status} /></div>
          </div>
          <IconButton ref={info} name="details" label="Сведения о чате" className="info"
            aria-controls="chat-details" aria-expanded={details} onClick={() => setDetails(!details)} />
        </header>
        <div className="messages" ref={pane} aria-label="Сообщения" role="log" aria-live="polite"
          onScroll={event => {
            const list = event.currentTarget;
            setAtEnd(list.scrollHeight - list.scrollTop - list.clientHeight <= 48);
          }}>
          {token && <button type="button" className="quiet-action self-center" disabled={moreBusy} onClick={older}>
            {moreBusy ? "Загрузка…" : "Предыдущие сообщения"}
          </button>}
          {historyError && <div className="self-center text-center">
            <p className="error-text" role="alert">{historyError}</p>
            <button type="button" className="quiet-action" onClick={() => setRetry(value => value + 1)}>Повторить</button>
          </div>}
          {!messages.length && !historyError && <p className="subtle self-center" role="status">
            {token === null ? "Загрузка сообщений…" : "Сообщений пока нет"}
          </p>}
          {messages.map((message, index) => {
            const sent = message.sender_id === userId;
            const day = dayLabel(message.created_at);
            const showDay = index === 0 || day !== dayLabel(messages[index - 1].created_at);
            const menuOpen = openReaction === message.id;

            return (
              <Fragment key={message.id}>
                {showDay && <div className="day">{day}</div>}
                <article className={`message ${sent ? "sent" : ""}`} aria-label={sent ? "Вы" : peer.username}>
                  <p>{message.body}</p>
                  <div className="message-meta">
                    <Timestamp value={message.created_at} />
                    {sent && <span className={`icon icon-check ${message.read_at ? "is-read" : ""}`} role="img"
                      aria-label={message.read_at ? "Прочитано" : "Отправлено"} title={message.read_at ? "Прочитано" : "Отправлено"} />}
                    <span className="message-menu">
                      <IconButton name="more" label="Реакции" aria-expanded={menuOpen}
                        onClick={() => setOpenReaction(menuOpen ? undefined : message.id)} />
                    </span>
                  </div>
                  {(menuOpen || message.reactions.length > 0) && <div className="reaction-panel" aria-label="Реакции">
                    {reactions.map(({ kind, icon, label }) => {
                      const entries = message.reactions.filter(value => value.reaction === kind);
                      if (!menuOpen && !entries.length) return null;
                      return <button key={kind} type="button" className="reaction" title={label}
                        aria-label={entries.length ? `${label} (${entries.length})` : label}
                        aria-pressed={entries.some(entry => entry.user_id === userId)} disabled={!!updatingReaction}
                        onClick={() => changeReaction(message, kind)}>
                        <span aria-hidden="true">{icon}</span>
                        {entries.length > 0 && <span aria-hidden="true">{entries.length}</span>}
                      </button>;
                    })}
                  </div>}
                </article>
              </Fragment>
            );
          })}
        </div>
        <div className="composer-area">
          {actionError && <p className="error-text mb-2" role="alert">{actionError}</p>}
          <form className="composer" onSubmit={send}>
            <textarea rows={1} ref={input} value={draft} onChange={event => setDraft(event.target.value)}
              onKeyDown={event => {
                if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
                  event.preventDefault();
                  event.currentTarget.form?.requestSubmit();
                }
              }} aria-label="Сообщение" placeholder="Сообщение" maxLength={8000} />
            <button className="send" type="submit" aria-label="Отправить сообщение" disabled={sending || !draft.trim()}>
              <Icon name="up" />
            </button>
          </form>
        </div>
      </section>
      <aside className="details" id="chat-details" aria-label="Сведения о чате" hidden={!details}>
        <div className="details-head mono">Сведения о чате
          <IconButton name="close" label="Закрыть сведения" onClick={() => { setDetails(false); info.current?.focus(); }} />
        </div>
        <div className="profile">
          <Avatar username={peer.username} />
          <h2>{peer.username}</h2>
          <span className="mono">@{peer.username}</span>
        </div>
        <dl className="profile-meta">
          <div><dt>Статус</dt><dd><Presence status={peer.status} /></dd></div>
          {peer.status === "OFFLINE" && peer.last_seen_at && (
            <div><dt>Последний визит</dt><dd><Timestamp value={peer.last_seen_at} compact /></dd></div>
          )}
        </dl>
      </aside>
    </>
  );
}
