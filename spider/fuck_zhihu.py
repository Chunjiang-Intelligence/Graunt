import asyncio
import json
import random
import re
import os
import time
from typing import List, Dict, Optional
from playwright.async_api import async_playwright, BrowserContext, Page, TimeoutError as PlaywrightTimeoutError

ACCOUNTS_CONFIG = [
    {
        "username": "user1", # 仅作标识用，文件名会用到
        # 浏览器 F12 -> Network -> Doc -> Headers -> Cookie 复制完整的字符串
        "cookie_str": "你的完整Cookie字符串_1", 
        # 代理格式: "http://user:pass@ip:port" 或 None (不使用代理)
        "proxy": None 
    },
    # 可以添加更多账户...
    # {
    #     "username": "user2",
    #     "cookie_str": "你的完整Cookie字符串_2",
    #     "proxy": "http://127.0.0.1:7890"
    # }
]

HEADLESS_MODE = True
SCROLL_COUNT_BEFORE_CLICK = 3
DATA_DIR = "zhihu_data"


class ZhihuDataCleaner:    
    @staticmethod
    def clean(text: str) -> str:
        if not text:
            return ""
        
        patterns = [
            r'著作权归作者所有。.*',
            r'商业转载请联系作者获得授权，非商业转载请注明出处。.*',
            r'作者：.*',
            r'链接：https://www.zhihu.com/.*',
            r'来源：知乎'
        ]
        
        lines = text.split('\n')
        cleaned_lines = []
        
        for line in reversed(lines):
            line = line.strip()
            if not line:
                continue
            
            is_copyright = False
            for p in patterns:
                if re.search(p, line):
                    is_copyright = True
                    break
            
            if is_copyright:
                continue
            else:
                pass

        clean_text = text
        suffix_pattern = r'(作者：.*[\r\n]+链接：.*[\r\n]+来源：知乎[\r\n]+著作权归作者所有.*)$'
        clean_text = re.sub(suffix_pattern, '', clean_text, flags=re.DOTALL)
        
        return clean_text.strip()

class ZhihuWorker:
    def __init__(self, config: Dict, playwright_instance):
        self.username = config['username']
        self.cookie_str = config['cookie_str']
        self.proxy_conf = config['proxy']
        self.p = playwright_instance
        self.context: Optional[BrowserContext] = None
        self.page: Optional[Page] = None
        
        if not os.path.exists(DATA_DIR):
            os.makedirs(DATA_DIR)

    def _parse_cookies(self, cookie_str: str) -> List[Dict]:
        cookies = []
        for item in cookie_str.split(';'):
            if '=' in item:
                name, value = item.strip().split('=', 1)
                cookies.append({
                    'name': name,
                    'value': value,
                    'domain': '.zhihu.com',
                    'path': '/'
                })
        return cookies

    async def init_browser(self):
        browser_args = [
            '--disable-blink-features=AutomationControlled',
            '--no-sandbox'
        ]
        
        browser = await self.p.chromium.launch(
            headless=HEADLESS_MODE,
            args=browser_args,
            proxy={"server": self.proxy_conf} if self.proxy_conf else None
        )
        
        self.context = await browser.new_context(
            user_agent="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
            viewport={'width': 1920, 'height': 1080},
            permissions=['clipboard-read', 'clipboard-write']
        )
        
        cookies = self._parse_cookies(self.cookie_str)
        await self.context.add_cookies(cookies)
        
        await self.context.add_init_script("""
            Object.defineProperty(navigator, 'webdriver', {
                get: () => undefined
            });
        """)
        
        self.page = await self.context.new_page()
        print(f"[{self.username}] Browser initialized.")

    async def random_sleep(self, min_s=1, max_s=3):
        await asyncio.sleep(random.uniform(min_s, max_s))

    async def scroll_feed(self):
        for _ in range(SCROLL_COUNT_BEFORE_CLICK):
            await self.page.keyboard.press("PageDown")
            await self.random_sleep(1, 2)
        print(f"[{self.username}] Scrolled feed.")

    async def copy_content_via_clipboard(self, target_page: Page) -> str:
        try:
            await target_page.wait_for_load_state("domcontentloaded")
            await self.random_sleep(2, 4)

            await target_page.mouse.click(500, 500)
            
            modifier = "Meta" if "Mac" in await target_page.evaluate("navigator.platform") else "Control"
            
            print(f"[{self.username}] Simulating {modifier}+A (Select All)...")
            await target_page.keyboard.press(f"{modifier}+A")
            await asyncio.sleep(0.5)
            
            print(f"[{self.username}] Simulating {modifier}+C (Copy)...")
            await target_page.keyboard.press(f"{modifier}+C")
            await asyncio.sleep(1)
            content = await target_page.evaluate("navigator.clipboard.readText()")
            
            return content
        except Exception as e:
            print(f"[{self.username}] Clipboard operation failed: {e}")
            return ""

    async def process_detail_page(self, url: str):
        new_page = await self.context.new_page()
        try:
            print(f"[{self.username}] Navigating to post: {url}")
            await new_page.goto(url)
            
            raw_text = await self.copy_content_via_clipboard(new_page)
            
            if not raw_text or len(raw_text) < 10:
                print(f"[{self.username}] Warning: Copied text is empty or too short.")
                return

            cleaned_text = ZhihuDataCleaner.clean(raw_text)
            
            try:
                title = await new_page.title()
                clean_title = re.sub(r'[\\/*?:"<>|]', "", title).strip()[:50]
            except:
                clean_title = f"unknown_{int(time.time())}"

            filename = f"{self.username}_{clean_title}_{int(time.time())}.txt"
            filepath = os.path.join(DATA_DIR, filename)
            
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(f"URL: {url}\n")
                f.write(f"Title: {title}\n")
                f.write("-" * 20 + "\n")
                f.write(cleaned_text)
            
            print(f"[{self.username}] Data saved to {filepath} (Size: {len(cleaned_text)} chars)")

        except Exception as e:
            print(f"[{self.username}] Error processing detail page: {e}")
        finally:
            await new_page.close()

    async def run(self):
        await self.init_browser()
        
        try:
            print(f"[{self.username}] Going to Zhihu Home...")
            await self.page.goto("https://www.zhihu.com", wait_until="networkidle")
            
            if "登录" in await self.page.title():
                print(f"[{self.username}] Error: Cookie invalid or expired. Need manual login.")
                return

            while True:
                await self.scroll_feed()
                
                post_locators = self.page.locator(".TopstoryItem .ContentItem-title a[href*='/question/'], .TopstoryItem .ContentItem-title a[href*='/p/']")
                count = await post_locators.count()
                
                print(f"[{self.username}] Found {count} potential posts in view.")
                
                if count > 0:
                    index = random.randint(0, min(count, 5) - 1)
                    target_locator = post_locators.nth(index)
                    
                    url = await target_locator.get_attribute("href")
                    
                    if url and not url.startswith("http"):
                        url = "https:" + url if url.startswith("//") else "https://www.zhihu.com" + url
                    
                    if url:
                        await self.process_detail_page(url)
                
                sleep_time = random.uniform(3, 6)
                print(f"[{self.username}] Sleeping {sleep_time:.2f}s before next loop...")
                await asyncio.sleep(sleep_time)

        except Exception as e:
            print(f"[{self.username}] Critical Loop Error: {e}")
        finally:
            if self.context:
                await self.context.close()

async def main():
    print("=== Zhihu Crawler via Clipboard Simulation Started ===")
    
    async with async_playwright() as playwright:
        tasks = []
        for account_cfg in ACCOUNTS_CONFIG:
            if not account_cfg['cookie_str']:
                print(f"Skipping {account_cfg['username']}: No cookie provided.")
                continue
                
            worker = ZhihuWorker(account_cfg, playwright)
            tasks.append(worker.run())
        
        if not tasks:
            print("No valid accounts configured.")
            return

        await asyncio.gather(*tasks)

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nCrawler stopped by user.")