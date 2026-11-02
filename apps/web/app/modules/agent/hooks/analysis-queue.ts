import type { FinancialSuggestion } from "../queries";

export type JobStatus = "queued" | "analyzing" | "done" | "failed" | "paused";

export interface Job {
  id: string;
  file: File;
  status: JobStatus;
  errorCode?: string;
}

export interface AnalysisQueue {
  add(file: File): string;
  cancel(id: string): void;
  retry(id: string): void;
  reanalyze(
    file: File,
    onResult?: (job: Job, suggestion: FinancialSuggestion) => void,
  ): string;
  cancelAll(): void;
  jobs(): Job[];
  statusOf(file: File): JobStatus | undefined;
}

export interface CreateAnalysisQueueArgs {
  run: (file: File, signal: AbortSignal) => Promise<FinancialSuggestion>;
  onResult: (job: Job, suggestion: FinancialSuggestion) => void;
  onChange: () => void;
  onError?: (job: Job, error: unknown) => void;
}

interface InternalJob extends Job {
  controller?: AbortController;
  onResult?: (job: Job, suggestion: FinancialSuggestion) => void;
}

const ANALYZABLE_EXTENSIONS = new Set([
  ".png",
  ".jpg",
  ".jpeg",
  ".webp",
  ".pdf",
]);

const ANALYZABLE_MIME_TYPES = new Set([
  "image/png",
  "image/jpeg",
  "image/webp",
  "application/pdf",
]);

export function isAnalyzableFile(file: File): boolean {
  if (file.type && ANALYZABLE_MIME_TYPES.has(file.type.toLowerCase())) {
    return true;
  }
  const name = file.name || "";
  const lastDot = name.lastIndexOf(".");
  if (lastDot === -1) return false;
  const ext = name.slice(lastDot).toLowerCase();
  return ANALYZABLE_EXTENSIONS.has(ext);
}

function isRateLimitedError(error: unknown): boolean {
  if (!error || typeof error !== "object") return false;
  const e = error as Record<string, unknown>;
  return (
    e.errorCode === "RATE_LIMITED" ||
    e.code === "RATE_LIMITED" ||
    e.status === 429
  );
}

function extractErrorCode(error: unknown): string | undefined {
  if (!error || typeof error !== "object") return undefined;
  const e = error as Record<string, unknown>;
  if (typeof e.errorCode === "string") return e.errorCode;
  if (typeof e.code === "string") return e.code;
  if (typeof e.status === "number") return String(e.status);
  return undefined;
}

export function createAnalysisQueue({
  run,
  onResult,
  onChange,
  onError,
}: CreateAnalysisQueueArgs): AnalysisQueue {
  let jobs: InternalJob[] = [];
  let activeJobId: string | null = null;
  let isPaused = false;
  let idCounter = 0;

  function findJobByFile(file: File): InternalJob | undefined {
    return jobs.find(
      (j) => j.file.name === file.name && j.file.size === file.size,
    );
  }

  function snapshotJobs(): Job[] {
    return jobs.map((j) => ({
      id: j.id,
      file: j.file,
      status: j.status,
      ...(j.errorCode ? { errorCode: j.errorCode } : {}),
    }));
  }

  async function processQueue() {
    if (isPaused) return;
    if (activeJobId !== null) return;

    const nextJob = jobs.find((j) => j.status === "queued");
    if (!nextJob) return;

    const jobId = nextJob.id;
    activeJobId = jobId;
    nextJob.status = "analyzing";
    const controller = new AbortController();
    nextJob.controller = controller;
    onChange();

    try {
      const suggestion = await run(nextJob.file, controller.signal);

      // Verify that this job was not canceled or aborted in-flight
      if (activeJobId !== jobId) {
        return;
      }
      const currentJob = jobs.find((j) => j.id === jobId);
      if (currentJob?.status !== "analyzing") {
        return;
      }

      currentJob.status = "done";
      currentJob.controller = undefined;
      activeJobId = null;
      onChange();

      const callback = currentJob.onResult ?? onResult;
      callback(
        {
          id: currentJob.id,
          file: currentJob.file,
          status: currentJob.status,
          ...(currentJob.errorCode ? { errorCode: currentJob.errorCode } : {}),
        },
        suggestion,
      );
    } catch (error) {
      if (activeJobId !== jobId) {
        return;
      }
      const currentJob = jobs.find((j) => j.id === jobId);
      if (currentJob?.status !== "analyzing") {
        return;
      }

      currentJob.controller = undefined;
      activeJobId = null;

      if (controller.signal.aborted) {
        return;
      }

      if (isRateLimitedError(error)) {
        currentJob.status = "failed";
        currentJob.errorCode = "RATE_LIMITED";
        for (const j of jobs) {
          if (j.status === "queued") {
            j.status = "paused";
          }
        }
        isPaused = true;
        onChange();
        onError?.(
          {
            id: currentJob.id,
            file: currentJob.file,
            status: currentJob.status,
            errorCode: currentJob.errorCode,
          },
          error,
        );
      } else {
        currentJob.status = "failed";
        currentJob.errorCode = extractErrorCode(error);
        onChange();
        onError?.(
          {
            id: currentJob.id,
            file: currentJob.file,
            status: currentJob.status,
            ...(currentJob.errorCode
              ? { errorCode: currentJob.errorCode }
              : {}),
          },
          error,
        );
      }
    } finally {
      if (activeJobId === null && !isPaused) {
        processQueue();
      }
    }
  }

  return {
    add(file: File): string {
      const existing = findJobByFile(file);
      if (existing) {
        return existing.id;
      }

      idCounter += 1;
      const id = `job_${idCounter}_${file.name}`;
      const newJob: InternalJob = {
        id,
        file,
        status: "queued",
      };
      jobs.push(newJob);
      onChange();
      processQueue();
      return id;
    },

    cancel(id: string): void {
      const job = jobs.find((j) => j.id === id);
      if (!job) return;

      if (job.status === "analyzing") {
        job.controller?.abort();
        if (activeJobId === id) {
          activeJobId = null;
        }
      }

      jobs = jobs.filter((j) => j.id !== id);
      onChange();
      processQueue();
    },

    retry(id: string): void {
      const job = jobs.find((j) => j.id === id);
      if (!job || (job.status !== "failed" && job.status !== "paused")) {
        return;
      }

      job.status = "queued";
      job.errorCode = undefined;
      isPaused = false;
      onChange();
      processQueue();
    },

    reanalyze(
      file: File,
      onResultOverride?: (job: Job, suggestion: FinancialSuggestion) => void,
    ): string {
      let job = findJobByFile(file);

      if (job) {
        if (job.status === "analyzing") {
          job.controller?.abort();
          if (activeJobId === job.id) {
            activeJobId = null;
          }
        }
        job.status = "queued";
        job.errorCode = undefined;
        job.file = file;
        job.onResult = onResultOverride;
      } else {
        idCounter += 1;
        const id = `job_${idCounter}_${file.name}`;
        job = {
          id,
          file,
          status: "queued",
          onResult: onResultOverride,
        };
        jobs.push(job);
      }

      isPaused = false;
      onChange();
      processQueue();
      return job.id;
    },

    cancelAll(): void {
      for (const j of jobs) {
        if (j.status === "analyzing") {
          j.controller?.abort();
        }
      }
      activeJobId = null;
      jobs = [];
      isPaused = false;
      onChange();
    },

    jobs(): Job[] {
      return snapshotJobs();
    },

    statusOf(file: File): JobStatus | undefined {
      const job = findJobByFile(file);
      return job?.status;
    },
  };
}
