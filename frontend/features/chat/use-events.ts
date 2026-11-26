"use client";

import { useEffect, useState } from "react";
import { readEventStream } from "../../lib/sse";

export function useEvents(onRefresh: () => void, onExpired: () => void) {
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    let retry: ReturnType<typeof setTimeout>;
    let refresh: ReturnType<typeof setTimeout> | undefined;
    let wait = 1000;

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
        await readEventStream(response.body, name => {
          if (["message.created", "messages.read", "reaction.changed", "presence.changed"].includes(name)) {
            scheduleRefresh();
          }
        });
      } catch {
        // Network errors and closed streams both use the reconnect path.
      }
      if (controller.signal.aborted) return;
      setConnected(false);
      retry = setTimeout(connect, wait);
      wait = Math.min(wait * 2, 15_000);
    }

    void connect();
    return () => {
      controller.abort();
      clearTimeout(retry);
      clearTimeout(refresh);
    };
  }, [onRefresh, onExpired]);

  return connected;
}
