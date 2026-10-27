import { CheckCircleFilled, SearchOutlined } from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Button, Empty, Input, Modal, Spin } from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { mediaInfiniteQueryOptions } from "../queries";
import { formatFileSize, isImageMedia } from "./media-utils";

export type MediaItem = components["schemas"]["model.MediaResponse"];

export type MediaPickerProps = {
  open: boolean;
  onClose: () => void;
  onSelect: (media: MediaItem) => void;
  selectedId?: number;
  wid: number;
};

export function MediaPicker({
  open,
  onClose,
  onSelect,
  selectedId,
  wid,
}: MediaPickerProps) {
  const { t } = useTranslation();
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<MediaItem | null>(null);

  const filters = useMemo(() => {
    return {
      query: search.trim() || undefined,
    };
  }, [search]);

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    useInfiniteQuery(mediaInfiniteQueryOptions(wid, filters));

  const items = useMemo(() => {
    return data?.pages.flatMap((p) => p.items) ?? [];
  }, [data]);

  const activeId = selected?.id ?? selectedId;

  const handleConfirm = () => {
    if (selected) {
      onSelect(selected);
      onClose();
    }
  };

  return (
    <Modal
      title={t("cms.media.pickerTitle")}
      open={open}
      onCancel={onClose}
      destroyOnHidden
      width={720}
      footer={[
        <Button key="cancel" onClick={onClose}>
          {t("common.cancel")}
        </Button>,
        <Button
          key="select"
          type="primary"
          disabled={!selected && !selectedId}
          onClick={handleConfirm}
        >
          {t("cms.media.select")}
        </Button>,
      ]}
    >
      <div style={{ marginBottom: 16 }}>
        <Input
          prefix={<SearchOutlined />}
          placeholder={t("cms.media.searchPlaceholder")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          allowClear
        />
      </div>

      {isLoading ? (
        <div style={{ textAlign: "center", padding: 48 }}>
          <Spin />
        </div>
      ) : items.length === 0 ? (
        <Empty description={t("cms.media.emptyTitle")} />
      ) : (
        <div
          style={{
            maxHeight: 400,
            overflowY: "auto",
            display: "grid",
            gridTemplateColumns: "repeat(auto-fill, minmax(130px, 1fr))",
            gap: 12,
            padding: 4,
          }}
        >
          {items.map((item) => {
            const isSelected = item.id === activeId;
            const isImg = isImageMedia(item.mimeType, item.name);
            return (
              <button
                key={item.id}
                type="button"
                className="media-picker-item"
                onClick={() => setSelected(item)}
                style={{
                  border: isSelected
                    ? "2px solid var(--omed-primary)"
                    : "1px solid var(--omed-border)",
                  borderRadius: 6,
                  padding: 8,
                  cursor: "pointer",
                  position: "relative",
                  background: isSelected
                    ? "var(--omed-tint)"
                    : "var(--omed-surface)",
                  display: "flex",
                  flexDirection: "column",
                  alignItems: "center",
                  textAlign: "center",
                  width: "100%",
                }}
              >
                {isSelected && (
                  <CheckCircleFilled
                    style={{
                      position: "absolute",
                      top: 4,
                      right: 4,
                      color: "var(--omed-primary)",
                      fontSize: 16,
                      zIndex: 1,
                    }}
                  />
                )}
                <div
                  style={{
                    width: "100%",
                    height: 80,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    overflow: "hidden",
                    borderRadius: 4,
                    background: "var(--omed-surface-muted)",
                    marginBottom: 6,
                  }}
                >
                  {isImg && item.url ? (
                    <img
                      src={item.url}
                      alt={item.altText || item.title || item.name}
                      style={{
                        width: "100%",
                        height: "100%",
                        objectFit: "cover",
                      }}
                    />
                  ) : (
                    <span
                      style={{
                        fontSize: 11,
                        color: "var(--omed-text-secondary)",
                      }}
                    >
                      {item.mimeType ?? "file"}
                    </span>
                  )}
                </div>
                <div
                  style={{
                    fontSize: 12,
                    fontWeight: 500,
                    width: "100%",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                    color: "var(--omed-text)",
                  }}
                  title={item.title || item.name}
                >
                  {item.title || item.name}
                </div>
                <div
                  style={{
                    fontSize: 11,
                    color: "var(--omed-text-secondary)",
                  }}
                >
                  {formatFileSize(item.size ?? 0)}
                </div>
              </button>
            );
          })}
        </div>
      )}

      {hasNextPage && (
        <div style={{ textAlign: "center", marginTop: 12 }}>
          <Button
            size="small"
            loading={isFetchingNextPage}
            onClick={() => fetchNextPage()}
          >
            {t("cms.media.loadMore")}
          </Button>
        </div>
      )}
    </Modal>
  );
}
