import assert from "node:assert/strict";
import { randomBytes } from "node:crypto";
import { setTimeout as pause } from "node:timers/promises";
import { readEventStream } from "../lib/sse.ts";

const base = new URL(process.env.CHAT_TEST_ORIGIN ?? "http://127.0.0.1:3000").origin;
const suffix = randomBytes(5).toString("hex");
const password = "🔒".repeat(72 - suffix.length) + suffix;

function client() {
  let cookie = "";
  return async (path: string, method = "GET", data?: object, origin = base, signal = AbortSignal.timeout(5000)) => {
    const headers = new Headers();
    if (cookie) headers.set("cookie", cookie);
    if (method !== "GET") headers.set("origin", origin);
    if (data) headers.set("content-type", "application/json");
    const response = await fetch(base + "/api/v1" + path, {
      headers, method, signal, body: data ? JSON.stringify(data) : undefined,
    });
    const setCookie = response.headers.get("set-cookie");
    if (setCookie) cookie = setCookie.split(";")[0];
    return response;
  };
}

async function json<T>(response: Response, code: number): Promise<T> {
  assert.equal(response.status, code, new URL(response.url).pathname);
  return response.json() as Promise<T>;
}

type Request = ReturnType<typeof client>;
type Identity = { id: string; username: string };
type Dialog = { id: string; unread_count: number; peer: Identity; last_message: { id: string } | null };
type Message = {
  id: string; body: string; read_at: string | null; reactions: { user_id: string; reaction: string }[];
  reply_to?: { id: string; sender_id: string; excerpt: string };
};
type MessagePage = { messages: Message[]; page_info: { next_page_token: string } };

async function register(request: Request, prefix: string) {
  const response = await request("/auth/register", "POST", { username: prefix + suffix, password });
  assert.match(response.headers.get("set-cookie") ?? "", /HttpOnly/i);
  assert.match(response.headers.get("set-cookie") ?? "", /SameSite=Strict/i);
  const auth = await json<{ expires_at: string; access_token?: string }>(response, 201);
  assert.ok(auth.expires_at);
  assert.equal(auth.access_token, undefined);
  return json<Identity>(await request("/users/me"), 200);
}

async function listen(request: Request) {
  const controller = new AbortController();
  const response = await request("/events", "GET", undefined, base, AbortSignal.any([
    controller.signal, AbortSignal.timeout(20_000),
  ]));
  assert.equal(response.status, 200);
  assert.match(response.headers.get("content-type") ?? "", /text\/event-stream/);
  assert.ok(response.body);
  const events: { name: string; payload: Record<string, string> }[] = [];
  const pump = readEventStream(response.body, (name, data) => events.push({ name, payload: JSON.parse(data).payload }))
    .catch(error => { if (!controller.signal.aborted) throw error; });
  return {
    close: async () => { controller.abort(); await pump; },
    has: async (name: string, key: string, value: string) => {
      const until = Date.now() + 4000;
      while (Date.now() < until) {
        if (events.some(event => event.name === name && event.payload[key] === value)) return;
        await pause(30);
      }
      assert.fail(`SSE event ${name} was not received`);
    },
  };
}

const alice = client();
const bob = client();
const outsider = client();

assert.equal((await alice("/dialogs")).status, 401);
assert.equal((await alice("/auth/login", "POST", { username: "alice", password }, "https://example.org")).status, 403);
await json(await alice("/auth/register", "POST", { username: "short_" + suffix, password: "🙂".repeat(7) }), 400);
await json(await alice("/auth/register", "POST", { username: "long_" + suffix, password: password + "x" }), 400);
const a = await register(alice, "alice_");
const signIn = client();
await json(await signIn("/auth/login", "POST", { username: a.username, password: password.slice(0, -1) + "🔑" }), 401);
await json(await signIn("/auth/login", "POST", { username: a.username, password }), 200);
const b = await register(bob, "bob_");
const other = await register(outsider, "other_");
assert.equal((await json<Identity>(await alice("/users/find?username=" + b.username.toUpperCase()), 200)).id, b.id);
const dialog = await json<Dialog>(await alice("/dialogs", "POST", { other_user_id: b.id }), 200);
const fromBob = await json<Dialog>(await bob("/dialogs", "POST", { other_user_id: a.id }), 200);
assert.equal(fromBob.id, dialog.id);
assert.equal(dialog.peer.id, b.id);

const streamA = await listen(alice);
let streamB: Awaited<ReturnType<typeof listen>> | undefined;
try {
  streamB = await listen(bob);
  const path = `/dialogs/${dialog.id}`;
  assert.equal((await outsider(path + "/messages")).status, 403);
  assert.equal((await outsider(path + "/typing", "POST", {})).status, 403);
  assert.equal((await alice(path + "/typing", "POST", {})).status, 204);
  await streamB.has("dialog.typing", "dialog_id", dialog.id);
  await streamB.has("dialog.typing", "user_id", a.id);
  const first = await json<Message>(await alice(path + "/messages", "POST", { body: "Первое сообщение" }), 201);
  await streamB.has("message.created", "message_id", first.id);
  const anotherDialog = await json<Dialog>(await bob("/dialogs", "POST", { other_user_id: other.id }), 200);
  const wrongDialog = await json(await bob(`/dialogs/${anotherDialog.id}/messages`, "POST", {
    body: "Чужая цитата", reply_to_message_id: first.id,
  }), 404);
  const missingQuote = await json(await bob(path + "/messages", "POST", {
    body: "Нет оригинала", reply_to_message_id: "10000000-0000-4000-8000-000000000004",
  }), 404);
  assert.deepEqual(wrongDialog, missingQuote);
  assert.equal((await bob(path + "/messages", "POST", { body: "Ответ", reply_to_message_id: "invalid" })).status, 400);
  assert.equal((await outsider(path + "/messages", "POST", { body: "Чужой чат", reply_to_message_id: first.id })).status, 403);
  const reply = await json<Message>(await bob(path + "/messages", "POST", { body: "Ответ", reply_to_message_id: first.id }), 201);
  const expectedQuote = { id: first.id, sender_id: a.id, excerpt: first.body };
  assert.deepEqual(reply.reply_to, expectedQuote);
  await streamA.has("message.created", "message_id", reply.id);
  await streamA.has("message.created", "reply_to_message_id", first.id);
  const list = await json<{ dialogs: Dialog[] }>(await bob("/dialogs"), 200);
  assert.equal(list.dialogs[0].last_message?.id, reply.id);
  assert.equal(list.dialogs[0].unread_count, 1);

  const page = await json<MessagePage>(await bob(path + "/messages?page_size=1"), 200);
  assert.equal(page.messages[0].id, reply.id);
  assert.deepEqual(page.messages[0].reply_to, expectedQuote);
  const next = new URLSearchParams({ page_size: "1", page_token: page.page_info.next_page_token });
  const older = await json<MessagePage>(await bob(path + "/messages?" + next), 200);
  assert.equal(older.messages[0].id, first.id);

  const read = await json<{ marked_count: number }>(await bob(path + "/read", "POST", { up_to_message_id: first.id }), 200);
  assert.equal(read.marked_count, 1);
  await streamA.has("messages.read", "up_to_message_id", first.id);
  assert.equal((await json<{ marked_count: number }>(
    await bob(path + "/read", "POST", { up_to_message_id: first.id }), 200,
  )).marked_count, 0);
  const reactionPath = `/messages/${first.id}/reaction`;
  assert.equal((await outsider(reactionPath, "PUT", { reaction: "LOVE" })).status, 403);
  assert.equal((await bob(reactionPath, "PUT", { reaction: "LOVE" })).status, 200);
  await streamA.has("reaction.changed", "message_id", first.id);
  const history = await json<MessagePage>(await alice(path + "/messages"), 200);
  assert.ok(history.messages[1].read_at);
  assert.deepEqual(history.messages[1].reactions.map(({ user_id, reaction }) => [user_id, reaction]), [[b.id, "LOVE"]]);
  assert.equal((await json<{ removed: boolean }>(await bob(reactionPath, "DELETE"), 200)).removed, true);

  assert.equal((await bob("/presence/status", "PUT", { status: "DND" })).status, 200);
  await streamA.has("presence.changed", "status", "DND");
  await streamB.close();
  streamB = undefined;
  await streamA.has("presence.changed", "status", "OFFLINE");
  assert.equal((await bob("/auth/session", "DELETE")).status, 204);
  assert.equal((await bob("/users/me")).status, 401);
} finally {
  await Promise.all([streamA.close(), streamB?.close()]);
}

assert.equal((await bob("/auth/login", "POST", { username: b.username, password })).status, 200);
assert.equal((await json<Identity>(await bob("/users/me"), 200)).id, b.id);
console.log("Passed: sessions, CSRF, user search, dialogue identity, membership, SSE, message history, quoted replies, typing, read receipts, reactions and presence.");
