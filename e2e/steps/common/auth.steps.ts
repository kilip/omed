import { Given } from "@cucumber/cucumber";
import type { CustomWorld } from "@e2e/support/world";
import type { User } from "@/auth";

Given("I have logged in as test user", async function (this: CustomWorld) {
  const testUser = this.auth.createUser({
    name: "Test User",
    email: "test@example.com",
    emailVerified: true,
  });

  let user: User | undefined = await this.db.query.users.findFirst({
    where: { email: testUser.email },
  });

  if (!user) {
    user = (await this.auth.saveUser(testUser)) as unknown as User;
  }

  const cookies = await this.auth.getCookies({
    userId: user.id,
  });

  this.context.userIds.push(user.id);
  await this.browserContext.addCookies(cookies);
});
