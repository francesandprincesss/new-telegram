import type { Dialog, Message, PageInfo, Reaction, ReactionKind, Status, User } from "./chat-data";

export class ApiError extends Error {
  constructor(readonly status: number, message: string) {
    super(message);
  }
}

const errors: Record<number, string> = {
  400: "Проверьте введённые данные",
  401: "Сессия закончилась. Войдите снова",
  403: "Доступ к этой операции закрыт",
  404: "Запись не найдена",
  409: "Такое имя пользователя уже занято",
  413: "Данные слишком велики",
  429: "Слишком много запросов. Попробуйте позже",
  502: "Сервер чатов недоступен",
};

export function errorText(error: unknown) {
  if (error instanceof ApiError) return error.message;
  return "Не удалось связаться с сервером. Попробуйте ещё раз";
}

async function request<T>(path: string, options: { method?: string; data?: object; signal?: AbortSignal } = {}): Promise<T> {
  const { method = "GET", data, signal } = options;
  const response = await fetch("/api/v1" + path, {
    method, signal, cache: "no-store",
    headers: data ? { "Content-Type": "application/json" } : undefined,
    body: data ? JSON.stringify(data) : undefined,
  });
  if (!response.ok) {
    const message = path === "/auth/login" && response.status === 401
      ? "Неверное имя пользователя или пароль"
      : path.startsWith("/users/find?") && response.status === 404
        ? "Пользователь не найден"
        : errors[response.status] ?? "Сервер не смог выполнить запрос";
    throw new ApiError(response.status, message);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

function page(token: string) {
  return "?" + new URLSearchParams({ page_size: "40", ...(token ? { page_token: token } : {}) });
}

export const api = {
  authenticate: (mode: "login" | "register", username: string, password: string) =>
    request<{ expires_at: string }>("/auth/" + mode, { method: "POST", data: { username, password } }),
  logout: () => request<void>("/auth/session", { method: "DELETE" }),
  me: (signal?: AbortSignal) => request<User>("/users/me", { signal }),
  find: (username: string, signal?: AbortSignal) =>
    request<User>("/users/find?" + new URLSearchParams({ username }), { signal }),
  dialogs: (token = "", signal?: AbortSignal) =>
    request<{ dialogs: Dialog[]; page_info: PageInfo }>("/dialogs" + page(token), { signal }),
  createDialog: (userId: string, signal?: AbortSignal) =>
    request<Dialog>("/dialogs", { method: "POST", data: { other_user_id: userId }, signal }),
  messages: (id: string, token = "", signal?: AbortSignal) =>
    request<{ messages: Message[]; page_info: PageInfo }>(`/dialogs/${id}/messages` + page(token), { signal }),
  send: (id: string, body: string) => request<Message>(`/dialogs/${id}/messages`, { method: "POST", data: { body } }),
  read: (id: string, messageId: string, signal?: AbortSignal) =>
    request<unknown>(`/dialogs/${id}/read`, { method: "POST", data: { up_to_message_id: messageId }, signal }),
  react: (id: string, reaction: ReactionKind | null) => reaction
    ? request<Reaction>(`/messages/${id}/reaction`, { method: "PUT", data: { reaction } })
    : request<{ removed: boolean }>(`/messages/${id}/reaction`, { method: "DELETE" }),
  status: (status: Exclude<Status, "OFFLINE">) =>
    request<{ user_id: string; status: Status }>("/presence/status", { method: "PUT", data: { status } }),
};
