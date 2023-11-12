import { Box, Card, Heading } from "@radix-ui/themes";
import type { MetaFunction } from "@remix-run/node";

export const meta: MetaFunction = () => {
  return [
    { title: "New Remix App" },
    { name: "description", content: "Welcome to Remix!" },
  ];
};

export default function Index() {
  return (
    <main className="flex w-full min-h-screen items-center justify-center">
      <Box>
        <Card m="8">
          <Heading className="p-8">Welcome to Quotes!</Heading>
        </Card>
      </Box>
    </main>
  );
}
