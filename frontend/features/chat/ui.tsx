import type { ComponentPropsWithRef } from "react";
import { statusLabels, type Status } from "../../lib/chat-data";

type IconName = "search" | "back" | "up" | "down" | "reply" | "details" | "close" | "check" | "plus" | "more";

export function Icon({ name }: { name: IconName }) {
  return <span className={`icon icon-${name}`} aria-hidden="true" />;
}

export function IconButton({ name, label, className = "", ...props }: ComponentPropsWithRef<"button"> & {
  name: IconName;
  label: string;
}) {
  return (
    <button type="button" className={`icon-button ${className}`} aria-label={label} title={label} {...props}>
      <Icon name={name} />
    </button>
  );
}

export function Avatar({ username }: { username: string }) {
  return <span className="avatar" aria-hidden="true">{username.slice(0, 2).toUpperCase()}</span>;
}

export function Presence({ status }: { status: Status }) {
  return (
    <span>
      {status === "ONLINE" && <span className="status-dot" aria-hidden="true" />}
      {statusLabels[status] ?? statusLabels.OFFLINE}
    </span>
  );
}

const timeFormat = new Intl.DateTimeFormat("ru-RU", { hour: "2-digit", minute: "2-digit" });
const dayFormat = new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "long", year: "numeric" });
const shortDate = new Intl.DateTimeFormat("ru-RU", { day: "numeric", month: "short" });

export function dayLabel(value: string) {
  const date = new Date(value);
  const today = new Date();
  return date.toDateString() === today.toDateString() ? "Сегодня" : dayFormat.format(date);
}

export function Timestamp({ value, compact = false }: { value: string; compact?: boolean }) {
  const date = new Date(value);
  const format = compact && date.toDateString() !== new Date().toDateString() ? shortDate : timeFormat;
  return <time dateTime={value} title={`${dayLabel(value)}, ${timeFormat.format(date)}`}>{format.format(date)}</time>;
}
