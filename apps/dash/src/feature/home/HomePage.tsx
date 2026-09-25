import type { Metadata } from "next";
import Status from "./component/Status";

export const metadata: Metadata = {
  title: "Home",
};

export default async function HomePage() {
  return (
    <div>
      <Status />
    </div>
  );
}
