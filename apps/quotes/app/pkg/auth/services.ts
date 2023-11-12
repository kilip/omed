import { withZod } from "@remix-validated-form/with-zod";
import z from "zod";
export const LoginValidator = withZod(
  z.object({
    email: z.string().min(1, "Please enter your registered mail address"),
    password: z.string().min(1, "Please entel your password!"),
  })
);
