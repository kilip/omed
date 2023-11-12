import { Card, Flex, Heading } from "@radix-ui/themes";
import type { PropsWithChildren } from "react";
import React from "react";
import { ValidatedForm } from "remix-validated-form";

type FormProps = React.ComponentPropsWithoutRef<typeof ValidatedForm> &
  PropsWithChildren & {
    title?: string;
    cardProps?: React.ComponentPropsWithoutRef<typeof Card>;
  };

export function Form({
  title,
  children,
  cardProps,
  ...rest
}: FormProps): JSX.Element {
  return (
    <Card {...cardProps}>
      <Flex
        className="p-2"
        direction="column">
        {title ? <Heading mb="4">{title}</Heading> : null}
        <ValidatedForm {...rest}>
          <Flex
            direction="column"
            gap="3">
            {children}
          </Flex>
        </ValidatedForm>
      </Flex>
    </Card>
  );
}
