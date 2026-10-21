import { expect, test } from "@playwright/test";
test("catalog search selects a local widget and explains its requirements", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByRole("button", { name: "위젯 추가", exact: true }).click();
  await page.getByLabel("카탈로그 검색", { exact: true }).fill("메모");
  await expect(
    page.getByRole("button", { name: "메모 선택", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "세계 시계 선택", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "메모 선택", exact: true }).click();
  await expect(page.getByLabel("위젯 종류", { exact: true })).toHaveValue(
    "note",
  );
  await page
    .getByLabel("카탈로그 검색", { exact: true })
    .fill("does-not-exist");
  await expect(
    page.getByRole("status").filter({ hasText: "일치하는 위젯이 없습니다" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "닫기", exact: true }).click();
});

test("field errors identify and focus the invalid input while preview and cancel do not save", async ({
  page,
  request,
}) => {
  const before = await (await request.get("/api/v1/dashboard")).json();
  await page.goto("/");
  await page.getByRole("button", { name: "위젯 추가", exact: true }).click();
  await page.getByLabel("위젯 종류", { exact: true }).selectOption("clock");
  await page.getByLabel("시간대", { exact: true }).fill("Not/A_Timezone");
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "위젯 추가", exact: true })
    .click();
  await expect(page.getByLabel("시간대", { exact: true })).toBeFocused();
  await expect(page.getByLabel("시간대", { exact: true })).toHaveAttribute(
    "aria-invalid",
    "true",
  );
  await expect(page.getByRole("alert")).toContainText("시간대를 확인");
  await page.getByLabel("시간대", { exact: true }).fill("Asia/Seoul");
  await expect(page.getByLabel("위젯 미리보기")).toBeVisible();
  await expect(page.getByLabel("위젯 미리보기")).toContainText(
    "저장 전에는 새 정보를 가져오지 않습니다.",
  );
  await page.getByRole("button", { name: "닫기", exact: true }).click();
  const after = await (await request.get("/api/v1/dashboard")).json();
  expect(after).toEqual(before);
});

test('command palette supports keyboard navigation and ignores composing or modal typing',async({page})=>{
 await page.goto('/');await expect(page.getByRole('heading',{name:'오늘도, 한눈에.'})).toBeVisible();
 await page.keyboard.press('Control+k');await expect(page.getByRole('dialog',{name:'무엇을 할까요?'})).toBeVisible();
 await page.getByRole('combobox',{name:'명령 검색'}).fill('데이터 소스');await page.keyboard.press('Enter');
 await expect(page.getByRole('heading',{name:'데이터 소스',exact:true})).toBeVisible();
 await page.keyboard.press('Control+k');await page.getByRole('combobox',{name:'명령 검색'}).fill('위젯 추가');await page.keyboard.press('Enter');
 await expect(page.getByRole('dialog',{name:'새 위젯 추가'})).toBeVisible();
 await page.getByLabel('카탈로그 검색',{exact:true}).fill('note/k');await expect(page.getByLabel('카탈로그 검색',{exact:true})).toHaveValue('note/k');
 await page.keyboard.press('Control+k');await expect(page.getByRole('dialog',{name:'무엇을 할까요?'})).toHaveCount(0);
 await page.keyboard.press('Escape');
 await page.evaluate(()=>window.dispatchEvent(new KeyboardEvent('keydown',{key:'k',ctrlKey:true,isComposing:true,bubbles:true})));
 await expect(page.getByRole('dialog',{name:'무엇을 할까요?'})).toHaveCount(0);
});
