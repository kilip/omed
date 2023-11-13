import { Box, Card, Heading } from "@radix-ui/themes";
import type { MetaFunction } from "@remix-run/node";
import { Link } from "@remix-run/react";

export const meta: MetaFunction = () => {
  return [
    { title: "Quotes" },
    { name: "description", content: "Welcome to Quotes!" },
  ];
};

export default function Index() {
  return (
    <main className="flex w-full min-h-screen items-center justify-center">
      <Box>
        <Card m="8">
          <Heading className="p-8">Welcome to Quotes!</Heading>
          <Link to="/login">Login</Link>
        </Card>
      </Box>
    </main>
  );
}
