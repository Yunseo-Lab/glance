import { useState, type ReactNode } from "react";
import { safeExternalURL } from "../links";
// Deliberately small syntax: no HTML, images, scripts, URL fetching or plugins.
export function markdownInline(text: string): ReactNode[] {
  const output: ReactNode[] = [];
  let index = 0,
    plainStart = 0,
    tokens = 0;
  while (index < text.length) {
    let end = -1,
      length = 1,
      kind = "";
    if (text.startsWith("**", index)) {
      kind = "strong";
      length = 2;
      end = text.indexOf("**", index + 2);
    } else if (text[index] === "`") {
      kind = "code";
      end = text.indexOf("`", index + 1);
    } else if (text[index] === "*") {
      kind = "em";
      end = text.indexOf("*", index + 1);
    } else if (text[index] === "[") {
      const split = text.indexOf("](", index + 1);
      if (split < 0) break;
      end = text.indexOf(")", split + 2);
      if (end < 0) break;
      if (index > plainStart) output.push(text.slice(plainStart, index));
      const address = safeExternalURL(text.slice(split + 2, end));
      output.push(
        address ? (
          <a
            key={index}
            href={address}
            target="_blank"
            rel="noopener noreferrer"
          >
            {text.slice(index + 1, split)}
          </a>
        ) : (
          text.slice(index, end + 1)
        ),
      );
      index = end + 1;
      plainStart = index;
      if (++tokens >= 256) break;
      continue;
    } else {
      index++;
      continue;
    }
    if (end < 0) break;
    if (index > plainStart) output.push(text.slice(plainStart, index));
    const value = text.slice(index + length, end);
    output.push(
      kind === "strong" ? (
        <strong key={index}>{value}</strong>
      ) : kind === "code" ? (
        <code key={index}>{value}</code>
      ) : (
        <em key={index}>{value}</em>
      ),
    );
    index = end + length;
    plainStart = index;
    if (++tokens >= 256) break;
  }
  if (plainStart < text.length) output.push(text.slice(plainStart));
  return output;
}
export default function SafeMarkdown({ body }: { body: string }) {
  const [expanded, setExpanded] = useState(false);
  const all = body.split("\n"),
    lines = all.slice(0, 200);
  const blocks: ReactNode[] = [];
  for (let index = 0; index < lines.length; index++) {
    const line = lines[index];
    if (line.startsWith("```")) {
      const code: string[] = [];
      const start = index;
      while (++index < lines.length && !lines[index].startsWith("```"))
        code.push(lines[index]);
      blocks.push(
        <pre key={start}>
          <code>{code.join("\n")}</code>
        </pre>,
      );
      continue;
    }
    const heading = /^(#{1,3})\s+(.+)$/.exec(line);
    if (heading) {
      const content = markdownInline(heading[2]);
      blocks.push(
        heading[1].length === 1 ? (
          <h3 key={index}>{content}</h3>
        ) : heading[1].length === 2 ? (
          <h4 key={index}>{content}</h4>
        ) : (
          <h5 key={index}>{content}</h5>
        ),
      );
      continue;
    }
    if (/^[-*] /.test(line)) {
      const items: ReactNode[] = [];
      const start = index;
      do {
        const text = lines[index].slice(2),
          task = /^\[([ xX])\]\s+(.+)$/.exec(text);
        items.push(
          <li key={index}>
            {task ? (
              <>
                <input
                  type="checkbox"
                  disabled
                  checked={task[1].toLowerCase() === "x"}
                  aria-label={task[2]}
                />
                {markdownInline(task[2])}
              </>
            ) : (
              markdownInline(text)
            )}
          </li>,
        );
        index++;
      } while (index < lines.length && /^[-*] /.test(lines[index]));
      index--;
      blocks.push(<ul key={start}>{items}</ul>);
      continue;
    }
    if (line.startsWith("> ")) {
      blocks.push(
        <blockquote key={index}>{markdownInline(line.slice(2))}</blockquote>,
      );
      continue;
    }
    if (line.trim()) blocks.push(<p key={index}>{markdownInline(line)}</p>);
  }
  return (
    <div className="safe-markdown">
      {expanded ? <pre className="note-full-text">{body}</pre> : blocks}
      {all.length > 200 && (
        <button
          className="text-button"
          onClick={() => setExpanded((value) => !value)}
        >
          {expanded ? "접어서 보기" : "전체 내용 보기"}
        </button>
      )}
    </div>
  );
}
