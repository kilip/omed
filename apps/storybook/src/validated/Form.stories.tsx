import { Form, ValidatedInputText } from "@omed/ui";
import { withZod } from "@remix-validated-form/with-zod";
import { Meta, StoryObj } from "@storybook/react";
import z from "zod";

const validator = withZod(
  z.object({
    email: z.string().min(4),
    password: z.string().min(4),
  })
);

const meta = {
  title: "Validated/Form",
  component: Form,
} satisfies Meta<typeof Form>;

export default meta;

type Story = StoryObj<typeof meta>;

export const LoginForm: Story = {
  args: {
    title: "Login Form",
    validator,
    children: (
      <>
        <ValidatedInputText name="email" />
        <ValidatedInputText name="password" />
      </>
    ),
  },
};
