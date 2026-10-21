import { DeleteOutlined, EditOutlined, PlusOutlined } from "@ant-design/icons";
import type { components, paths } from "@omed/openapi";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import {
  App,
  Button,
  Empty,
  type MenuProps,
  Popconfirm,
  Space,
  Table,
  type TableProps,
  Tooltip,
} from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import type { SearchFacet } from "~/components/SearchBar";
import { api, unwrapVoid } from "~/lib/api";
import { formatDate } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { permissionsQueryOptions } from "~/modules/core/queries";
import { StatusTag } from "../components/status-tag";
import { useCmsPermissions } from "../hooks";
import { postKeys, postsInfiniteQueryOptions } from "../queries";
import type { Route } from "./+types/posts";

type PostItem = components["schemas"]["model.PostResponse"];
type SearchFilters = NonNullable<
  paths["/workspaces/{wid}/cms/posts/search"]["post"]["requestBody"]
>["content"]["application/json"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureInfiniteQueryData(postsInfiniteQueryOptions(wid, {})),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid };
}

export default function PostsRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { canCreatePost, canDeletePost } = useCmsPermissions(wid);

  const [queryText, setQueryText] = useState("");
  const [statusFilter, setStatusFilter] = useState<string | undefined>();

  const filters = useMemo<SearchFilters>(() => {
    return {
      query: queryText.trim() || undefined,
      status: statusFilter,
    };
  }, [queryText, statusFilter]);

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    useInfiniteQuery(postsInfiniteQueryOptions(wid, filters));

  const posts = useMemo(() => {
    return data?.pages.flatMap((page) => page.items) ?? [];
  }, [data]);

  const handleDelete = async (id?: number) => {
    if (!id) return;
    try {
      await unwrapVoid(
        api.DELETE("/workspaces/{wid}/cms/posts/{id}", {
          params: { path: { wid, id } },
        }),
      );
      message.success(t("common.deleted"));
      await queryClient.invalidateQueries({
        queryKey: postKeys.all(wid),
      });
    } catch {
      message.error(t("common.error"));
    }
  };

  const facets = useMemo<SearchFacet[]>(() => {
    const items: SearchFacet[] = [];
    if (statusFilter) {
      items.push({
        key: `status-${statusFilter}`,
        label: t("cms.posts.filterStatus", {
          status: t(`cms.status.${statusFilter}`, statusFilter),
        }),
        onRemove: () => setStatusFilter(undefined),
      });
    }
    return items;
  }, [statusFilter, t]);

  const filterMenuItems: MenuProps["items"] = [
    {
      key: "all",
      label: t("cms.posts.allStatuses", "All statuses"),
      onClick: () => setStatusFilter(undefined),
    },
    { type: "divider" },
    {
      key: "DRAFT",
      label: t("cms.status.DRAFT", "Draft"),
      onClick: () => setStatusFilter("DRAFT"),
    },
    {
      key: "SCHEDULED",
      label: t("cms.status.SCHEDULED", "Scheduled"),
      onClick: () => setStatusFilter("SCHEDULED"),
    },
    {
      key: "PUBLISHED",
      label: t("cms.status.PUBLISHED", "Published"),
      onClick: () => setStatusFilter("PUBLISHED"),
    },
  ];

  const columns: TableProps<PostItem>["columns"] = [
    {
      title: "",
      dataIndex: "coverImage",
      key: "cover",
      width: 60,
      render: (_, record) => {
        const url = record.coverImage?.url;
        if (!url) return null;
        return (
          <img
            src={url}
            alt=""
            style={{
              width: 40,
              height: 30,
              objectFit: "cover",
              borderRadius: 4,
            }}
          />
        );
      },
    },
    {
      title: t("cms.posts.titleField", "Title"),
      dataIndex: "title",
      key: "title",
      render: (text, record) => (
        <div>
          <Link
            to={`/cms/posts/${record.id}`}
            style={{ fontWeight: 600, color: "var(--omed-primary-text)" }}
          >
            {text}
          </Link>
          <div style={{ fontSize: 12, color: "var(--omed-text-secondary)" }}>
            /{record.slug}
          </div>
        </div>
      ),
    },
    {
      title: t("common.status", "Status"),
      dataIndex: "effectiveStatus",
      key: "status",
      width: 120,
      render: (_, record) => (
        <StatusTag
          status={record.status}
          effectiveStatus={record.effectiveStatus}
        />
      ),
    },
    {
      title: t("common.date", "Date"),
      dataIndex: "publishAt",
      key: "publishAt",
      width: 160,
      render: (val) => (val ? formatDate(val, i18n.language) : "-"),
    },
    {
      title: "",
      key: "actions",
      width: 90,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title={t("common.edit")}>
            <Button
              type="text"
              size="small"
              icon={<EditOutlined />}
              onClick={() => navigate(`/cms/posts/${record.id}`)}
            />
          </Tooltip>
          <Popconfirm
            title={t("cms.posts.deleteConfirm")}
            disabled={!canDeletePost}
            onConfirm={() => handleDelete(record.id)}
          >
            <Tooltip title={t("common.delete")}>
              <Button
                type="text"
                danger
                size="small"
                icon={<DeleteOutlined />}
                disabled={!canDeletePost}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Page
      title={t("cms.posts.title")}
      search={{
        value: queryText,
        onChange: setQueryText,
        placeholder: t("cms.posts.search"),
        facets,
        filterItems: filterMenuItems,
        selectedFilterKeys: statusFilter ? [statusFilter] : ["all"],
      }}
      actions={
        canCreatePost ? (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate("/cms/posts/new")}
          >
            {t("cms.posts.new")}
          </Button>
        ) : undefined
      }
    >
      <Panel>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={posts}
          loading={isLoading}
          pagination={false}
          locale={{
            emptyText: <Empty description={t("cms.posts.empty")} />,
          }}
        />

        {hasNextPage && (
          <div style={{ textAlign: "center", marginTop: 16 }}>
            <Button
              loading={isFetchingNextPage}
              onClick={() => fetchNextPage()}
            >
              {t("common.loadMore")}
            </Button>
          </div>
        )}
      </Panel>
    </Page>
  );
}
