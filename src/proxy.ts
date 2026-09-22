import { headers } from "next/headers";
import { type NextRequest, NextResponse } from "next/server";
import { auth } from "@/auth";

export async function proxy(request: NextRequest) {
  const session = await auth.api.getSession({
    headers: await headers(),
  });

  if (!session) {
    const url = new URL("/login", request.url);
    url.searchParams.set("ref", request.nextUrl.pathname);
    return NextResponse.redirect(url.href);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/home"], // Specify the routes the middleware applies to
};
