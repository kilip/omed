import {
  CalendarOutlined,
  CheckCircleOutlined,
  StopOutlined,
} from "@ant-design/icons";
import {
  App,
  Button,
  DatePicker,
  Modal,
  Popconfirm,
  Space,
  Typography,
} from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { formatDate } from "~/lib/format";
import { type ContentStatus, StatusTag } from "./status-tag";

export function getEffectiveStatus(
  status?: string,
  publishAt?: string | null,
  now = new Date(),
): ContentStatus {
  if (status === "PUBLISHED") {
    if (publishAt && new Date(publishAt).getTime() > now.getTime()) {
      return "SCHEDULED";
    }
    return "PUBLISHED";
  }
  return "DRAFT";
}

export function getAvailableTransitions(effectiveStatus: string): {
  canPublish: boolean;
  canSchedule: boolean;
  canUnpublish: boolean;
} {
  const norm = effectiveStatus.toUpperCase();
  switch (norm) {
    case "PUBLISHED":
      return { canPublish: false, canSchedule: false, canUnpublish: true };
    case "SCHEDULED":
      return { canPublish: true, canSchedule: true, canUnpublish: true };
    default:
      return { canPublish: true, canSchedule: true, canUnpublish: false };
  }
}

export function isPastDate(
  date: dayjs.Dayjs | Date | string,
  now = new Date(),
): boolean {
  const d = dayjs(date);
  return d.valueOf() <= now.getTime();
}

export type PublishPanelProps = {
  status?: string;
  effectiveStatus?: string;
  publishAt?: string | null;
  onPublish?: () => Promise<void> | void;
  onSchedule?: (publishAt: string) => Promise<void> | void;
  onUnpublish?: () => Promise<void> | void;
  disabled?: boolean;
  loading?: boolean;
};

export function PublishPanel({
  status,
  effectiveStatus,
  publishAt,
  onPublish,
  onSchedule,
  onUnpublish,
  disabled = false,
  loading = false,
}: PublishPanelProps) {
  const { t, i18n } = useTranslation();
  const { message } = App.useApp();
  const [scheduleModalOpen, setScheduleModalOpen] = useState(false);
  const [selectedDate, setSelectedDate] = useState<Dayjs | null>(null);

  const effective = effectiveStatus || getEffectiveStatus(status, publishAt);
  const { canPublish, canSchedule, canUnpublish } =
    getAvailableTransitions(effective);

  const handleOpenSchedule = () => {
    // Default to 1 hour in the future
    setSelectedDate(dayjs().add(1, "hour").minute(0).second(0));
    setScheduleModalOpen(true);
  };

  const handleConfirmSchedule = async () => {
    if (!selectedDate) return;
    if (isPastDate(selectedDate)) {
      message.error(
        t("cms.publish.pastDateError", "Scheduled time must be in the future"),
      );
      return;
    }
    if (onSchedule) {
      await onSchedule(selectedDate.toISOString());
      setScheduleModalOpen(false);
    }
  };

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        gap: 12,
        padding: 16,
        background: "var(--omed-surface)",
        border: "1px solid var(--omed-border)",
        borderRadius: "var(--omed-radius)",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
        }}
      >
        <span style={{ fontWeight: 600, color: "var(--omed-text)" }}>
          {t("common.status", "Status")}
        </span>
        <StatusTag effectiveStatus={effective} />
      </div>

      {publishAt && (
        <div style={{ fontSize: 12, color: "var(--omed-text-secondary)" }}>
          {effective === "SCHEDULED"
            ? t("cms.publish.scheduledFor", "Scheduled for: {{date}}", {
                date: formatDate(publishAt, i18n.language),
              })
            : t("cms.publish.publishedAt", "Published at: {{date}}", {
                date: formatDate(publishAt, i18n.language),
              })}
        </div>
      )}

      <Space wrap>
        {canPublish && (
          <Button
            type="primary"
            icon={<CheckCircleOutlined />}
            disabled={disabled || loading}
            loading={loading}
            onClick={() => onPublish?.()}
          >
            {t("cms.publish.publishNow", "Publish now")}
          </Button>
        )}

        {canSchedule && (
          <Button
            icon={<CalendarOutlined />}
            disabled={disabled || loading}
            loading={loading}
            onClick={handleOpenSchedule}
          >
            {t("cms.publish.schedule", "Schedule...")}
          </Button>
        )}

        {canUnpublish && (
          <Popconfirm
            title={t(
              "cms.publish.unpublishConfirm",
              "Revert this content to draft?",
            )}
            disabled={disabled || loading}
            onConfirm={() => onUnpublish?.()}
          >
            <Button
              danger
              icon={<StopOutlined />}
              disabled={disabled || loading}
              loading={loading}
            >
              {t("cms.publish.unpublish", "Unpublish")}
            </Button>
          </Popconfirm>
        )}
      </Space>

      <Modal
        title={t("cms.publish.scheduleTitle", "Schedule publication")}
        open={scheduleModalOpen}
        onCancel={() => setScheduleModalOpen(false)}
        onOk={handleConfirmSchedule}
        okText={t("cms.publish.scheduleConfirm", "Schedule")}
        cancelText={t("common.cancel")}
        okButtonProps={{
          disabled: !selectedDate || isPastDate(selectedDate),
        }}
      >
        <div style={{ margin: "20px 0" }}>
          <Typography.Paragraph type="secondary">
            {t(
              "cms.publish.scheduleDesc",
              "Select a future date and time for this content to be published.",
            )}
          </Typography.Paragraph>
          <DatePicker
            showTime
            style={{ width: "100%" }}
            value={selectedDate}
            onChange={(d) => setSelectedDate(d)}
            disabledDate={(current) =>
              Boolean(current?.endOf("day").isBefore(dayjs()))
            }
          />
        </div>
      </Modal>
    </div>
  );
}
