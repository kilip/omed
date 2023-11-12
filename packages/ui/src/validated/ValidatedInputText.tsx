import { useField } from "remix-validated-form";
import { InputText } from "../base";

type ValidatedInputTextProps = React.ComponentPropsWithoutRef<typeof InputText>;

export function ValidatedInputText({
  name,
  ...rest
}: ValidatedInputTextProps): JSX.Element {
  const { error, getInputProps } = useField(name);

  const props = {
    ...rest,
    ...{
      error,
    },
  };
  return (
    <InputText
      {...props}
      {...getInputProps({ id: name })}
    />
  );
}
