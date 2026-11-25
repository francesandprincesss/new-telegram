export function eventStreamParser(onEvent: (name: string, data: string) => void) {
  let buffer = "";
  let name = "message";
  let data: string[] = [];

  function line(value: string) {
    if (value === "") {
      if (data.length) onEvent(name, data.join("\n"));
      name = "message";
      data = [];
      return;
    }
    const separator = value.indexOf(":");
    const field = separator < 0 ? value : value.slice(0, separator);
    const content = separator < 0 ? "" : value.slice(separator + 1).replace(/^ /, "");
    if (field === "event") name = content || "message";
    if (field === "data") data.push(content);
  }

  return (chunk: string) => {
    buffer += chunk;
    while (buffer.length) {
      const end = buffer.search(/[\r\n]/);
      if (end < 0 || (buffer[end] === "\r" && end === buffer.length - 1)) break;
      const length = buffer[end] === "\r" && buffer[end + 1] === "\n" ? 2 : 1;
      line(buffer.slice(0, end));
      buffer = buffer.slice(end + length);
    }
  };
}

export async function readEventStream(
  body: ReadableStream<Uint8Array>,
  onEvent: (name: string, data: string) => void,
) {
  const reader = body.getReader();
  const decode = new TextDecoder();
  const parse = eventStreamParser(onEvent);
  try {
    for (;;) {
      const { value, done } = await reader.read();
      if (done) break;
      parse(decode.decode(value, { stream: true }));
    }
    parse(decode.decode());
  } finally {
    reader.releaseLock();
  }
}
