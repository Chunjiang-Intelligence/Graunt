import asyncio
import uvloop
import aiohttp
import redis.asyncio as redis
import orjson
import logging
from fake_useragent import UserAgent
from urllib.parse import urljoin

asyncio.set_event_loop_policy(uvloop.EventLoopPolicy())

REDIS_URL = "redis://127.0.0.1:6379"
REDIS_QUEUE_KEY = "spider:target_urls"
REDIS_SEEN_KEY = "spider:seen_reddit_ids"
CONCURRENCY = 64
USER_AGENT = UserAgent().random
ENTRY_POINT = "https://www.reddit.com/r/all.json?limit=100"

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger("reddit_reaper")

class RedditScanner:
    def __init__(self):
        self.redis = redis.from_url(REDIS_URL, decode_responses=True)
        self.queue = asyncio.Queue()
        self.seen_cache = set()
        
    async def push_to_redis(self, urls):
        if not urls: return
        async with self.redis.pipeline() as pipe:
            for url in urls:
                pipe.lpush(REDIS_QUEUE_KEY, url)
            await pipe.execute()
        logger.info(f"注入 {len(urls)} 个 reddit 帖子到 Rust 队列")

    async def fetch_json(self, session, url):
        try:
            async with session.get(url, timeout=10) as response:
                if response.status == 200:
                    return orjson.loads(await response.read())
                elif response.status == 429:
                    logger.warning("被 Reddit 限流，休眠 2 秒...")
                    await asyncio.sleep(2)
                else:
                    logger.debug(f"HTTP {response.status}: {url}")
        except Exception as e:
            pass
        return None

    async def worker(self, name, session):
        while True:
            url = await self.queue.get()
            data = await self.fetch_json(session, url)
            
            new_target_urls = []
            next_page_url = None

            if data and isinstance(data, dict):
                if 'data' in data and 'children' in data['data']:
                    children = data['data']['children']
                    after = data['data'].get('after')
                    if after:
                        base = url.split('?')[0]
                        next_page_url = f"{base}?after={after}&limit=100"
                    async with self.redis.pipeline() as pipe:
                        for child in children:
                            kind = child.get('kind')
                            cdata = child.get('data', {})
                            permalink = cdata.get('permalink')
                            
                            if kind == 't3' and permalink:
                                full_json_url = f"https://www.reddit.com{permalink}.json"
                                pipe.sadd(REDIS_SEEN_KEY, cdata.get('id'))
                                new_target_urls.append(full_json_url)
                        
                        results = await pipe.execute()
                    
                    final_urls = [url for url, is_new in zip(new_target_urls, results) if is_new]
                    
                    if final_urls:
                        await self.push_to_redis(final_urls)

            if next_page_url:
                await self.queue.put(next_page_url)
            
            self.queue.task_done()

    async def run(self):
        headers = {
            "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
            "Accept": "application/json"
        }
        
        timeout = aiohttp.ClientTimeout(total=15)
        connector = aiohttp.TCPConnector(limit=CONCURRENCY * 2, ssl=False, ttl_dns_cache=300)
        
        async with aiohttp.ClientSession(connector=connector, headers=headers, timeout=timeout) as session:
            await self.queue.put(ENTRY_POINT)
            workers = [asyncio.create_task(self.worker(f"w-{i}", session)) for i in range(CONCURRENCY)]
            while True:
                qsize = self.queue.qsize()
                if qsize == 0 and len(workers) > 0:
                     logger.warning("队列空闲...")
                await asyncio.sleep(5)

if __name__ == "__main__":
    scanner = RedditScanner()
    try:
        asyncio.run(scanner.run())
    except KeyboardInterrupt:
        pass