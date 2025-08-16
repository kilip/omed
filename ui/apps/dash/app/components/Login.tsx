import { Button } from "@omed/ui/components/button";

type Props = {
  error?: string;
};

export default function Login({ error }: Props) {
  return (
    <div>
      <h1>Login to Omed</h1>
      {error && <div>{error}</div>}

      <form method="POST" className="flex flex-col">
        <div>
          <p>Please sign in</p>
        </div>
        <label>
          Email: <input type="text" name="email" />
        </label>
        <label>
          Password: <input type="password" name="password" />
        </label>
        <div>
          <Button>Login</Button>
        </div>
      </form>
    </div>
  );
}
