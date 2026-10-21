import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import ReactMarkdown from "react-markdown";
import { describe, expect, it } from "vitest";

describe("MarkdownEditor / ReactMarkdown preview", () => {
  it("renders markdown headings, lists, and formatting", () => {
    const md = "# Heading 1\n\n* Item 1\n* Item 2\n\n**Bold text**";
    const html = renderToStaticMarkup(createElement(ReactMarkdown, null, md));
    expect(html).toContain("<h1>Heading 1</h1>");
    expect(html).toContain("<li>Item 1</li>");
    expect(html).toContain("<strong>Bold text</strong>");
  });

  it("does not render raw HTML tags or execute scripts (D10)", () => {
    const unsafeMd =
      'Hello <script>alert("xss")</script> <div id="raw">dangerous</div>';
    const html = renderToStaticMarkup(
      createElement(ReactMarkdown, null, unsafeMd),
    );
    expect(html).not.toContain("<script>");
    expect(html).not.toContain('<div id="raw">');
    // Raw HTML should be escaped or stripped
  });
});
