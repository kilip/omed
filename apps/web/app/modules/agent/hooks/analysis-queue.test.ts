import { describe, expect, it, vi } from "vitest";
import type { FinancialSuggestion } from "../queries";
import {
  type AnalysisQueue,
  createAnalysisQueue,
  isAnalyzableFile,
} from "./analysis-queue";

function makeFile(name: string, size = 1024, type = "image/png"): File {
  return new File([new ArrayBuffer(size)], name, { type });
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

describe("AnalysisQueue", () => {
  it("runs jobs one at a time in add order, starting the second only after the first settles", async () => {
    const d1 = deferred<FinancialSuggestion>();
    const d2 = deferred<FinancialSuggestion>();

    const run = vi.fn((file: File, _signal: AbortSignal) => {
      if (file.name === "file1.png") return d1.promise;
      if (file.name === "file2.png") return d2.promise;
      return Promise.reject(new Error("unknown file"));
    });

    const onResult = vi.fn();
    const onChange = vi.fn();

    const queue: AnalysisQueue = createAnalysisQueue({
      run,
      onResult,
      onChange,
    });

    const file1 = makeFile("file1.png", 100);
    const file2 = makeFile("file2.png", 200);

    const id1 = queue.add(file1);
    const id2 = queue.add(file2);

    expect(id1).toBeDefined();
    expect(id2).toBeDefined();
    expect(id1).not.toBe(id2);

    // Concurrency is 1: only file1 has started
    expect(run).toHaveBeenCalledTimes(1);
    expect(run).toHaveBeenCalledWith(file1, expect.any(AbortSignal));

    const jobsAfterAdd = queue.jobs();
    expect(jobsAfterAdd).toHaveLength(2);
    expect(jobsAfterAdd[0]?.status).toBe("analyzing");
    expect(jobsAfterAdd[1]?.status).toBe("queued");

    // Settle first job
    const suggestion1: FinancialSuggestion = {
      description: "First invoice",
      documentTotal: "100.00",
    };
    d1.resolve(suggestion1);

    await vi.waitFor(() => {
      expect(onResult).toHaveBeenCalledWith(
        expect.objectContaining({ id: id1, status: "done" }),
        suggestion1,
      );
    });

    // Second job starts only after the first settles
    await vi.waitFor(() => {
      expect(run).toHaveBeenCalledTimes(2);
      expect(run).toHaveBeenLastCalledWith(file2, expect.any(AbortSignal));
    });

    const jobsMid = queue.jobs();
    expect(jobsMid[0]?.status).toBe("done");
    expect(jobsMid[1]?.status).toBe("analyzing");

    // Settle second job
    const suggestion2: FinancialSuggestion = {
      description: "Second receipt",
      documentTotal: "50.00",
    };
    d2.resolve(suggestion2);

    await vi.waitFor(() => {
      expect(onResult).toHaveBeenCalledWith(
        expect.objectContaining({ id: id2, status: "done" }),
        suggestion2,
      );
    });

    const finalJobs = queue.jobs();
    expect(finalJobs[0]?.status).toBe("done");
    expect(finalJobs[1]?.status).toBe("done");
  });

  it("cancel on the running job aborts its signal, never calls onResult for it, and the queue moves on", async () => {
    const d1 = deferred<FinancialSuggestion>();
    const d2 = deferred<FinancialSuggestion>();

    let signal1: AbortSignal | undefined;
    const run = vi.fn((file: File, signal: AbortSignal) => {
      if (file.name === "file1.png") {
        signal1 = signal;
        return d1.promise;
      }
      return d2.promise;
    });

    const onResult = vi.fn();
    const onChange = vi.fn();

    const queue = createAnalysisQueue({ run, onResult, onChange });

    const file1 = makeFile("file1.png", 100);
    const file2 = makeFile("file2.png", 200);

    const id1 = queue.add(file1);
    queue.add(file2);

    expect(run).toHaveBeenCalledTimes(1);
    expect(signal1?.aborted).toBe(false);

    // Cancel running job
    queue.cancel(id1);

    expect(signal1?.aborted).toBe(true);

    // Later settling of d1 should be discarded (Review Focus 1)
    d1.resolve({ description: "Should be discarded" });
    await Promise.resolve();

    expect(onResult).not.toHaveBeenCalledWith(
      expect.objectContaining({ id: id1 }),
      expect.anything(),
    );

    // Queue moves on to file2
    await vi.waitFor(() => {
      expect(run).toHaveBeenCalledTimes(2);
      expect(run).toHaveBeenLastCalledWith(file2, expect.any(AbortSignal));
    });

    d2.resolve({ description: "Second job ok" });
    await vi.waitFor(() => {
      expect(onResult).toHaveBeenCalledTimes(1);
    });
  });

  it("cancel on a queued job drops it so it never runs", async () => {
    const d1 = deferred<FinancialSuggestion>();
    const d2 = deferred<FinancialSuggestion>();

    const run = vi.fn((file: File) => {
      if (file.name === "file1.png") return d1.promise;
      return d2.promise;
    });

    const onResult = vi.fn();
    const onChange = vi.fn();

    const queue = createAnalysisQueue({ run, onResult, onChange });

    const file1 = makeFile("file1.png", 100);
    const file2 = makeFile("file2.png", 200);

    queue.add(file1);
    const id2 = queue.add(file2);

    // Cancel queued job before it starts
    queue.cancel(id2);

    expect(queue.jobs().some((j) => j.id === id2)).toBe(false);

    // Settle file1
    d1.resolve({ description: "Done 1" });
    await vi.waitFor(() => {
      expect(onResult).toHaveBeenCalledTimes(1);
    });

    // File2 should never run
    expect(run).toHaveBeenCalledTimes(1);
  });

  it("a result that settles after cancelAll is discarded (Review Focus 1)", async () => {
    const d1 = deferred<FinancialSuggestion>();
    const d2 = deferred<FinancialSuggestion>();

    let signal1: AbortSignal | undefined;
    const run = vi.fn((file: File, signal: AbortSignal) => {
      if (file.name === "file1.png") {
        signal1 = signal;
        return d1.promise;
      }
      return d2.promise;
    });

    const onResult = vi.fn();
    const onChange = vi.fn();

    const queue = createAnalysisQueue({ run, onResult, onChange });

    const file1 = makeFile("file1.png", 100);
    const file2 = makeFile("file2.png", 200);

    queue.add(file1);
    queue.add(file2);

    queue.cancelAll();

    expect(signal1?.aborted).toBe(true);
    expect(queue.jobs()).toHaveLength(0);

    // Settle both after cancelAll
    d1.resolve({ description: "Ignored 1" });
    d2.resolve({ description: "Ignored 2" });
    await Promise.resolve();

    expect(onResult).not.toHaveBeenCalled();
    expect(run).toHaveBeenCalledTimes(1); // second never started
  });

  it("a rejection with errorCode: 'RATE_LIMITED' marks that job failed, marks remaining queued jobs paused, and stops the queue; retry resumes processing", async () => {
    const d1 = deferred<FinancialSuggestion>();
    const d2 = deferred<FinancialSuggestion>();

    const run = vi.fn((file: File) => {
      if (file.name === "file1.png") return d1.promise;
      if (file.name === "file2.png") return d2.promise;
      return Promise.reject(new Error("unknown file"));
    });

    const onResult = vi.fn();
    const onChange = vi.fn();
    const onError = vi.fn();

    const queue = createAnalysisQueue({
      run,
      onResult,
      onChange,
      onError,
    });

    const file1 = makeFile("file1.png", 100);
    const file2 = makeFile("file2.png", 200);
    const file3 = makeFile("file3.png", 300);

    const id1 = queue.add(file1);
    const id2 = queue.add(file2);
    const id3 = queue.add(file3);

    // Reject file1 with RATE_LIMITED
    const rateLimitError = {
      errorCode: "RATE_LIMITED",
      message: "Too many requests",
    };
    d1.reject(rateLimitError);

    await vi.waitFor(() => {
      expect(onError).toHaveBeenCalledWith(
        expect.objectContaining({
          id: id1,
          status: "failed",
          errorCode: "RATE_LIMITED",
        }),
        rateLimitError,
      );
    });

    const jobsAfterRateLimit = queue.jobs();
    expect(jobsAfterRateLimit.find((j) => j.id === id1)?.status).toBe("failed");
    expect(jobsAfterRateLimit.find((j) => j.id === id1)?.errorCode).toBe(
      "RATE_LIMITED",
    );
    expect(jobsAfterRateLimit.find((j) => j.id === id2)?.status).toBe("paused");
    expect(jobsAfterRateLimit.find((j) => j.id === id3)?.status).toBe("paused");

    // Queue stopped: run is NOT called for file2
    expect(run).toHaveBeenCalledTimes(1);

    // Calling retry on a paused job resumes processing
    queue.retry(id2);

    await vi.waitFor(() => {
      expect(run).toHaveBeenCalledTimes(2);
      expect(run).toHaveBeenLastCalledWith(file2, expect.any(AbortSignal));
    });

    const jobsAfterResume = queue.jobs();
    expect(jobsAfterResume.find((j) => j.id === id2)?.status).toBe("analyzing");

    // Settle file2
    d2.resolve({ description: "Success 2" });
    await vi.waitFor(() => {
      expect(onResult).toHaveBeenCalledWith(
        expect.objectContaining({ id: id2, status: "done" }),
        expect.anything(),
      );
    });
  });

  it("any other rejection marks the job failed and the queue continues with the next job", async () => {
    const d1 = deferred<FinancialSuggestion>();
    const d2 = deferred<FinancialSuggestion>();

    const run = vi.fn((file: File) => {
      if (file.name === "file1.png") return d1.promise;
      return d2.promise;
    });

    const onResult = vi.fn();
    const onChange = vi.fn();
    const onError = vi.fn();

    const queue = createAnalysisQueue({
      run,
      onResult,
      onChange,
      onError,
    });

    const file1 = makeFile("file1.png", 100);
    const file2 = makeFile("file2.png", 200);

    const id1 = queue.add(file1);
    const id2 = queue.add(file2);

    const otherError = {
      code: "UNSUPPORTED_DOCUMENT",
      message: "Not supported",
    };
    d1.reject(otherError);

    await vi.waitFor(() => {
      expect(onError).toHaveBeenCalledWith(
        expect.objectContaining({
          id: id1,
          status: "failed",
          errorCode: "UNSUPPORTED_DOCUMENT",
        }),
        otherError,
      );
    });

    // Queue moves on to file2
    await vi.waitFor(() => {
      expect(run).toHaveBeenCalledTimes(2);
      expect(run).toHaveBeenLastCalledWith(file2, expect.any(AbortSignal));
    });

    d2.resolve({ description: "Success 2" });
    await vi.waitFor(() => {
      expect(onResult).toHaveBeenCalledWith(
        expect.objectContaining({ id: id2, status: "done" }),
        expect.anything(),
      );
    });
  });

  it("adding the same file (name and size) twice returns the same id and runs once", () => {
    const run = vi.fn(() => new Promise<FinancialSuggestion>(() => {}));
    const queue = createAnalysisQueue({
      run,
      onResult: vi.fn(),
      onChange: vi.fn(),
    });

    const fileA = makeFile("invoice.png", 2048);
    const fileB = makeFile("invoice.png", 2048);

    const idA = queue.add(fileA);
    const idB = queue.add(fileB);

    expect(idB).toBe(idA);
    expect(queue.jobs()).toHaveLength(1);
    expect(run).toHaveBeenCalledTimes(1);
  });

  it("reanalyze re-runs analysis for a file even if done/failed and invokes the custom onResult", async () => {
    const d1 = deferred<FinancialSuggestion>();
    const d2 = deferred<FinancialSuggestion>();

    let runCount = 0;
    const run = vi.fn((_file: File) => {
      runCount++;
      return runCount === 1 ? d1.promise : d2.promise;
    });

    const defaultOnResult = vi.fn();
    const customOnResult = vi.fn();
    const queue = createAnalysisQueue({
      run,
      onResult: defaultOnResult,
      onChange: vi.fn(),
    });

    const file = makeFile("receipt.pdf", 1000, "application/pdf");
    const id = queue.add(file);

    d1.resolve({ description: "Initial run" });
    await vi.waitFor(() => {
      expect(defaultOnResult).toHaveBeenCalledTimes(1);
    });

    expect(queue.jobs()[0]?.status).toBe("done");

    // Reanalyze with custom onResult (overwrite mode)
    queue.reanalyze(file, customOnResult);

    await vi.waitFor(() => {
      expect(run).toHaveBeenCalledTimes(2);
    });

    expect(queue.jobs()[0]?.status).toBe("analyzing");

    const reanalyzeSuggestion: FinancialSuggestion = {
      description: "Overwritten suggestion",
    };
    d2.resolve(reanalyzeSuggestion);

    await vi.waitFor(() => {
      expect(customOnResult).toHaveBeenCalledWith(
        expect.objectContaining({ id, status: "done" }),
        reanalyzeSuggestion,
      );
    });

    // Default onResult should NOT have been called for reanalyze
    expect(defaultOnResult).toHaveBeenCalledTimes(1);
  });

  it("statusOf returns the status of a file or undefined", () => {
    const run = vi.fn(() => new Promise<FinancialSuggestion>(() => {}));
    const queue = createAnalysisQueue({
      run,
      onResult: vi.fn(),
      onChange: vi.fn(),
    });

    const file1 = makeFile("doc.pdf", 500, "application/pdf");
    const file2 = makeFile("other.pdf", 600, "application/pdf");

    expect(queue.statusOf(file1)).toBeUndefined();

    queue.add(file1);
    expect(queue.statusOf(file1)).toBe("analyzing");
    expect(queue.statusOf(file2)).toBeUndefined();
  });
});

describe("isAnalyzableFile", () => {
  it("returns true for png, jpg, jpeg, webp, and pdf by extension or MIME type", () => {
    expect(isAnalyzableFile(makeFile("photo.png", 10, "image/png"))).toBe(true);
    expect(isAnalyzableFile(makeFile("photo.jpg", 10, "image/jpeg"))).toBe(
      true,
    );
    expect(isAnalyzableFile(makeFile("photo.jpeg", 10, "image/jpeg"))).toBe(
      true,
    );
    expect(isAnalyzableFile(makeFile("photo.webp", 10, "image/webp"))).toBe(
      true,
    );
    expect(isAnalyzableFile(makeFile("doc.pdf", 10, "application/pdf"))).toBe(
      true,
    );

    // Case-insensitive extensions
    expect(isAnalyzableFile(makeFile("receipt.PNG", 10, ""))).toBe(true);
    expect(isAnalyzableFile(makeFile("receipt.PDF", 10, ""))).toBe(true);
    expect(isAnalyzableFile(makeFile("receipt.JPEG", 10, ""))).toBe(true);
  });

  it("returns false for csv, xls, xlsx, and unknown files", () => {
    expect(isAnalyzableFile(makeFile("data.csv", 10, "text/csv"))).toBe(false);
    expect(
      isAnalyzableFile(makeFile("data.xls", 10, "application/vnd.ms-excel")),
    ).toBe(false);
    expect(
      isAnalyzableFile(
        makeFile(
          "data.xlsx",
          10,
          "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        ),
      ),
    ).toBe(false);
    expect(isAnalyzableFile(makeFile("notes.txt", 10, "text/plain"))).toBe(
      false,
    );
    expect(isAnalyzableFile(makeFile("unknown", 10, ""))).toBe(false);
  });
});
