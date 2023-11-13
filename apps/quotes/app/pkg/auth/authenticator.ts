// app/services/auth.server.ts
import type { User } from "@omed/db";
import { Authenticator } from "remix-auth";
import { FormStrategy } from "remix-auth-form";
import { login, sessionStorage } from ".";

function authFormStrategy(authenticator: Authenticator) {
  authenticator.use(
    new FormStrategy(async ({ form }) => {
      const email = form.get("email") as string;
      const password = form.get("password") as string;
      const user = await login({ email, password });
      // the type of this user must match the type you pass to the Authenticator
      // the strategy will automatically inherit the type if you instantiate
      // directly inside the `use` method
      return null;
    }),
    // each strategy has a name and can be changed to use another one
    // same strategy multiple times, especially useful for the OAuth2 strategy.
    "user-pass"
  );
}

type AuthStrategy = (authenticator: Authenticator) => void;

// Create an instance of the authenticator, pass a generic with what
// strategies will return and will store in the session
const authenticator = new Authenticator<User>(sessionStorage);

function registerStrategy(authenticator: Authenticator) {
  const strategies: AuthStrategy[] = [authFormStrategy];
  strategies.forEach((strategy) => {
    console.log(strategy);
    strategy(authenticator);
  });
}

registerStrategy(authenticator);
export default authenticator;
