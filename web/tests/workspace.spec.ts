import { test, expect } from "@playwright/test";
test("注册、创建 Endpoint、真实捕获、搜索、详情、管理和退出", async ({
  page,
  request,
}) => {
  const email = `browser-${Date.now()}@example.com`;
  await page.goto("/");
  await page.getByRole("button", { name: "创建账户", exact: true }).click();
  await page.getByLabel("邮箱地址").fill(email);
  await page
    .getByLabel("密码", { exact: true })
    .fill("test-browser-password-123");
  await page.getByRole("button", { name: "注册并进入工作台" }).click();
  await expect(
    page.getByRole("heading", { name: "Webhook 工作台", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "新建 Endpoint", exact: true })
    .first()
    .click();
  await page.getByLabel("Endpoint 名称").fill("Stripe production");
  await page
    .getByRole("button", { name: "创建 Endpoint", exact: true })
    .click();
  await expect(page.getByTestId("endpoint-url")).toBeVisible();
  const url = await page.getByTestId("endpoint-url").textContent();
  expect(url).toContain("/h/");
  const response = await request.post(url! + "/payment?version=2", {
    data: { type: "payment.succeeded", amount: 2048 },
    headers: {
      Authorization: "Bearer SUPER_SECRET",
      "Content-Type": "application/json",
    },
  });
  expect(response.status()).toBe(202);
  await page.getByRole("button", { name: "刷新事件" }).click();
  await expect(
    page.getByRole("button", { name: /查看事件.*payment/ }),
  ).toBeVisible();
  await page.getByRole("button", { name: /查看事件.*payment/ }).click();
  await expect(page.getByRole("heading", { name: "事件详情" })).toBeVisible();
  await expect(page.getByTestId("body-viewer")).toContainText(
    "payment.succeeded",
  );
  await page.getByRole("button", { name: "Raw", exact: true }).click();
  await expect(page.getByTestId("body-viewer")).toContainText(
    '{"type":"payment.succeeded","amount":2048}',
  );
  await page.getByRole("button", { name: "Query", exact: true }).click();
  await expect(page.getByText("version=2", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "重放事件", exact: true }).click();
  await page
    .getByLabel("目标 URL")
    .fill("http://169.254.169.254/latest/meta-data");
  await page.getByRole("button", { name: "发送重放" }).click();
  await expect(page.getByRole("alert")).toContainText("仅允许公网");
  await page.getByRole("button", { name: "取消", exact: true }).click();
  await page.getByRole("button", { name: "Headers", exact: true }).click();
  await expect(page.getByTestId("headers-viewer")).toContainText("[REDACTED]");
  await expect(page.getByTestId("headers-viewer")).toContainText(
    new URL(url!).host,
  );
  await expect(page.getByTestId("headers-viewer")).not.toContainText(
    "SUPER_SECRET",
  );
  await page.getByRole("button", { name: "关闭详情" }).click();
  await page.getByPlaceholder("搜索路径或请求内容…").fill("nonexistent");
  await expect(page.getByText("没有匹配的事件")).toBeVisible();
  await page.getByPlaceholder("搜索路径或请求内容…").fill("payment.succeeded");
  await expect(
    page.getByRole("button", { name: /查看事件.*payment/ }),
  ).toBeVisible();
  await page.getByRole("button", { name: "管理当前 Endpoint" }).click();
  await page.getByLabel("Endpoint 名称").fill("Stripe renamed");
  await page.getByRole("button", { name: "保存修改" }).click();
  await expect(
    page.getByRole("heading", { name: "Stripe renamed", exact: true }),
  ).toBeVisible();
  await page.screenshot({ path: "../work/desktop.png", fullPage: true });
  await page.getByRole("button", { name: "退出登录" }).click();
  await expect(page.getByRole("heading", { name: "欢迎回来" })).toBeVisible();
  await page.getByLabel("邮箱地址").fill(email);
  await page
    .getByLabel("密码", { exact: true })
    .fill("test-browser-password-123");
  await page.getByRole("button", { name: "登录工作台", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Stripe renamed", exact: true }),
  ).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await expect
    .poll(async () =>
      page
        .locator(".sidebar")
        .evaluate((el) => el.getBoundingClientRect().right),
    )
    .toBeLessThanOrEqual(0);
  await page.getByRole("button", { name: "打开导航" }).click();
  await expect(page.getByRole("navigation", { name: "主导航" })).toBeVisible();
  await page.getByRole("button", { name: "工作台", exact: true }).click();
  await expect
    .poll(async () =>
      page
        .locator(".sidebar")
        .evaluate((el) => el.getBoundingClientRect().right),
    )
    .toBeLessThanOrEqual(0);
  await page.screenshot({ path: "../work/mobile.png", fullPage: true });
  await page.getByRole("button", { name: "管理当前 Endpoint" }).click();
  await page
    .getByRole("button", { name: "删除 Endpoint", exact: true })
    .click();
  await page.getByRole("button", { name: "确认永久删除" }).click();
  await expect(page.getByText("创建你的第一个 Endpoint")).toBeVisible();
});
