import { Form, ValidatedInputText } from "@omed/ui";
import { Button, Flex } from "@radix-ui/themes";
import { LoginValidator } from "../services";

export default function Login() {
  return (
    <Form
      validator={LoginValidator}
      title="Login to Quotes"
      style={{ minWidth: 300 }}>
      <Flex
        direction="column"
        gap="2">
        <Button
          type="button"
          style={{ backgroundColor: "black" }}>
          Login / Register with GitHub
        </Button>
        <Button
          type="button"
          color="red">
          Login / Register with Google
        </Button>
      </Flex>

      <Flex
        direction="column"
        className="p-4">
        <ValidatedInputText
          name="email"
          label="Email address"
        />
        <ValidatedInputText
          type="password"
          name="password"
        />
      </Flex>
      <Flex>
        <Button>Login</Button>
      </Flex>
    </Form>
  );
}
