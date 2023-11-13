import { beforeEach } from "vitest";
import { mockDeep, mockReset } from "vitest-mock-extended";
import { users as baseUsers } from "../users";

// 2
beforeEach(() => {
  mockReset(users);
});

// 3
const users = mockDeep<typeof baseUsers>();
export default users;
