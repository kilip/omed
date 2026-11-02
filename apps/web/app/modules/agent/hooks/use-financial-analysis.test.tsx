import type { UseQueryResult } from "@tanstack/react-query";
import { useQuery } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { FinancialStatus, FinancialSuggestion } from "../queries";
import { analyzeFinancialDocument } from "../queries";
import type { Job } from "./analysis-queue";
import { useAgentPermissions } from "./use-agent-permissions";
import { useFinancialAnalysis } from "./use-financial-analysis";

const { getCleanupEffect, setCleanupEffect } = vi.hoisted(() => {
  let cleanup: (() => void) | undefined;
  return {
    getCleanupEffect: () => cleanup,
    setCleanupEffect: (fn?: () => void) => {
      cleanup = fn;
    },
  };
});

vi.mock("react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react")>();
  return {
    ...actual,
    useEffect: vi.fn((effect: () => (() => void) | undefined) => {
      const cleanup = effect();
      if (typeof cleanup === "function") {
        setCleanupEffect(cleanup);
      }
    }),
  };
});

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQuery: vi.fn(),
  };
});

vi.mock("./use-agent-permissions", () => ({
  useAgentPermissions: vi.fn(),
}));

vi.mock("../queries", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../queries")>();
  return {
    ...actual,
    analyzeFinancialDocument: vi.fn(),
    financialStatusQueryOptions: vi.fn((wid: number) => ({
      queryKey: ["agent", wid, "financial-status"],
      queryFn: vi.fn(),
    })),
  };
});

function makeFile(name: string, size = 1024, type = "image/png"): File {
  return new File([new ArrayBuffer(size)], name, { type });
}

function mockPermissions(canUseFinancial: boolean) {
  return {
    canUseFinancial,
    canManage: false,
    canManageSettings: false,
    canRead: false,
    isAdmin: false,
    isOwner: false,
    isViewer: false,
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe("useFinancialAnalysis hook", () => {
  const wid = 42;

  beforeEach(() => {
    vi.clearAllMocks();
    setCleanupEffect(undefined);
  });

  function renderHookHarness(options: {
    flow?: "INCOME" | "EXPENSE";
    onResult?: (job: Job, s: FinancialSuggestion) => void;
    onError?: (job: Job, err: unknown) => void;
  }) {
    let currentHook!: ReturnType<typeof useFinancialAnalysis>;

    function Harness() {
      currentHook = useFinancialAnalysis({
        wid,
        flow: options.flow ?? "EXPENSE",
        onResult: options.onResult ?? vi.fn(),
        onError: options.onError,
      });
      return null;
    }

    renderToString(<Harness />);
    return {
      get hook() {
        return currentHook;
      },
      unmount() {
        getCleanupEffect()?.();
      },
    };
  }

  it("with no canUseFinancial, the status query is not enabled and enqueue never calls API", () => {
    vi.mocked(useAgentPermissions).mockReturnValue(mockPermissions(false));

    vi.mocked(useQuery).mockReturnValue({
      data: undefined,
    } as UseQueryResult<FinancialStatus>);

    const { hook } = renderHookHarness({});

    expect(useQuery).toHaveBeenCalledWith(
      expect.objectContaining({
        enabled: false,
      }),
    );
    expect(hook.available).toBe(false);

    // Enqueue should be a no-op
    const file = makeFile("invoice.png");
    hook.enqueue(file);

    expect(analyzeFinancialDocument).not.toHaveBeenCalled();
    expect(hook.jobs).toHaveLength(0);
  });

  it("with status.available=false or reason set, enqueue is a no-op", () => {
    vi.mocked(useAgentPermissions).mockReturnValue(mockPermissions(true));

    // available=false
    vi.mocked(useQuery).mockReturnValue({
      data: { available: false },
    } as UseQueryResult<FinancialStatus>);

    const harness1 = renderHookHarness({});
    expect(harness1.hook.available).toBe(false);

    harness1.hook.enqueue(makeFile("invoice.png"));
    expect(analyzeFinancialDocument).not.toHaveBeenCalled();

    // reason is set even if available=true
    vi.mocked(useQuery).mockReturnValue({
      data: { available: true, reason: "NO_PROVIDER" },
    } as UseQueryResult<FinancialStatus>);

    const harness2 = renderHookHarness({});
    expect(harness2.hook.available).toBe(false);

    harness2.hook.enqueue(makeFile("receipt.pdf", 100, "application/pdf"));
    expect(analyzeFinancialDocument).not.toHaveBeenCalled();
  });

  it("when permitted and available=true, enqueue queues analyzable files and calls API", async () => {
    vi.mocked(useAgentPermissions).mockReturnValue(mockPermissions(true));

    vi.mocked(useQuery).mockReturnValue({
      data: { available: true, enabled: true },
    } as UseQueryResult<FinancialStatus>);

    const d = deferred<FinancialSuggestion>();
    vi.mocked(analyzeFinancialDocument).mockReturnValue(d.promise);

    const onResult = vi.fn();
    const { hook } = renderHookHarness({ onResult });

    expect(hook.available).toBe(true);

    const file = makeFile("bill.png");
    hook.enqueue(file);

    expect(analyzeFinancialDocument).toHaveBeenCalledWith(
      wid,
      "EXPENSE",
      file,
      expect.any(AbortSignal),
    );

    const suggestion: FinancialSuggestion = { documentTotal: "125.00" };
    d.resolve(suggestion);

    await vi.waitFor(() => {
      expect(onResult).toHaveBeenCalledWith(
        expect.objectContaining({ file, status: "done" }),
        suggestion,
      );
    });
  });

  it("non-analyzable files (.csv, .xlsx) are never queued (Review Focus 2 & 3)", () => {
    vi.mocked(useAgentPermissions).mockReturnValue(mockPermissions(true));

    vi.mocked(useQuery).mockReturnValue({
      data: { available: true, enabled: true },
    } as UseQueryResult<FinancialStatus>);

    const { hook } = renderHookHarness({});

    const csvFile = makeFile("statement.csv", 50, "text/csv");
    const xlsxFile = makeFile(
      "records.xlsx",
      100,
      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    );

    hook.enqueue(csvFile);
    hook.enqueue(xlsxFile);

    expect(analyzeFinancialDocument).not.toHaveBeenCalled();
    expect(hook.jobs).toHaveLength(0);
  });

  it("unmount calls cancelAll and a later-settling result does not call onResult (Review Focus 2 & 3)", async () => {
    vi.mocked(useAgentPermissions).mockReturnValue(mockPermissions(true));

    vi.mocked(useQuery).mockReturnValue({
      data: { available: true, enabled: true },
    } as UseQueryResult<FinancialStatus>);

    const d = deferred<FinancialSuggestion>();
    let passedSignal: AbortSignal | undefined;
    vi.mocked(analyzeFinancialDocument).mockImplementation(
      (_wid, _flow, _file, signal) => {
        passedSignal = signal;
        return d.promise;
      },
    );

    const onResult = vi.fn();
    const { hook, unmount } = renderHookHarness({ onResult });

    const file = makeFile("receipt.pdf", 100, "application/pdf");
    hook.enqueue(file);

    expect(analyzeFinancialDocument).toHaveBeenCalledTimes(1);
    expect(passedSignal?.aborted).toBe(false);

    // Unmount the component
    unmount();

    expect(passedSignal?.aborted).toBe(true);

    // Later settling should be discarded
    d.resolve({ description: "Late settling after unmount" });
    await Promise.resolve();

    expect(onResult).not.toHaveBeenCalled();
  });

  it("cancel removes the file from the queue and reanalyze allows overwrite mode", async () => {
    vi.mocked(useAgentPermissions).mockReturnValue(mockPermissions(true));

    vi.mocked(useQuery).mockReturnValue({
      data: { available: true, enabled: true },
    } as UseQueryResult<FinancialStatus>);

    const d1 = deferred<FinancialSuggestion>();
    const d2 = deferred<FinancialSuggestion>();
    let callCount = 0;
    vi.mocked(analyzeFinancialDocument).mockImplementation(() => {
      callCount++;
      return callCount === 1 ? d1.promise : d2.promise;
    });

    const defaultOnResult = vi.fn();
    const customOnResult = vi.fn();
    const { hook } = renderHookHarness({ onResult: defaultOnResult });

    const file = makeFile("invoice.jpg", 300, "image/jpeg");
    expect(hook.statusOf?.(file)).toBeUndefined();

    hook.enqueue(file);
    expect(hook.statusOf?.(file)).toBe("analyzing");

    d1.resolve({ description: "Initial" });
    await vi.waitFor(() => {
      expect(defaultOnResult).toHaveBeenCalledTimes(1);
    });
    expect(hook.statusOf?.(file)).toBe("done");

    // Reanalyze with custom onResult callback
    hook.reanalyze(file, customOnResult);
    expect(analyzeFinancialDocument).toHaveBeenCalledTimes(2);

    d2.resolve({ description: "Overwritten" });
    await vi.waitFor(() => {
      expect(customOnResult).toHaveBeenCalledWith(
        expect.objectContaining({ file, status: "done" }),
        expect.objectContaining({ description: "Overwritten" }),
      );
    });
    expect(defaultOnResult).toHaveBeenCalledTimes(1); // not called again

    // Cancel removes it
    hook.cancel(file);
    expect(hook.statusOf?.(file)).toBeUndefined();
  });
});
