import { Card, Empty, Typography } from "antd";
import { UnderConstruction } from "~/shared/ui/UnderConstruction";

export function meta() {
	return [
		{ title: "Omed | Home" },
		{ name: "description", content: "Welcome to React Router!" },
	];
}

export default function Home() {
	return <UnderConstruction pageTitle="Home" />;
}
