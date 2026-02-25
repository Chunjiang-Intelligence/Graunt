import asyncio
import httpx
import random
import json
import logging
import re
import os
from typing import List

CONCURRENCY = 30           # 详情页最大并发请求数 (按需调整)
MAX_PAGES_TO_FETCH = 20    # 首页信息流要拉取的总页数 (每页20条)
GIDS = 2                   # 米游社游戏分区 ID (2代表原神)

PROXIES = [
    # "http://user:pass@111.222.111.222:8080",
    # "http://user:pass@333.444.333.444:8080",
    None 
]

USER_AGENTS = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/120.0"
]

OUTPUT_FILE = "miyoushe_articles.jsonl"

logging.basicConfig(level=logging.INFO, format='%(asctime)s | %(levelname)s | %(message)s')

def get_random_headers():
    return {
        "User-Agent": random.choice(USER_AGENTS),
        "Accept": "application/json, text/plain, */*",
        "Origin": "https://www.miyoushe.com",
        "Referer": "https://www.miyoushe.com/"
    }

def get_proxy():
    p = random.choice(PROXIES)
    return p if p else None

async def fetch_post_ids_from_page(page: int) -> List[dict]:
    api_url = f"https://bbs-api-static.miyoushe.com/apihub/wapi/webHome?gids={GIDS}&page={page}&page_size=20"
    
    proxy = get_proxy()
    async with httpx.AsyncClient(proxies=proxy, verify=False) as client:
        try:
            resp = await client.get(api_url, headers=get_random_headers(), timeout=10)
            resp.raise_for_status()
            data = resp.json()
            
            if data.get("retcode") != 0:
                logging.error(f"第 {page} 页接口返回错误: {data.get('message')}")
                return []
                
            posts_data = data.get("data", {}).get("recommended_posts", [])
            extracted_posts = []
            
            for item in posts_data:
                post = item.get("post", {})
                post_id = post.get("post_id")
                subject = post.get("subject")
                if post_id and subject:
                    extracted_posts.append({
                        "post_id": post_id,
                        "title": subject,
                        "url": f"https://www.miyoushe.com/ys/article/{post_id}"
                    })
                    
            logging.info(f"成功抓取第 {page} 页，提取到 {len(extracted_posts)} 个帖子ID")
            return extracted_posts
            
        except Exception as e:
            logging.error(f"抓取第 {page} 页失败: {e}")
            return []


async def fetch_article_detail(post_info: dict):
    post_id = post_info["post_id"]
    url = post_info["url"]
    
    detail_api = f"https://bbs-api-static.miyoushe.com/post/wapi/getPostFull?gids={GIDS}&post_id={post_id}&read=1"
    
    proxy = get_proxy()
    async with httpx.AsyncClient(proxies=proxy, verify=False) as client:
        try:
            resp = await client.get(detail_api, headers=get_random_headers(), timeout=10)
            resp.raise_for_status()
            data = resp.json()
            
            if data.get("retcode") == 0:
                post_detail = data.get("data", {}).get("post", {}).get("post", {})
                raw_content = post_detail.get("content", "")
                
                clean_content = re.sub(r'<[^>]+>', '', raw_content).strip()
                
                post_info["content"] = clean_content[:2048]
                logging.success(f"抓取正文成功: {post_info['title'][:15]}... ({len(clean_content)} 字)")
                return post_info
            else:
                logging.warning(f"详情接口返回异常: {data.get('message')} (post_id: {post_id})")
                return None
                
        except Exception as e:
            logging.debug(f"抓取正文失败 (post_id: {post_id}): {e}")
            return None

async def worker(queue: asyncio.Queue, file_obj):
    while True:
        post_info = await queue.get()
        if post_info is None:
            break
            
        result = await fetch_article_detail(post_info)
        
        if result and result.get("content"):
            file_obj.write(json.dumps(result, ensure_ascii=False) + "\n")
            file_obj.flush()
            
        queue.task_done()
        await asyncio.sleep(random.uniform(0.5, 2.0))

async def main():
    logger = logging.getLogger()    
    task_queue = asyncio.Queue()
    
    logger.info(f"正在拉取前 {MAX_PAGES_TO_FETCH} 页的帖子列表...")
    
    fetch_tasks = [fetch_post_ids_from_page(p) for p in range(1, MAX_PAGES_TO_FETCH + 1)]
    pages_results = await asyncio.gather(*fetch_tasks)
    
    total_posts = 0
    for page_items in pages_results:
        for item in page_items:
            await task_queue.put(item)
            total_posts += 1
            
    logger.info(f"列表抓取完成，共提取 {total_posts} 个待爬取文章任务放入队列。")
    
    f = open(OUTPUT_FILE, "a", encoding="utf-8")
    
    workers = [asyncio.create_task(worker(task_queue, f)) for _ in range(CONCURRENCY)]
    
    await task_queue.join()
    
    for _ in range(CONCURRENCY):
        await task_queue.put(None)
    await asyncio.gather(*workers)
    
    f.close()
    logger.info(f"所有任务完成，数据已保存至 {OUTPUT_FILE}")

if __name__ == "__main__":
    if os.name == 'nt':
        asyncio.set_event_loop_policy(asyncio.WindowsSelectorEventLoopPolicy())
        
    asyncio.run(main())