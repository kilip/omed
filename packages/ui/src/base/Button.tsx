import { CheckCircledIcon, ReloadIcon, TrashIcon } from "@radix-ui/react-icons";
import { Button as BaseButton } from "@radix-ui/themes";

const buttonAction = {
  save: "Save",
  delete: "Delete",
  reset: "Reset",
} as const;

interface ButtonProps
  extends React.ComponentPropsWithoutRef<typeof BaseButton> {
  action?: keyof typeof buttonAction;
}

export function Button({
  action = "save",
  variant = "solid",
  color,
  size = "2",
  ...rest
}: ButtonProps): JSX.Element {
  const label = buttonAction[action];
  let ButtonIcon: typeof CheckCircledIcon;
  let mcolor: typeof color;

  switch (action) {
    case "delete":
      mcolor = "red";
      ButtonIcon = TrashIcon;
      break;
    case "reset":
      mcolor = "blue";
      ButtonIcon = ReloadIcon;
      break;
    default:
      mcolor = "green";
      ButtonIcon = CheckCircledIcon;
  }
  const props = {
    ...rest,
    ...{
      color: mcolor,
      variant,
      size,
    },
  };
  return (
    <BaseButton {...props}>
      <ButtonIcon
        height="18"
        width="18"
      />
      {label}
    </BaseButton>
  );
}
