import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { en } from "~/i18n/en";
import { id } from "~/i18n/id";
import {
  mapOtpErrorKey,
  OtpLoginForm,
  type OtpState,
  otpStepReducer,
  remainingSeconds,
} from "./OtpLoginForm";

vi.mock("react-router", () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (
      key: string,
      options?: { defaultValue?: string; error?: string } | string,
    ) => {
      if (typeof options === "object" && options?.error) {
        return `${key}:${options.error}`;
      }
      if (typeof options === "object" && options?.defaultValue) {
        return options.defaultValue;
      }
      if (typeof options === "string") return options;
      return key;
    },
    i18n: { language: "en" },
  }),
}));

vi.mock("~/lib/auth-client", () => ({
  authClient: {
    otp: {
      request: vi.fn(),
      verify: vi.fn(),
    },
  },
}));

describe("OtpLoginForm logic and rendering", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("otpStepReducer moves to the code step on requested", () => {
    const initial: OtpState = {
      step: "email",
      email: "",
      codeLength: 6,
      resendAt: 0,
    };
    const next = otpStepReducer(initial, {
      type: "requested",
      email: "user@example.com",
      codeLength: 8,
      resendAfterSeconds: 30,
      now: 1000,
    });
    expect(next).toEqual({
      step: "code",
      email: "user@example.com",
      codeLength: 8,
      resendAt: 31000,
    });
  });

  it("otpStepReducer returns to the email step on changeEmail", () => {
    const codeStepState: OtpState = {
      step: "code",
      email: "user@example.com",
      codeLength: 8,
      resendAt: 31000,
    };
    const next = otpStepReducer(codeStepState, { type: "changeEmail" });
    expect(next.step).toBe("email");
    expect(next.email).toBe("user@example.com");
  });

  it("remainingSeconds rounds up and never goes below zero", () => {
    expect(remainingSeconds(10000, 0)).toBe(10);
    expect(remainingSeconds(10000, 100)).toBe(10);
    expect(remainingSeconds(10000, 9001)).toBe(1);
    expect(remainingSeconds(10000, 10000)).toBe(0);
    expect(remainingSeconds(10000, 15000)).toBe(0);
  });

  it("mapOtpErrorKey maps AUTH_OTP_INVALID, AUTH_OTP_UNAVAILABLE, and anything else", () => {
    expect(mapOtpErrorKey("AUTH_OTP_INVALID")).toBe("login.otp.invalid");
    expect(mapOtpErrorKey("AUTH_OTP_UNAVAILABLE")).toBe(
      "login.otp.unavailable",
    );
    expect(mapOtpErrorKey("AUTH_OTP_EXPIRED")).toBe("login.otp.failed");
    expect(mapOtpErrorKey(undefined)).toBe("login.otp.failed");
    expect(mapOtpErrorKey("UNKNOWN")).toBe("login.otp.failed");
  });

  it("renders the email step first", () => {
    const html = renderToString(<OtpLoginForm />);
    expect(html).toContain("login.otp.emailLabel");
    expect(html).toContain("login.otp.sendCode");
  });

  it("en and id define the same login.otp keys", () => {
    const expectedKeys = [
      "changeEmail",
      "codeHint",
      "divider",
      "emailLabel",
      "emailPlaceholder",
      "failed",
      "invalid",
      "resend",
      "resendIn",
      "sendCode",
      "unavailable",
    ].sort();

    const enOtp = en.login.otp;
    const idOtp = id.login.otp;

    expect(Object.keys(enOtp).sort()).toEqual(expectedKeys);
    expect(Object.keys(idOtp).sort()).toEqual(expectedKeys);

    expect(enOtp.divider).toBe("or");
    expect(enOtp.emailLabel).toBe("Email");
    expect(enOtp.emailPlaceholder).toBe("you@example.com");
    expect(enOtp.sendCode).toBe("Send code");
    expect(enOtp.codeHint).toBe(
      "Enter the {{length}}-digit code we sent to {{email}}.",
    );
    expect(enOtp.resend).toBe("Resend code");
    expect(enOtp.resendIn).toBe("Resend code in {{seconds}}s");
    expect(enOtp.changeEmail).toBe("Use a different email");
    expect(enOtp.invalid).toBe("That code is invalid or expired.");
    expect(enOtp.unavailable).toBe("Email sign-in is not available right now.");
    expect(enOtp.failed).toBe("Could not send the code. Try again.");

    expect(idOtp.divider).toBe("atau");
    expect(idOtp.emailLabel).toBe("Email");
    expect(idOtp.emailPlaceholder).toBe("anda@contoh.com");
    expect(idOtp.sendCode).toBe("Kirim kode");
    expect(idOtp.codeHint).toBe(
      "Masukkan kode {{length}} digit yang kami kirim ke {{email}}.",
    );
    expect(idOtp.resend).toBe("Kirim ulang kode");
    expect(idOtp.resendIn).toBe("Kirim ulang kode dalam {{seconds}} dtk");
    expect(idOtp.changeEmail).toBe("Gunakan email lain");
    expect(idOtp.invalid).toBe("Kode tidak valid atau sudah kedaluwarsa.");
    expect(idOtp.unavailable).toBe(
      "Masuk dengan email belum tersedia saat ini.",
    );
    expect(idOtp.failed).toBe("Gagal mengirim kode. Coba lagi.");
  });
});
