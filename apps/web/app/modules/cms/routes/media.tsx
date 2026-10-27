import {
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  FilePdfOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import {
  App,
  Button,
  Card,
  Drawer,
  Empty,
  Form,
  Input,
  Popconfirm,
  Spin,
  Switch,
  Tag,
  Tooltip,
  Upload,
} from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { ApiError, api, unwrapOne, unwrapVoid } from "~/lib/api";
import { env } from "~/lib/env";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { permissionsQueryOptions } from "~/modules/core/queries";
import {
  formatFileSize,
  getPublicMediaUrl,
  isImageMedia,
  validateMediaFile,
} from "../components/media-utils";
import { useCmsPermissions } from "../hooks";
import { mediaInfiniteQueryOptions, mediaKeys } from "../queries";
import type { Route } from "./+types/media";

type MediaItem = components["schemas"]["model.MediaResponse"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureInfiniteQueryData(mediaInfiniteQueryOptions(wid, {})),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid };
}

export default function MediaRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { run, pending } = useAction();
  const { canUploadMedia, canEditMedia, canDeleteMedia } =
    useCmsPermissions(wid);

  const [search, setSearch] = useState("");
  const [editingMedia, setEditingMedia] = useState<MediaItem | null>(null);
  const [form] = Form.useForm();

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

  const validateAndNotify = (file: File): boolean => {
    const check = validateMediaFile(file);
    if (!check.valid) {
      if (check.reason === "too_large") {
        message.error(
          t("cms.media.validation.tooLarge", {
            maxSize: env.cmsMediaMaxSizeMB,
          }),
        );
      } else if (check.reason === "invalid_type") {
        message.error(
          t("cms.media.validation.invalidType", {
            types: env.cmsMediaAllowedTypes.join(", "),
          }),
        );
      } else {
        message.error(t("cms.media.validation.empty"));
      }
      return false;
    }
    return true;
  };

  const handleUpload = async (file: File) => {
    if (!validateAndNotify(file)) return false;

    const formData = new FormData();
    formData.append("file", file);
    formData.append("title", file.name);

    await run(async () => {
      await unwrapOne(
        api.POST("/workspaces/{wid}/cms/media", {
          params: { path: { wid } },
          body: formData as unknown as { file: string },
          bodySerializer: (body) => body as unknown as BodyInit,
        }),
      );
      await queryClient.invalidateQueries({
        queryKey: mediaKeys.all(wid),
      });
    }, "common.saved");

    return false;
  };

  const handleDelete = async (id?: number) => {
    if (!id) return;
    try {
      await unwrapVoid(
        api.DELETE("/workspaces/{wid}/cms/media/{id}", {
          params: { path: { wid, id } },
        }),
      );
      message.success(t("common.deleted"));
      await queryClient.invalidateQueries({
        queryKey: mediaKeys.all(wid),
      });
    } catch (err: unknown) {
      if (
        err instanceof ApiError &&
        (err.status === 409 || err.message.includes("MEDIA_IN_USE"))
      ) {
        message.error(t("cms.media.deleteInUseError"));
      } else {
        message.error(t("common.error"));
      }
    }
  };

  const handleCopyUrl = async (id?: number) => {
    if (!id) return;
    const url = getPublicMediaUrl(wid, id);
    try {
      await navigator.clipboard.writeText(url);
      message.success(t("cms.media.copiedPublicUrl"));
    } catch {
      message.error(t("common.error"));
    }
  };

  const handleOpenEdit = (media: MediaItem) => {
    setEditingMedia(media);
    form.setFieldsValue({
      title: media.title ?? "",
      altText: media.altText ?? "",
      caption: media.caption ?? "",
      isPublic: media.isPublic ?? true,
    });
  };

  const handleSaveEdit = async () => {
    const mediaId = editingMedia?.id;
    if (!mediaId) return;
    const values = await form.validateFields();
    await run(async () => {
      await unwrapOne(
        api.PUT("/workspaces/{wid}/cms/media/{id}", {
          params: { path: { wid, id: mediaId } },
          body: {
            title: values.title,
            altText: values.altText,
            caption: values.caption,
            isPublic: values.isPublic,
          },
        }),
      );
      await queryClient.invalidateQueries({
        queryKey: mediaKeys.all(wid),
      });
      setEditingMedia(null);
    }, "common.saved");
  };

  return (
    <Page
      title={t("cms.media.title")}
      search={{
        value: search,
        onChange: setSearch,
        placeholder: t("cms.media.searchPlaceholder"),
      }}
      actions={
        canUploadMedia ? (
          <Upload
            showUploadList={false}
            beforeUpload={handleUpload}
            accept={env.cmsMediaAllowedTypes.join(",")}
          >
            <Button type="primary" icon={<UploadOutlined />} loading={pending}>
              {t("cms.media.upload")}
            </Button>
          </Upload>
        ) : undefined
      }
    >
      <Panel>
        {isLoading ? (
          <div style={{ textAlign: "center", padding: 48 }}>
            <Spin />
          </div>
        ) : items.length === 0 ? (
          <Empty description={t("cms.media.emptyTitle")} />
        ) : (
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(220px, 1fr))",
              gap: 16,
            }}
          >
            {items.map((item) => {
              const isImg = isImageMedia(item.mimeType, item.name);
              return (
                <Card
                  key={item.id}
                  hoverable
                  style={{
                    display: "flex",
                    flexDirection: "column",
                    overflow: "hidden",
                  }}
                  styles={{
                    body: {
                      flex: 1,
                      display: "flex",
                      flexDirection: "column",
                      padding: 12,
                    },
                  }}
                  cover={
                    <div
                      style={{
                        height: 140,
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        background: "var(--omed-surface-muted)",
                        overflow: "hidden",
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
                        <FilePdfOutlined
                          style={{
                            fontSize: 48,
                            color: "var(--omed-text-secondary)",
                          }}
                        />
                      )}
                    </div>
                  }
                  actions={[
                    ...(item.isPublic
                      ? [
                          <Tooltip
                            key="copy"
                            title={t("cms.media.copyPublicUrl")}
                          >
                            <Button
                              type="text"
                              size="small"
                              icon={<CopyOutlined />}
                              onClick={() => handleCopyUrl(item.id)}
                            />
                          </Tooltip>,
                        ]
                      : []),
                    <Tooltip key="edit" title={t("cms.media.edit")}>
                      <Button
                        type="text"
                        size="small"
                        icon={<EditOutlined />}
                        disabled={!canEditMedia}
                        onClick={() => handleOpenEdit(item)}
                      />
                    </Tooltip>,
                    <Popconfirm
                      key="del"
                      title={t("cms.media.deleteConfirm")}
                      disabled={!canDeleteMedia}
                      onConfirm={() => handleDelete(item.id)}
                    >
                      <Tooltip title={t("cms.media.delete")}>
                        <Button
                          type="text"
                          danger
                          size="small"
                          icon={<DeleteOutlined />}
                          disabled={!canDeleteMedia}
                        />
                      </Tooltip>
                    </Popconfirm>,
                  ]}
                >
                  <div style={{ flex: 1 }}>
                    <div
                      style={{
                        fontWeight: 600,
                        fontSize: 14,
                        overflow: "hidden",
                        textOverflow: "ellipsis",
                        whiteSpace: "nowrap",
                        marginBottom: 4,
                        color: "var(--omed-text)",
                      }}
                      title={item.title || item.name}
                    >
                      {item.title || item.name}
                    </div>
                    <div
                      style={{
                        fontSize: 12,
                        color: "var(--omed-text-secondary)",
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                      }}
                    >
                      <span>{formatFileSize(item.size ?? 0)}</span>
                      {item.isPublic ? (
                        <Tag color="success">{t("cms.media.public")}</Tag>
                      ) : (
                        <Tag>{t("cms.media.private")}</Tag>
                      )}
                    </div>
                  </div>
                </Card>
              );
            })}
          </div>
        )}

        {hasNextPage && (
          <div style={{ textAlign: "center", marginTop: 24 }}>
            <Button
              loading={isFetchingNextPage}
              onClick={() => fetchNextPage()}
            >
              {t("cms.media.loadMore")}
            </Button>
          </div>
        )}
      </Panel>

      <Drawer
        title={t("cms.media.edit")}
        open={Boolean(editingMedia)}
        onClose={() => setEditingMedia(null)}
        destroyOnHidden
        extra={
          <Button
            type="primary"
            loading={pending}
            disabled={!canEditMedia}
            onClick={handleSaveEdit}
          >
            {t("common.save")}
          </Button>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="title"
            label={t("cms.media.mediaTitle")}
            rules={[{ required: true }]}
          >
            <Input disabled={!canEditMedia} />
          </Form.Item>
          <Form.Item name="altText" label={t("cms.media.altText")}>
            <Input disabled={!canEditMedia} />
          </Form.Item>
          <Form.Item name="caption" label={t("cms.media.caption")}>
            <Input.TextArea rows={3} disabled={!canEditMedia} />
          </Form.Item>
          <Form.Item
            name="isPublic"
            label={t("cms.media.isPublic")}
            valuePropName="checked"
          >
            <Switch disabled={!canEditMedia} />
          </Form.Item>
        </Form>
      </Drawer>
    </Page>
  );
}
