export type Status = "ONLINE" | "IDLE" | "DND" | "OFFLINE";
export type ReactionKind = "LIKE" | "LOVE" | "LAUGH" | "WOW";

export interface User {
  id: string;
  username: string;
  status: Status;
  last_seen_at: string | null;
}

export interface Reaction {
  message_id: string;
  user_id: string;
  reaction: ReactionKind;
  created_at: string;
  updated_at: string;
}

export type MessageQuote = { id: string; sender_id: string; excerpt: string };

export type ComposerDraft = { body: string; replyTo?: MessageQuote };

export interface Message {
  id: string;
  dialog_id: string;
  sender_id: string;
  body: string;
  created_at: string;
  read_at: string | null;
  reactions: Reaction[];
  reply_to?: MessageQuote;
}

export interface Dialog {
  id: string;
  peer: User;
  last_message: Message | null;
  unread_count: number;
  created_at: string;
  updated_at: string;
}

export type PageInfo = { next_page_token: string };

export const statusLabels: Record<Status, string> = {
  ONLINE: "В сети",
  IDLE: "Отошёл",
  DND: "Не беспокоить",
  OFFLINE: "Не в сети",
};

export const reactions: { kind: ReactionKind; icon: string; label: string }[] = [
  { kind: "LIKE", icon: "👍", label: "Нравится" },
  { kind: "LOVE", icon: "❤️", label: "Любовь" },
  { kind: "LAUGH", icon: "😄", label: "Смешно" },
  { kind: "WOW", icon: "😮", label: "Удивление" },
];

function compareTime(a: string, b: string) {
  // Date truncates the fractional part past milliseconds, but PostgreSQL stores microseconds.
  const submillis = (value: string) => (value.match(/\.([0-9]+)/)?.[1] ?? "").padEnd(9, "0").slice(3);
  return Date.parse(a) - Date.parse(b) || submillis(a).localeCompare(submillis(b));
}

export function mergeMessages(previous: Message[], fresh: Message[]) {
  const messages = new Map(previous.map(message => [message.id, message]));
  for (const message of fresh) {
    const knownReadAt = messages.get(message.id)?.read_at;
    messages.set(message.id, knownReadAt && !message.read_at
      ? { ...message, read_at: knownReadAt }
      : message);
  }
  return [...messages.values()].sort((a, b) =>
    compareTime(a.created_at, b.created_at) || a.id.localeCompare(b.id)
  );
}

export function applyReadReceipt(messages: Message[], cutoffId: string, readerId: string, readAt: string) {
  const cutoff = messages.findIndex(message => message.id === cutoffId);
  if (cutoff < 0) return messages;
  return messages.map((message, index) => index <= cutoff && message.sender_id !== readerId && !message.read_at
    ? { ...message, read_at: readAt }
    : message);
}

export function mergeDialogs(previous: Dialog[], fresh: Dialog[]) {
  const dialogs = new Map(previous.map(dialog => [dialog.id, dialog]));
  for (const dialog of fresh) dialogs.set(dialog.id, dialog);
  return [...dialogs.values()].sort((a, b) =>
    compareTime(b.updated_at, a.updated_at) || b.id.localeCompare(a.id)
  );
}

export function validateCredentials(username: string, password: string, registering: boolean) {
  if (!/^[A-Za-z0-9_]{3,32}$/.test(username)) {
    return "Имя пользователя должно содержать 3–32 латинские буквы, цифры или подчёркивания";
  }
  const length = Array.from(password).length;
  if (registering && (length < 8 || length > 72)) return "Длина пароля должна быть от 8 до 72 символов";
  if (!registering && (length === 0 || length > 72)) return "Проверьте имя пользователя и пароль";
  return "";
}

export function validateMessage(body: string) {
  if (!body.trim()) return "Введите сообщение";
  if ([...body.trim()].length > 4000) return "Сообщение должно содержать не более 4000 символов";
  return "";
}
