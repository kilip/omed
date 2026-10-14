import { App } from "antd";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useRevalidator } from "react-router";

/**
 * Runs a mutation, reports success or failure, then revalidates the route
 * loaders. Resolves to `true` when the mutation succeeded.
 */
export function useAction() {
  const { message } = App.useApp();
  const { t } = useTranslation();
  const revalidator = useRevalidator();
  const [pending, setPending] = useState(false);

  const run = useCallback(
    async (action: () => Promise<unknown>, successKey?: string) => {
      setPending(true);
      try {
        await action();
        if (successKey) message.success(t(successKey));
        await revalidator.revalidate();
        return true;
      } catch (error) {
        message.error(
          error instanceof Error ? error.message : t("common.error"),
        );
        return false;
      } finally {
        setPending(false);
      }
    },
    [message, revalidator, t],
  );

  return { run, pending };
}
