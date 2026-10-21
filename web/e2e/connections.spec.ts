import {expect,test} from '@playwright/test';
test('connection secrets stay private through creation, failed test, revoke and deletion',async({page,request})=>{
 await page.route('**/api/v1/connections/*/test',route=>route.fulfill({contentType:'application/json',body:JSON.stringify({connected:false,capabilities:[],errorCode:'authentication'})}));
 await page.goto('/#/sources');await page.getByRole('button',{name:'연결 추가',exact:true}).click();
 await page.getByLabel('연결 이름',{exact:true}).fill('검증용 연결');await page.getByLabel('접근 토큰',{exact:true}).fill('fixture_token_never_sent_to_github');await page.getByRole('button',{name:'연결 저장',exact:true}).click();
 const card=page.locator('article.resource-card').filter({has:page.getByRole('heading',{name:'검증용 연결',exact:true})});
 await expect(card.getByText(/비밀정보 보관됨/)).toBeVisible();
 const metadata=await(await request.get('/api/v1/connections')).text();expect(metadata).not.toContain('fixture_token');expect(metadata).not.toContain('ciphertext');
 await card.getByRole('button',{name:'검증용 연결 연결 테스트'}).click();await expect(card.getByText(/인증 정보를 확인/)).toBeVisible();
 page.on('dialog',dialog=>dialog.accept());await card.getByRole('button',{name:'검증용 연결 연결 철회'}).click();await expect(card.getByText(/철회됨.*저장된 비밀정보 없음/)).toBeVisible();await expect(card.getByRole('button',{name:'검증용 연결 연결 테스트'})).toBeDisabled();
 await card.getByRole('button',{name:'검증용 연결 연결 삭제'}).click();await expect(card).toHaveCount(0);
});
test('source pause persists, identifies affected boards and resumes',async({page,request})=>{
 const sources=await(await request.get('/api/v1/sources')).json();const source=sources.find((item:{source:{type:string}})=>item.source.type==='rss');expect(source).toBeTruthy();
 const hostname=new URL(source.source.config.url).hostname;
 try{
 await page.goto('/#/sources');const card=page.locator('article.resource-card').filter({has:page.getByRole('heading',{name:hostname,exact:true})});
 await card.getByRole('button',{name:`${hostname} 일시정지`,exact:true}).click();await expect(card.getByRole('button',{name:`${hostname} 계속 갱신`,exact:true})).toBeVisible();await page.reload();await expect(card.getByRole('button',{name:`${hostname} 지금 갱신`,exact:true})).toBeDisabled();await expect(card.getByText(/개 보드에서 사용 중/)).toBeVisible();
 await card.getByRole('button',{name:`${hostname} 계속 갱신`,exact:true}).click();await expect(card.getByRole('button',{name:`${hostname} 일시정지`,exact:true})).toBeVisible();await card.getByRole('button',{name:/보드 열기/}).first().click();await expect(page.locator('article.widget-card').first()).toBeVisible();
 }finally{await request.post(`/api/v1/sources/${source.source.id}/pause`,{headers:{Origin:'http://127.0.0.1:8091'},data:{paused:false}})}
});
