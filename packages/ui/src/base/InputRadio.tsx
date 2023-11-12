import { Flex, RadioGroup, Text } from "@radix-ui/themes";
import { camelCase, upperFirst } from "lodash";

interface RadioItem {
  label: string;
  value?: string;
}

interface InputRadioProps
  extends React.ComponentPropsWithoutRef<typeof RadioGroup.Root> {
  name: string;
  label?: string;
  items: RadioItem[];
}

function InputRadioItem({ item }: { item: RadioItem }): JSX.Element {
  const value = item.value ?? item.label;

  return (
    <Text
      as="label"
      key={`${camelCase(item.label.toString())}`}>
      <Flex gap="2">
        <RadioGroup.Item value={value} />
        {item.label}
      </Flex>
    </Text>
  );
}
export function InputRadio({
  name,
  label,
  items,
  highContrast = true,
  variant = "classic",
  color = "purple",
  ...rest
}: InputRadioProps): JSX.Element {
  const mLabel = label ?? upperFirst(name.replace(/-|_/, " "));
  const props = {
    ...rest,
    ...{
      name,
      highContrast,
      variant,
      color,
    },
  };
  return (
    <RadioGroup.Root {...props}>
      <Text weight="bold">{mLabel}</Text>
      <Flex
        align="center"
        gap="2"
        mt="2">
        {items.map((item, index) => (
          <InputRadioItem
            item={item}
            key={camelCase(item.label) + index.toString()}
          />
        ))}
      </Flex>
    </RadioGroup.Root>
  );
}
