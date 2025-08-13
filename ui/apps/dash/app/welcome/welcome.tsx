import { Button } from "@omed/ui/components/button";
import { useState } from "react";

export function Welcome() {
  const [text, setText] = useState("Hello World");
  function clicked() {
    setText("Hello World Clicked");
  }

  return (
    <main className="flex flex-col items-center justify-center pt-16 pb-4">
      <h1 className="text-4xl font-extrabold">{text}</h1>
      <Button onClick={clicked} variant={"destructive"}>
        Hello World
      </Button>
    </main>
  );
}
