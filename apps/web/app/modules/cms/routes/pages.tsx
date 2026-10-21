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
import { pageKeys, pagesInfiniteQueryOptions } from "../queries";
import type { Route } from "./+types/pages";

type PageItem = components["schemas"]["model.PageResponse"];
type SearchFilters = NonNullable<
  paths["/workspaces/{wid}/cms/pages/search"]["post"]["requestBody"]
>["content"]["application/json"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureInfiniteQueryData(pagesInfiniteQueryOptions(wid, {})),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid };
}

export default function PagesRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const queryClient = useQueryClient();
  const { canCreatePage, canDeletePage } = useCmsPermissions(wid);

  const [queryText, setQueryText] = useState("");
  const [statusFilter, setStatusFilter] = useState<string | undefined>();

  const filters = useMemo<SearchFilters>(() => {
    return {
      query: queryText.trim() || undefined,
      status: statusFilter,
    };
  }, [queryText, statusFilter]);

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } =
    useInfiniteQuery(pagesInfiniteQueryOptions(wid, filters));

  const pages = useMemo(() => {
    return data?.pages.flatMap((page) => page.items) ?? [];
  }, [data]);

  const handleDelete = async (id?: number) => {
    if (!id) return;
    try {
      await unwrapVoid(
        api.DELETE("/workspaces/{wid}/cms/pages/{id}", {
          params: { path: { wid, id } },
        }),
      );
      message.success(t("common.deleted"));
      await queryClient.invalidateQueries({
        queryKey: pageKeys.all(wid),
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
        label: t("cms.pages.filterStatus", {
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
      label: t("cms.pages.allStatuses", "All statuses"),
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

  const columns: TableProps<PageItem>["columns"] = [
    {
      title: t("cms.pages.titleField", "Title"),
      dataIndex: "title",
      key: "title",
      render: (text, record) => (
        <div>
          <Link
            to={`/cms/pages/${record.id}`}
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
      width: 140,
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
      width: 180,
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
              onClick={() => navigate(`/cms/pages/${record.id}`)}
            />
          </Tooltip>
          <Popconfirm
            title={t("cms.pages.deleteConfirm")}
            disabled={!canDeletePage}
            onConfirm={() => handleDelete(record.id)}
          >
            <Tooltip title={t("common.delete")}>
              <Button
                type="text"
                danger
                size="small"
                icon={<DeleteOutlined />}
                disabled={!canDeletePage}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Page
      title={t("cms.pages.title")}
      search={{
        value: queryText,
        onChange: setQueryText,
        placeholder: t("cms.pages.search"),
        facets,
        filterItems: filterMenuItems,
        selectedFilterKeys: statusFilter ? [statusFilter] : ["all"],
      }}
      actions={
        canCreatePage ? (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate("/cms/pages/new")}
          >
            {t("cms.pages.new")}
          </Button>
        ) : undefined
      }
    >
      <Panel>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={pages}
          loading={isLoading}
          pagination={false}
          locale={{
            emptyText: <Empty description={t("cms.pages.empty")} />,
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
