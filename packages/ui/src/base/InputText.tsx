import { Flex, Text, TextField, type TextFieldInput } from "@radix-ui/themes";
import { upperFirst } from "lodash";

type InputTextProps = React.ComponentPropsWithoutRef<typeof TextFieldInput> & {
  name: string;
  label?: string;
  error?: string;
};

export function InputText({
  name,
  label,
  error,
  ...rest
}: InputTextProps): JSX.Element {
  const _label = label ?? upperFirst(name.replace(/-|_/, " "));
  const merged = {
    ...rest,
    ...{
      name,
    },
  };

  return (
    <Flex
      direction="column"
      gap="2">
      <Text
        as="label"
        htmlFor={name}
        weight="medium">
        {_label}
      </Text>
      <TextField.Input {...merged} />
      {error ? (
        <Text
          as="span"
          color="red"
          size="2"
          weight="medium">
          {error}
        </Text>
      ) : null}
    </Flex>
  );
}
