import { NextRequest, NextResponse } from "next/server";

const cookieName = "chat_session";
const uuid = "[a-fA-F0-9-]{36}";
const routes: Record<string, RegExp> = {
  GET: new RegExp(`^(?:users/(?:me|find|${uuid})|dialogs(?:/${uuid}/messages)?|events)$`),
  POST: new RegExp(`^(?:auth/(?:login|register)|dialogs(?:/${uuid}/(?:messages|read))?)$`),
  PUT: new RegExp(`^(?:presence/status|messages/${uuid}/reaction)$`),
  DELETE: new RegExp(`^(?:auth/session|messages/${uuid}/reaction)$`),
};

function failure(status: number, error: string) {
  return NextResponse.json({ error }, { status, headers: { "Cache-Control": "no-store" } });
}

async function proxy(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  const endpoint = path.join("/");
  const method = request.method;
  if (!routes[method]?.test(endpoint)) return failure(404, "Маршрут не найден");

  const origin = request.headers.get("origin");
  const publicOrigin = process.env.CHAT_APP_ORIGIN ||
    `${request.nextUrl.protocol}//${request.headers.get("host") ?? request.nextUrl.host}`;
  if (method !== "GET" && ((origin && origin !== publicOrigin) || request.headers.get("sec-fetch-site") === "cross-site")) {
    return failure(403, "Запрос с другого сайта отклонён");
  }

  const cookieOptions = {
    httpOnly: true, sameSite: "strict" as const, path: "/api/v1",
    secure: publicOrigin.startsWith("https://"),
  };
  if (endpoint === "auth/session") {
    const response = new NextResponse(null, { status: 204 });
    response.cookies.set(cookieName, "", { ...cookieOptions, maxAge: 0 });
    return response;
  }

  const auth = endpoint.startsWith("auth/");
  const token = request.cookies.get(cookieName)?.value;
  if (!auth && !token) return failure(401, "Войдите в аккаунт");

  const headers = new Headers({ accept: endpoint === "events" ? "text/event-stream" : "application/json" });
  let body: string | undefined;
  if (method === "POST" || method === "PUT") {
    if (request.headers.get("content-type")?.split(";")[0] !== "application/json") {
      return failure(415, "Ожидается JSON");
    }
    // A bounded reader also covers requests without Content-Length.
    const reader = request.body?.getReader();
    const decoder = new TextDecoder();
    let size = 0;
    body = "";
    if (reader) {
      try {
        for (;;) {
          const { value, done } = await reader.read();
          if (done) break;
          size += value.byteLength;
          if (size > 64 * 1024) {
            await reader.cancel();
            return failure(413, "Данные слишком велики");
          }
          body += decoder.decode(value, { stream: true });
        }
        body += decoder.decode();
      } finally {
        reader.releaseLock();
      }
    }
    headers.set("content-type", "application/json");
  }
  if (!auth) headers.set("authorization", `Bearer ${token}`);

  let upstream: Response;
  try {
    const url = new URL(`/api/v1/${endpoint}`, process.env.CHAT_API_ORIGIN ?? "http://127.0.0.1:8080");
    url.search = request.nextUrl.search;
    const signal = endpoint === "events"
      ? request.signal
      : AbortSignal.any([request.signal, AbortSignal.timeout(17_000)]);
    upstream = await fetch(url, { method, headers, body, signal, redirect: "error", cache: "no-store" });
  } catch {
    return failure(502, "Сервер чатов недоступен");
  }

  if (auth && upstream.ok) {
    try {
      const { access_token, expires_at } = await upstream.json();
      const expires = new Date(expires_at);
      if (typeof access_token !== "string" || !access_token || !Number.isFinite(expires.getTime())) {
        return failure(502, "Некорректный ответ сервера");
      }
      const response = NextResponse.json({ expires_at }, {
        status: upstream.status, headers: { "Cache-Control": "no-store" },
      });
      response.cookies.set(cookieName, access_token, { ...cookieOptions, expires });
      return response;
    } catch {
      return failure(502, "Некорректный ответ сервера");
    }
  }

  const response = new NextResponse(upstream.body, {
    status: upstream.status,
    headers: {
      "Content-Type": upstream.headers.get("content-type") ?? "application/json",
      "Cache-Control": "no-store",
      "X-Accel-Buffering": "no",
    },
  });
  if (upstream.status === 401 && !auth) {
    response.cookies.set(cookieName, "", { ...cookieOptions, maxAge: 0 });
  }
  return response;
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const DELETE = proxy;
