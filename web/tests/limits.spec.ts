import { test, expect } from "@playwright/test";
async function account(page: import("@playwright/test").Page) {
  const origin = process.env.BASE_URL || "http://localhost:5173";
  const r = await page.request.post("/api/auth/register", {
    data: {
      email: `limit-${Date.now()}@example.com`,
      password: "password-for-limits-123",
    },
    headers: { Origin: origin },
  });
  expect(r.status()).toBe(201);
  return { origin, ...(await r.json()) };
}
test("大请求体预览限制 DOM 节点并保留下载", async ({ page }) => {
  const a = await account(page);
  const r = await page.request.post("/api/endpoints", {
    data: { name: "large payload" },
    headers: { Origin: a.origin, "X-CSRF-Token": a.csrf },
  });
  const endpoint = await r.json();
  const captured = await page.request.post("/h/" + endpoint.token + "/large", {
    data: "\n".repeat(20000),
    headers: { "Content-Type": "text/plain" },
  });
  expect(captured.status()).toBe(202);
  await page.goto("/");
  await page.getByRole("button", { name: "刷新事件" }).click();
  await page.getByRole("button", { name: "查看事件 POST /large" }).click();
  await expect(page.getByTestId("body-viewer")).toBeVisible();
  expect(
    await page.locator(".line-numbers > span").count(),
  ).toBeLessThanOrEqual(200);
  await expect(page.getByText(/已限制预览/)).toBeVisible();
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "下载原始 Body" }).click();
  expect((await download).suggestedFilename()).toMatch(/\.bin$/);
});
test("忙碌弹窗拒绝 Escape，失败仍可见", async ({ page }) => {
  await account(page);
  await page.goto("/");
  await page
    .getByRole("button", { name: "新建 Endpoint", exact: true })
    .first()
    .click();
  await page.getByLabel("Endpoint 名称").fill("中".repeat(30));
  let release!: () => void;
  let started!: () => void;
  const pending = new Promise<void>((resolve) => (release = resolve));
  const ready = new Promise<void>((resolve) => (started = resolve));
  await page.route("**/api/endpoints", async (route) => {
    if (route.request().method() !== "POST") {
      await route.continue();
      return;
    }
    const response = await route.fetch();
    started();
    await pending;
    await route.fulfill({ response });
  });
  await page
    .getByRole("button", { name: "创建 Endpoint", exact: true })
    .click();
  await ready;
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toBeVisible();
  release();
  await expect(page.getByRole("alert")).toContainText("1–80");
  await page.getByRole("button", { name: "取消", exact: true }).click();
});
