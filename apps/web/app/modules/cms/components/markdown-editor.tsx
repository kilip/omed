import { Input, Segmented } from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";

export type MarkdownEditorProps = {
  value?: string;
  onChange?: (value: string) => void;
  placeholder?: string;
  rows?: number;
  disabled?: boolean;
};

export function MarkdownEditor({
  value = "",
  onChange,
  placeholder,
  rows = 12,
  disabled = false,
}: MarkdownEditorProps) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<"write" | "preview">("write");

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
        }}
      >
        <Segmented
          value={tab}
          onChange={(val) => setTab(val as "write" | "preview")}
          options={[
            {
              value: "write",
              label: t("cms.editor.write", "Write"),
            },
            {
              value: "preview",
              label: t("cms.editor.preview", "Preview"),
            },
          ]}
        />
      </div>

      {tab === "write" ? (
        <Input.TextArea
          value={value}
          onChange={(e) => onChange?.(e.target.value)}
          placeholder={
            placeholder ||
            t("cms.editor.placeholder", "Write markdown content here...")
          }
          rows={rows}
          disabled={disabled}
        />
      ) : (
        <div
          data-testid="markdown-preview"
          style={{
            minHeight: rows * 22,
            padding: "8px 12px",
            border: "1px solid var(--omed-border)",
            borderRadius: "var(--omed-radius)",
            background: "var(--omed-surface-muted)",
            overflowY: "auto",
            color: "var(--omed-text)",
          }}
        >
          {value.trim() ? (
            <ReactMarkdown>{value}</ReactMarkdown>
          ) : (
            <span style={{ color: "var(--omed-text-muted)" }}>
              {t("cms.editor.emptyPreview", "Nothing to preview.")}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
