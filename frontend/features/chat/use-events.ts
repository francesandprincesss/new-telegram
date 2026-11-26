"use client";

import { useEffect, useState } from "react";
import { readEventStream } from "../../lib/sse";

const typingTimeout = 5_000;

export function useEvents(userId: string, onRefresh: () => void, onExpired: () => void) {
  const [connected, setConnected] = useState(false);
  const [typingDialogs, setTypingDialogs] = useState<ReadonlySet<string>>(() => new Set());

  useEffect(() => {
    const controller = new AbortController();
    let retry: ReturnType<typeof setTimeout>;
    let refresh: ReturnType<typeof setTimeout> | undefined;
    let wait = 1000;
    const typing = new Map<string, ReturnType<typeof setTimeout>>();

    function clearTyping(dialogId: string) {
      if (!typing.has(dialogId)) return;
      clearTimeout(typing.get(dialogId));
      typing.delete(dialogId);
      setTypingDialogs(new Set(typing.keys()));
    }

    function resetTyping() {
      for (const timer of typing.values()) clearTimeout(timer);
      typing.clear();
    }

    function handleTypingEvent(name: string, data: string) {
      let payload: { dialog_id?: unknown; user_id?: unknown; sender_id?: unknown } | undefined;
      try {
        payload = JSON.parse(data)?.payload;
      } catch {
        return;
      }
      const dialogId = payload?.dialog_id;
      if (typeof dialogId !== "string") return;
      if (name === "message.created") {
        if (typeof payload?.sender_id === "string" && payload.sender_id !== userId) clearTyping(dialogId);
        return;
      }
      if (typeof payload?.user_id !== "string" || payload.user_id === userId) return;
      const previous = typing.get(dialogId);
      clearTimeout(previous);
      typing.set(dialogId, setTimeout(() => clearTyping(dialogId), typingTimeout));
      if (!previous) setTypingDialogs(new Set(typing.keys()));
    }

    function scheduleRefresh() {
      if (refresh) return;
      refresh = setTimeout(() => {
        refresh = undefined;
        onRefresh();
      }, 120);
    }

    async function connect() {
      try {
        const response = await fetch("/api/v1/events", { signal: controller.signal, cache: "no-store" });
        if (response.status === 401) {
          onExpired();
          return;
        }
        if (!response.ok || !response.body || !response.headers.get("content-type")?.startsWith("text/event-stream")) {
          throw new Error("SSE unavailable");
        }

        wait = 1000;
        setConnected(true);
        scheduleRefresh();
        await readEventStream(response.body, (name, data) => {
          if (controller.signal.aborted) return;
          if (name === "dialog.typing" || (name === "message.created" && typing.size)) {
            handleTypingEvent(name, data);
          }
          if (["message.created", "messages.read", "reaction.changed", "presence.changed"].includes(name)) {
            scheduleRefresh();
          }
        });
      } catch {
        // Network errors and closed streams both use the reconnect path.
      }
      if (controller.signal.aborted) return;
      setConnected(false);
      if (typing.size) {
        resetTyping();
        setTypingDialogs(new Set());
      }
      retry = setTimeout(connect, wait);
      wait = Math.min(wait * 2, 15_000);
    }

    void connect();
    return () => {
      controller.abort();
      clearTimeout(retry);
      clearTimeout(refresh);
      resetTyping();
    };
  }, [userId, onRefresh, onExpired]);

  return { connected, typingDialogs };
}
