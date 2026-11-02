import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useReducer, useRef } from "react";
import type { FinancialSuggestion } from "../queries";
import {
  analyzeFinancialDocument,
  financialStatusQueryOptions,
} from "../queries";
import {
  type AnalysisQueue,
  createAnalysisQueue,
  isAnalyzableFile,
  type Job,
  type JobStatus,
} from "./analysis-queue";
import { useAgentPermissions } from "./use-agent-permissions";

export interface UseFinancialAnalysisArgs {
  wid: number;
  flow: "INCOME" | "EXPENSE";
  onResult: (job: Job, suggestion: FinancialSuggestion) => void;
  onError?: (job: Job, error: unknown) => void;
}

export interface UseFinancialAnalysisReturn {
  available: boolean;
  jobs: Job[];
  enqueue: (file: File) => void;
  reanalyze: (
    file: File,
    onResult: (job: Job, suggestion: FinancialSuggestion) => void,
  ) => void;
  cancel: (file: File) => void;
  cancelAll: () => void;
  statusOf: (file: File) => JobStatus | undefined;
}

export function useFinancialAnalysis({
  wid,
  flow,
  onResult,
  onError,
}: UseFinancialAnalysisArgs): UseFinancialAnalysisReturn {
  const { canUseFinancial } = useAgentPermissions(wid);

  const statusQuery = useQuery({
    ...financialStatusQueryOptions(wid),
    enabled: Boolean(wid && canUseFinancial),
  });

  const available = Boolean(
    canUseFinancial &&
      statusQuery.data?.available === true &&
      !statusQuery.data?.reason,
  );

  const [, forceUpdate] = useReducer((x: number) => x + 1, 0);
  const isMountedRef = useRef(true);

  const argsRef = useRef({ wid, flow, onResult, onError });
  argsRef.current = { wid, flow, onResult, onError };

  const queueRef = useRef<AnalysisQueue | null>(null);
  if (!queueRef.current) {
    queueRef.current = createAnalysisQueue({
      run: (file, signal) =>
        analyzeFinancialDocument(
          argsRef.current.wid,
          argsRef.current.flow,
          file,
          signal,
        ),
      onResult: (job, suggestion) => argsRef.current.onResult(job, suggestion),
      onError: (job, error) => argsRef.current.onError?.(job, error),
      onChange: () => {
        if (isMountedRef.current) {
          forceUpdate();
        }
      },
    });
  }

  const queue = queueRef.current;

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
      queue.cancelAll();
    };
  }, [queue]);

  const enqueue = useCallback(
    (file: File) => {
      if (!available || !isAnalyzableFile(file)) {
        return;
      }
      queue.add(file);
    },
    [available, queue],
  );

  const reanalyze = useCallback(
    (
      file: File,
      onResultOverride: (job: Job, suggestion: FinancialSuggestion) => void,
    ) => {
      if (!available || !isAnalyzableFile(file)) {
        return;
      }
      queue.reanalyze(file, onResultOverride);
    },
    [available, queue],
  );

  const cancel = useCallback(
    (file: File) => {
      const match = queue
        .jobs()
        .find((j) => j.file.name === file.name && j.file.size === file.size);
      if (match) {
        queue.cancel(match.id);
      }
    },
    [queue],
  );

  const cancelAll = useCallback(() => {
    queue.cancelAll();
  }, [queue]);

  const statusOf = useCallback(
    (file: File) => {
      return queue.statusOf(file);
    },
    [queue],
  );

  return {
    available,
    jobs: queue.jobs(),
    enqueue,
    reanalyze,
    cancel,
    cancelAll,
    statusOf,
  };
}
