import assert from "node:assert/strict";
import test from "node:test";
import { mergeMessages, validateCredentials, validateMessage, type Message } from "../lib/chat-data.ts";
import { eventStreamParser, readEventStream } from "../lib/sse.ts";

test("history deduplicates by ID and preserves PostgreSQL timestamp order", () => {
  const message = (id: string, created_at: string): Message => ({
    id, created_at, dialog_id: "chat", sender_id: "user", body: "Сообщение", read_at: null, reactions: [],
  });
  const first = message("c", "2026-11-25T10:15:05Z");
  const second = message("b", "2026-11-25T10:15:05.000100Z");
  const third = message("a", "2026-11-25T10:15:05.000101Z");
  const read = { ...second, read_at: "2026-11-25T10:17:00Z" };
  assert.deepEqual(mergeMessages([second, third], [read, first]), [first, read, third]);
});

test("registration counts Unicode code points while login accepts short legacy passwords", () => {
  assert.ok(validateCredentials("ab", "abcdefgh", true));
  assert.equal(validateCredentials("user_12", "я".repeat(8), true), "");
  assert.equal(validateCredentials("user_12", "🙂".repeat(72), true), "");
  assert.ok(validateCredentials("user_12", "🙂".repeat(73), true));
  assert.ok(validateCredentials("user_12", "🙂".repeat(7), true));
  assert.equal(validateCredentials("user_12", "🙂🙂", false), "");
});

test("message validation counts trimmed Unicode code points", () => {
  assert.ok(validateMessage(" \n "));
  assert.equal(validateMessage(" " + "🙂".repeat(4000) + " "), "");
  assert.ok(validateMessage("🙂".repeat(4001)));
});

test("SSE parses split CRLF frames, comments and multiple data lines", () => {
  const received: string[][] = [];
  const parse = eventStreamParser((name, data) => received.push([name, data]));
  parse(": connected\r\n\r");
  parse("\nevent: message.created\r\ndata: one\r\ndata: ");
  parse("two\r\n\r\n");
  assert.deepEqual(received, [["message.created", "one\ntwo"]]);
});

test("SSE decodes UTF-8 split between chunks and discards incomplete frames", async () => {
  const encoded = new TextEncoder().encode("event: reaction.changed\ndata: Привет\n\ndata: incomplete\n");
  const splitter = encoded.findIndex(byte => byte > 127) + 1;
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      controller.enqueue(encoded.slice(0, splitter));
      controller.enqueue(encoded.slice(splitter));
      controller.close();
    },
  });
  const received: string[][] = [];
  await readEventStream(stream, (name, data) => received.push([name, data]));
  assert.deepEqual(received, [["reaction.changed", "Привет"]]);
});
