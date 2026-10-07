import { ToolOutlined } from "@ant-design/icons";
import { Button, Card, Space, Typography, theme } from "antd";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";

type UnderConstructionProps = {
  /** Judul halaman, tampil di atas card. Kosongkan kalau sudah ada heading lain. */
  pageTitle?: string;
  title?: string;
  description?: string;
  /** Tujuan tombol kembali. Set `null` untuk menyembunyikan tombol. */
  backTo?: string | null;
  backLabel?: string;
};

export function UnderConstruction({
  pageTitle,
  title,
  description,
  backTo = "/",
  backLabel,
}: UnderConstructionProps) {
  const { t } = useTranslation("underConstruction");
  const { token } = theme.useToken();

  const resolvedTitle = title ?? t("title");
  const resolvedDescription = description ?? t("description");
  const resolvedBackLabel = backLabel ?? t("backHome");

  return (
    <>
      {pageTitle && (
        <Typography.Title level={3} style={{ marginTop: 0 }}>
          {pageTitle}
        </Typography.Title>
      )}
      <Card styles={{ body: { padding: "64px 24px" } }}>
        <Space
          orientation="vertical"
          size={16}
          align="center"
          style={{ width: "100%", textAlign: "center" }}
        >
          <span
            aria-hidden
            style={{
              width: 72,
              height: 72,
              borderRadius: 20,
              display: "grid",
              placeItems: "center",
              fontSize: 32,
              color: token.colorPrimary,
              background: token.colorPrimaryBg,
            }}
          >
            <ToolOutlined />
          </span>

          <div>
            <Typography.Title level={4} style={{ margin: 0 }}>
              {resolvedTitle}
            </Typography.Title>
            <Typography.Paragraph
              type="secondary"
              style={{ margin: "8px auto 0", maxWidth: 360 }}
            >
              {resolvedDescription}
            </Typography.Paragraph>
          </div>

          {backTo !== null && (
            <Link to={backTo}>
              <Button type="primary">{resolvedBackLabel}</Button>
            </Link>
          )}
        </Space>
      </Card>
    </>
  );
}
