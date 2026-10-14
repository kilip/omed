import { UserOutlined } from "@ant-design/icons";
import { defineModule } from "~/lib/modules";

export default defineModule({
  id: "user",
  labelKey: "user.label",
  descriptionKey: "user.description",
  icon: UserOutlined,
  order: 20,
  menu: [
    {
      key: "users",
      path: "",
      labelKey: "user.menu.users",
      icon: UserOutlined,
    },
  ],
});
