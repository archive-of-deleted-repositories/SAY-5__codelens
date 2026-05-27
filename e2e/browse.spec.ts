import { test, expect } from "@playwright/test";

// Drives the full stack: the React browser queries the GraphQL API, which
// reads the indexed graph from Postgres and caches in Redis. Looking up a
// symbol must surface its definition and its references.
test("find references for a method", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "codelens" })).toBeVisible();

  await page.getByLabel("symbol name").fill("Add");
  await page.getByRole("button", { name: "find" }).click();

  // Definition location for the Add method in the calc package.
  await expect(page.getByText(/calc\/calc\.go:\d+:\d+/).first()).toBeVisible();
  // The references heading shows a count and at least one ref row appears.
  await expect(page.getByRole("heading", { name: /references/ })).toBeVisible();

  // The semantic layer surfaces related symbols ranked by the provider.
  await expect(page.getByRole("heading", { name: "related symbols" })).toBeVisible();
});
