import { useQueryClient } from "@tanstack/react-query";
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
  const queryClient = useQueryClient();
  const [pending, setPending] = useState(false);

  const run = useCallback(
    async (action: () => Promise<unknown>, successKey?: string) => {
      setPending(true);
      try {
        await action();
        if (successKey) message.success(t(successKey));
        await Promise.all([
          revalidator.revalidate(),
          queryClient.invalidateQueries(),
        ]);
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
    [message, revalidator, queryClient, t],
  );

  return { run, pending };
}
