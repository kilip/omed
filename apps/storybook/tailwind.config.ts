import type { Config } from "tailwindcss";

export default {
  content: ["./src/**/*.ts", "./node_modules/@omed/ui/**/*.tsx"],
  theme: {
    extend: {},
  },
  plugins: [],
} satisfies Config;
