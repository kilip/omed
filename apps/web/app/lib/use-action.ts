import { useQueryClient } from "@tanstack/react-query";
import { App } from "antd";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { useRevalidator } from "react-router";
import i18n from "~/i18n";
import { ApiError } from "./api";

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
        let msg = t("common.error");
        if (error instanceof ApiError && error.code) {
          const translationKey = `fin.errors.${error.code}`;
          msg = i18n.exists(translationKey) ? t(translationKey) : error.message;
        } else if (error instanceof ApiError && error.status === 403) {
          msg = error.message || t("common.forbidden");
        } else if (error instanceof Error) {
          msg = error.message;
        }
        message.error(msg);
        return false;
      } finally {
        setPending(false);
      }
    },
    [message, revalidator, queryClient, t],
  );

  return { run, pending };
}
