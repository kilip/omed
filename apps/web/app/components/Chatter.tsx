import { Avatar, Button, Input, Segmented } from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";

export type ChatterKind = "message" | "note" | "tracking";

export type ChatterMessage = {
  id: number;
  author: string;
  /** Already formatted for display. */
  at: string;
  body: string;
  kind: ChatterKind;
};

type ChatterProps = {
  messages: ChatterMessage[];
  onPost: (kind: Exclude<ChatterKind, "tracking">, body: string) => void;
};

type Filter = "all" | "conversation" | "tracking";

/** Odoo-style chatter: composer, a filter, and the record's message history. */
export function Chatter({ messages, onPost }: ChatterProps) {
  const { t } = useTranslation();
  const [mode, setMode] = useState<Exclude<ChatterKind, "tracking">>("message");
  const [draft, setDraft] = useState("");
  const [filter, setFilter] = useState<Filter>("all");

  const visible = messages.filter((m) =>
    filter === "all"
      ? true
      : filter === "tracking"
        ? m.kind === "tracking"
        : m.kind !== "tracking",
  );

  const post = () => {
    const body = draft.trim();
    if (!body) return;
    onPost(mode, body);
    setDraft("");
  };

  return (
    <section
      aria-label={t("chatter.title")}
      className="surface flex min-w-0 flex-col gap-3 p-2.5"
      style={{ borderRadius: "var(--omed-radius-lg)" }}
    >
      <div className="flex flex-col gap-2">
        <Segmented<Exclude<ChatterKind, "tracking">>
          size="small"
          value={mode}
          onChange={setMode}
          options={[
            { value: "message", label: t("chatter.sendMessage") },
            { value: "note", label: t("chatter.logNote") },
          ]}
        />
        <Input.TextArea
          rows={2}
          value={draft}
          placeholder={t("chatter.placeholder")}
          aria-label={t("chatter.placeholder")}
          onChange={(e) => setDraft(e.target.value)}
        />
        <div className="flex justify-end">
          <Button type="primary" disabled={draft.trim() === ""} onClick={post}>
            {t(mode === "message" ? "chatter.send" : "chatter.log")}
          </Button>
        </div>
      </div>

      <Segmented<Filter>
        size="small"
        value={filter}
        onChange={setFilter}
        options={[
          { value: "all", label: t("chatter.all") },
          { value: "conversation", label: t("chatter.conversations") },
          { value: "tracking", label: t("chatter.changes") },
        ]}
      />

      {visible.length === 0 ? (
        <p className="m-0" style={{ color: "var(--omed-text-secondary)" }}>
          {t("chatter.empty")}
        </p>
      ) : (
        <ul className="m-0 flex list-none flex-col gap-3 p-0">
          {visible.map((m) => (
            <li key={m.id} className="flex gap-2">
              <Avatar size={24}>{m.author.slice(0, 1).toUpperCase()}</Avatar>
              <div className="min-w-0">
                <div className="flex flex-wrap items-baseline gap-x-2">
                  <span className="font-medium">{m.author}</span>
                  <span
                    className="text-xs"
                    style={{ color: "var(--omed-text-secondary)" }}
                  >
                    {m.at}
                  </span>
                </div>
                <div
                  className="whitespace-pre-wrap break-words"
                  style={
                    m.kind === "tracking"
                      ? { color: "var(--omed-text-secondary)" }
                      : m.kind === "note"
                        ? {
                            background: "var(--omed-surface-muted)",
                            borderRadius: "var(--omed-radius)",
                            padding: "4px 8px",
                          }
                        : undefined
                  }
                >
                  {m.body}
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
