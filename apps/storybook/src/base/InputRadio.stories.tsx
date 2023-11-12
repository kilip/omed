import { InputRadio } from "@omed/ui";
import { Card } from "@radix-ui/themes";
import { Meta, StoryObj } from "@storybook/react";

const meta = {
  title: "Base/InputRadio",
  component: InputRadio,
  decorators: [
    (Story) => {
      return (
        <Card style={{ maxWidth: 400 }}>
          <Story />
        </Card>
      );
    },
  ],
} satisfies Meta<typeof InputRadio>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Gender: Story = {
  args: {
    name: "Gender",
    items: [
      {
        label: "Male",
      },
      {
        label: "Female",
      },
    ],
  },
};
