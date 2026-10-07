import { redirect } from "react-router";

export async function clientLoader() {
	throw redirect("/home");
}

export default function IndexRoute() {
	return <div></div>;
}
