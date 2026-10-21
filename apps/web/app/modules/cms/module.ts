import {
  FileOutlined,
  FileTextOutlined,
  PictureOutlined,
  ReadOutlined,
} from "@ant-design/icons";
import { defineModule } from "~/lib/modules";

export default defineModule({
  id: "cms",
  labelKey: "cms.label",
  descriptionKey: "cms.description",
  icon: ReadOutlined,
  order: 40,
  menu: [
    {
      key: "posts",
      path: "posts",
      labelKey: "cms.menu.posts",
      icon: FileTextOutlined,
    },
    {
      key: "pages",
      path: "pages",
      labelKey: "cms.menu.pages",
      icon: FileOutlined,
    },
    {
      key: "media",
      path: "media",
      labelKey: "cms.menu.media",
      icon: PictureOutlined,
    },
  ],
});
