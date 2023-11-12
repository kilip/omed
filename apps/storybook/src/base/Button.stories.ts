import { Button } from "@omed/ui";
import type { Meta, StoryObj } from "@storybook/react";

const meta = {
  title: "Base/Button",
  component: Button,
} satisfies Meta<typeof Button>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Save: Story = {
  args: {
    action: "save",
  },
};

export const Delete: Story = {
  args: {
    action: "delete",
  },
};

export const Reset: Story = {
  args: {
    action: "reset",
  },
};
