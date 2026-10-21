import dayjs from "dayjs";
import { describe, expect, it } from "vitest";
import { en } from "~/i18n/en";
import { id } from "~/i18n/id";
import {
  getAvailableTransitions,
  getEffectiveStatus,
  isPastDate,
} from "./publish-panel";

describe("PublishPanel transition logic & past-date blocking", () => {
  const fixedNow = new Date("2026-10-21T12:00:00Z");

  describe("getEffectiveStatus", () => {
    it("returns DRAFT when status is DRAFT", () => {
      expect(getEffectiveStatus("DRAFT", null, fixedNow)).toBe("DRAFT");
      expect(
        getEffectiveStatus("DRAFT", "2026-10-25T12:00:00Z", fixedNow),
      ).toBe("DRAFT");
    });

    it("returns SCHEDULED when status is PUBLISHED and publishAt is in the future", () => {
      expect(
        getEffectiveStatus("PUBLISHED", "2026-10-21T13:00:00Z", fixedNow),
      ).toBe("SCHEDULED");
    });

    it("returns PUBLISHED when status is PUBLISHED and publishAt is in the past or now", () => {
      expect(
        getEffectiveStatus("PUBLISHED", "2026-10-21T11:00:00Z", fixedNow),
      ).toBe("PUBLISHED");
      expect(
        getEffectiveStatus("PUBLISHED", "2026-10-21T12:00:00Z", fixedNow),
      ).toBe("PUBLISHED");
    });
  });

  describe("getAvailableTransitions", () => {
    it("allows publish and schedule, but not unpublish, for DRAFT", () => {
      const transitions = getAvailableTransitions("DRAFT");
      expect(transitions.canPublish).toBe(true);
      expect(transitions.canSchedule).toBe(true);
      expect(transitions.canUnpublish).toBe(false);
    });

    it("allows publish, schedule, and unpublish for SCHEDULED", () => {
      const transitions = getAvailableTransitions("SCHEDULED");
      expect(transitions.canPublish).toBe(true);
      expect(transitions.canSchedule).toBe(true);
      expect(transitions.canUnpublish).toBe(true);
    });

    it("allows only unpublish for PUBLISHED", () => {
      const transitions = getAvailableTransitions("PUBLISHED");
      expect(transitions.canPublish).toBe(false);
      expect(transitions.canSchedule).toBe(false);
      expect(transitions.canUnpublish).toBe(true);
    });
  });

  describe("past-date blocking (isPastDate)", () => {
    it("identifies past dates and times as blocked", () => {
      expect(isPastDate("2026-10-21T11:59:59Z", fixedNow)).toBe(true);
      expect(isPastDate(dayjs(fixedNow).subtract(1, "day"), fixedNow)).toBe(
        true,
      );
      expect(isPastDate(fixedNow, fixedNow)).toBe(true);
    });

    it("identifies future dates and times as allowed", () => {
      expect(isPastDate("2026-10-21T12:00:01Z", fixedNow)).toBe(false);
      expect(isPastDate(dayjs(fixedNow).add(2, "hours"), fixedNow)).toBe(false);
      expect(isPastDate(dayjs(fixedNow).add(1, "day"), fixedNow)).toBe(false);
    });
  });

  describe("status translations completeness", () => {
    it("en and id define matching status translations", () => {
      const statuses = ["DRAFT", "SCHEDULED", "PUBLISHED"] as const;
      for (const s of statuses) {
        expect(en.cms.status[s]).toBeDefined();
        expect(id.cms.status[s]).toBeDefined();
        expect(en.cms.status[s].length).toBeGreaterThan(0);
        expect(id.cms.status[s].length).toBeGreaterThan(0);
      }
    });

    it("en and id define matching publish action translations", () => {
      const keys = [
        "publishNow",
        "schedule",
        "unpublish",
        "unpublishConfirm",
        "scheduleTitle",
        "scheduleConfirm",
        "scheduleDesc",
        "scheduledFor",
        "publishedAt",
        "pastDateError",
      ] as const;
      for (const k of keys) {
        expect(en.cms.publish[k]).toBeDefined();
        expect(id.cms.publish[k]).toBeDefined();
      }
    });
  });
});
