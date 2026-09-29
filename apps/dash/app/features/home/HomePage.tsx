import Status from "./component/Status";

export const meta = () => [{ title: "Home" }];
export default function HomePage() {
  return (
    <div>
      <Status />
    </div>
  );
}
