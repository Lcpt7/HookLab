import { test, expect } from "@playwright/test";
test("登录失败显示真实服务错误，未登录 API 禁止访问", async ({
  page,
  request,
}) => {
  await page.goto("/");
  await page.getByLabel("邮箱地址").fill("missing@example.com");
  await page.getByLabel("密码", { exact: true }).fill("wrong-password-123");
  await page.getByRole("button", { name: "登录工作台", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("邮箱或密码不正确");
  expect((await request.get("/api/endpoints")).status()).toBe(401);
  await page.screenshot({ path: "../work/auth.png", fullPage: true });
});
