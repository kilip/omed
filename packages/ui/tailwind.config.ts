/* eslint-disable unicorn/filename-case -- disable filename case */
import type { Config } from "tailwindcss";

// eslint-disable-next-line import/no-default-export -- disable
export default {
  content: ["./src/**/*.tsx"],
  theme: {
    extend: {},
  },
  plugins: [],
} satisfies Config;
