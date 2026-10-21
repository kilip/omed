import { redirect } from "react-router";

export function clientLoader() {
  return redirect("posts");
}

export default function CmsIndex() {
  return null;
}
