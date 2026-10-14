import { Button, Drawer, Form, type FormInstance, Modal } from "antd";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

type RecordModalProps<T extends object> = {
  open: boolean;
  title: string;
  variant?: "modal" | "drawer";
  initialValues?: Partial<T>;
  submitting?: boolean;
  onClose: () => void;
  /** Resolve to `true` to close the dialog. */
  onSubmit: (values: T) => Promise<boolean>;
  children: (form: FormInstance<T>) => ReactNode;
};

/** A create/edit form in a modal or a drawer with consistent buttons. */
export function RecordModal<T extends object>({
  open,
  title,
  variant = "modal",
  initialValues,
  submitting,
  onClose,
  onSubmit,
  children,
}: RecordModalProps<T>) {
  const { t } = useTranslation();
  const [form] = Form.useForm<T>();

  const submit = async () => {
    const values = await form.validateFields().catch(() => undefined);
    if (!values) return;
    if (await onSubmit(values)) onClose();
  };

  const body = (
    <Form
      form={form}
      layout="vertical"
      initialValues={initialValues}
      onFinish={submit}
      requiredMark={false}
    >
      {children(form)}
    </Form>
  );

  if (variant === "drawer") {
    return (
      <Drawer
        open={open}
        title={title}
        size={420}
        onClose={onClose}
        destroyOnHidden
        extra={
          <Button type="primary" loading={submitting} onClick={submit}>
            {t("common.save")}
          </Button>
        }
      >
        {body}
      </Drawer>
    );
  }

  return (
    <Modal
      open={open}
      title={title}
      onCancel={onClose}
      onOk={submit}
      okText={t("common.save")}
      cancelText={t("common.cancel")}
      confirmLoading={submitting}
      destroyOnHidden
    >
      {body}
    </Modal>
  );
}
