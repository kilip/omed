import { InputText } from "@omed/ui";
import { Card } from "@radix-ui/themes";
import { Meta, StoryObj } from "@storybook/react";
const meta = {
  title: "Base/InputText",
  component: InputText,
  decorators: [
    (Story) => {
      return (
        <Card>
          <Story />
        </Card>
      );
    },
  ],
} satisfies Meta<typeof InputText>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Email: Story = {
  args: {
    name: "email_address",
  },
};
export const Password: Story = {
  args: {
    name: "password",
    placeholder: "Enter your password",
    type: "password",
  },
};
export const WithError: Story = {
  args: {
    name: "input_with_error",
    placeholder: "Error input",
    error: "Some error message",
  },
};
