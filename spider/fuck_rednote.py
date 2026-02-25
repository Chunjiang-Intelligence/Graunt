import asyncio
import json
import logging
import random
import os
from typing import List, Dict, Set

from playwright.async_api import async_playwright, BrowserContext, Page, Error as PlaywrightError


# 账号与代理池配置，小红书风控极严，大量爬取必须提供多个高可用代理和不同账号的 Cookie
ACCOUNTS_CONFIG = [
    {
        "name": "Account_1_Feeder", # 建议把第一个账号作为发现，用来刷首页拿ID
        "cookie": "你的Cookie_1", # F12 -> Network -> Doc/Fetch -> 复制 Cookie
        "proxy": "http://user1:pass1@192.168.1.101:8080" # 代理配置，无代理填 None
    },
    {
        "name": "Account_2_Worker",
        "cookie": "你的Cookie_2",
        "proxy": "http://user2:pass2@192.168.1.102:8080"
    },
    # {
    #     "name": "Account_3_Worker",
    #     "cookie": "你的Cookie_3",
    #     "proxy": "http://user3:pass3@192.168.1.103:8080"
    # }
]

HEADLESS_MODE = True            # 是否开启无头模式
MAX_SCROLL_PAGES = 5            # 详情页评论区最多滚动的次数 (控制单篇笔记爬取深度)
OUTPUT_FILE = "xhs_data.jsonl"  # 结果保存路径 (JSONL格式，每行一个完整笔记+评论)


logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s | %(levelname)s | %(message)s',
    datefmt='%H:%M:%S'
)
logger = logging.getLogger(__name__)

# 任务队列与去重集合
task_queue = asyncio.Queue(maxsize=2000)
crawled_notes: Set[str] = set()
lock = asyncio.Lock()


class BrowserManager:
    def __init__(self, config: dict, playwright):
        self.config = config
        self.name = config['name']
        self.cookie_str = config['cookie']
        self.proxy_conf = config['proxy']
        self.p = playwright
        self.browser = None
        self.context: BrowserContext = None

    def _parse_cookies(self) -> List[dict]:
        cookies = []
        if not self.cookie_str:
            return cookies
        for item in self.cookie_str.split(';'):
            if '=' in item:
                k, v = item.strip().split('=', 1)
                cookies.append({
                    'name': k, 'value': v,
                    'domain': '.xiaohongshu.com', 'path': '/'
                })
        return cookies

    async def start(self):
        proxy_settings = {"server": self.proxy_conf} if self.proxy_conf else None
        self.browser = await self.p.chromium.launch(
            headless=HEADLESS_MODE,
            proxy=proxy_settings,
            args=[
                '--disable-blink-features=AutomationControlled',
                '--no-sandbox', '--disable-infobars'
            ]
        )
        self.context = await self.browser.new_context(
            user_agent="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
            viewport={'width': 1280, 'height': 800}
        )
        await self.context.add_cookies(self._parse_cookies())
        
        await self.context.add_init_script("""
            Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
            setInterval(() => {
                const closeBtn = document.querySelector('.login-container .icon-btn-wrapper');
                if (closeBtn) closeBtn.click();
            }, 2000);
        """)
        logger.info(f"[{self.name}] 浏览器启动成功 | 代理: {self.proxy_conf}")

    async def close(self):
        if self.context: await self.context.close()
        if self.browser: await self.browser.close()



async def discover_engine(manager: BrowserManager):
    page = await manager.context.new_page()
    
    async def handle_homefeed(response):
        if "/api/sns/web/v1/homefeed" in response.url and response.status == 200:
            try:
                data = await response.json()
                items = data.get("data", {}).get("items", [])
                new_count = 0
                for item in items:
                    note_id = item.get("id")
                    if note_id and note_id not in crawled_notes:
                        crawled_notes.add(note_id)
                        await task_queue.put(note_id)
                        new_count += 1
                logger.info(f"[{manager.name}] 发现引擎捕获新笔记: {new_count} 个 | 队列当前: {task_queue.qsize()}")
            except Exception as e:
                logger.error(f"[{manager.name}] 解析Homefeed失败: {e}")

    page.on("response", handle_homefeed)

    try:
        logger.info(f"[{manager.name}] 启动发现 -> 访问首页...")
        await page.goto("https://www.xiaohongshu.com/explore", wait_until="networkidle", timeout=60000)
        
        while True:
            scroll_y = random.randint(800, 1200)
            await page.mouse.wheel(0, scroll_y)
            
            await asyncio.sleep(random.uniform(2, 4))
            
            if task_queue.qsize() > 1000:
                logger.info(f"[{manager.name}] 队列积压，暂停发现引擎 10 秒...")
                await asyncio.sleep(10)
            
            if "验证" in await page.title():
                logger.warning(f"[{manager.name}] 发现引擎触发验证码，等待人工处理或自动刷新...")
                await asyncio.sleep(20)
                await page.reload()

    except Exception as e:
        logger.error(f"[{manager.name}] 发现引擎崩溃: {e}")
    finally:
        await page.close()


async def worker_engine(manager: BrowserManager):
    """
    从队列获取 note_id，进入详情页，利用 SSR 提取正文，拦截 API 提取评论
    """
    page = await manager.context.new_page()
    
    while True:
        note_id = await task_queue.get()
        logger.info(f"[{manager.name}] 开始处理任务: {note_id}")

        crawled_data = {
            "note_id": note_id,
            "detail": {},
            "comments": []
        }
        
        # 拦截评论接口 (v2/comment/page 和 sub/page)
        async def handle_comments(response):
            if ("/api/sns/web/v2/comment/page" in response.url or "/api/sns/web/v2/comment/sub/page" in response.url) \
               and response.status == 200:
                try:
                    res_json = await response.json()
                    comments = res_json.get("data", {}).get("comments", [])
                    if comments:
                        crawled_data["comments"].extend(comments)
                        logger.debug(f"[{manager.name}] 截获评论: {len(comments)} 条")
                except:
                    pass

        page.on("response", handle_comments)

        try:
            url = f"https://www.xiaohongshu.com/discovery/item/{note_id}"
            await page.goto(url, wait_until="domcontentloaded", timeout=30000)

            # 提取 SSR 注入的笔记详情
            detail_data = await page.evaluate("""() => {
                return window.__INITIAL_STATE__ && window.__INITIAL_STATE__.note && window.__INITIAL_STATE__.note.noteDetailMap ? 
                       window.__INITIAL_STATE__.note.noteDetailMap[Object.keys(window.__INITIAL_STATE__.note.noteDetailMap)[0]] : null;
            }""")

            if detail_data:
                crawled_data["detail"] = detail_data
                logger.info(f"[{manager.name}] 成功提取笔记详情: {detail_data.get('title', '无标题')[:20]}...")
            else:
                logger.warning(f"[{manager.name}] SSR数据提取失败 (可能被风控或页面结构变更)")

            prev_height = 0
            for _ in range(MAX_SCROLL_PAGES):
                await page.evaluate("window.scrollBy(0, document.body.scrollHeight)")
                await asyncio.sleep(random.uniform(1.0, 2.5))
                
                curr_height = await page.evaluate("document.body.scrollHeight")
                if curr_height == prev_height:
                    break # 到底了
                prev_height = curr_height
            
            if crawled_data["detail"] or crawled_data["comments"]:
                async with lock:
                    with open(OUTPUT_FILE, "a", encoding="utf-8") as f:
                        f.write(json.dumps(crawled_data, ensure_ascii=False) + "\n")
                logger.info(f"[{manager.name}] 数据落盘完毕 | 评论数: {len(crawled_data['comments'])}")

        except PlaywrightError as e:
            logger.error(f"[{manager.name}] 页面加载/操作超时: {e}")
        except Exception as e:
            logger.error(f"[{manager.name}] 未知错误: {e}")
        finally:
            page.remove_listener("response", handle_comments)
            task_queue.task_done()
            
            await asyncio.sleep(random.uniform(2, 5))



async def main():
    async with async_playwright() as p:
        managers = []
        
        for acc_cfg in ACCOUNTS_CONFIG:
            mgr = BrowserManager(acc_cfg, p)
            await mgr.start()
            managers.append(mgr)
        
        if not managers:
            logger.error("未配置有效账号，退出")
            return

        tasks = []
        producer_mgr = managers[0]
        consumer_mgrs = managers[1:] if len(managers) > 1 else [managers[0]]
        
        tasks.append(asyncio.create_task(discover_engine(producer_mgr)))
        
        for mgr in consumer_mgrs:
            tasks.append(asyncio.create_task(worker_engine(mgr)))

        try:
            await asyncio.gather(*tasks)
        except KeyboardInterrupt:
            logger.info("用户停止爬虫...")
        finally:
            logger.info("正在关闭浏览器资源...")
            for mgr in managers:
                await mgr.close()

if __name__ == "__main__":
    if os.name == 'nt':
        asyncio.set_event_loop_policy(asyncio.WindowsSelectorEventLoopPolicy())
    
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        pass