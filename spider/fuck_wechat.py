import asyncio
import httpx
import re
import random
import json
import logging
import time
from typing import List
from openai import AsyncOpenAI

OPENAI_API_KEY = "sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
OPENAI_BASE_URL = "https://api.openai.com/v1" 
LLM_MODEL = "gpt-5"
CONCURRENCY = 50
FETCH_PAGES_PER_WORD = 1

PROXIES = [
    None  # 填入你的代理 IP，例如 "http://127.0.0.1:7890"
]

USER_AGENTS = [
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"
]

OUTPUT_FILE = "wechat_realtime_articles.txt"

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s | %(levelname)s | %(message)s',
    datefmt='%H:%M:%S'
)
client_llm = AsyncOpenAI(api_key=OPENAI_API_KEY, base_url=OPENAI_BASE_URL)

def get_headers():
    return {
        "User-Agent": random.choice(USER_AGENTS),
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
        "Referer": "https://weixin.sogou.com/",
        "Connection": "keep-alive"
    }

def get_proxy():
    p = random.choice(PROXIES)
    return p if p else None

async def fetch_realtime_hotwords_agent() -> List[str]:
    logging.info("联网搜索最新热点...")

    extract_tool = {
        "type": "function",
        "function": {
            "name": "extract_keywords",
            "description": "从新闻文本中提取热门搜索关键词",
            "parameters": {
                "type": "object",
                "properties": {
                    "keywords": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "关键词列表，如 ['iPhone 16发布', '亚运会金牌', '华为Mate60']"
                    }
                },
                "required": ["keywords"]
            }
        }
    }

    try:
        response = await client_llm.chat.completions.create(
            model=LLM_MODEL,
            messages=[{"role": "user", "content": "现在微博、知乎、百度热搜的前10名是什么？请只列出关键词。"}],
            extra_body={"tools": [{"type": "web_search"}]} # 假设中转商支持此字段
        )

        if response.choices[0].message.tool_calls:
            tool_call = response.choices[0].message.tool_calls[0]
            args = json.loads(tool_call.function.arguments)
            keywords = args.get("keywords", [])
            logging.info(f"捕获实时热词: {keywords}")
            return keywords
        else:
            content = response.choices[0].message.content
            logging.warning("LLM 未调用工具，降级解析。")
            return [line.strip() for line in content.split('\n') if len(line) < 10][:10]

    except Exception as e:
        logging.error(f"Agent 运行失败: {e}")
        return ["人工智能", "Python", "新能源汽车", "MyGO", "金融", "超时空辉夜姬"]

async def search_sogou(client: httpx.AsyncClient, keyword: str, page: int):
    base_url = "https://weixin.sogou.com/weixin"
    params = {
        "type": "2",
        "s_from": "input",
        "query": keyword,
        "ie": "utf8",
        "page": page
    }
    
    try:
        resp = await client.get(base_url, params=params, headers=get_headers(), timeout=10)
        
        if "验证码" in resp.text or "antispider" in str(resp.url):
            logging.warning(f"关键词 [{keyword}] 触发搜狗验证码")
            return []

        raw_links = re.findall(r'href="(/link\?url=[^"]+)"', resp.text)
        full_links = [
            f"https://weixin.sogou.com{link}".replace("&amp;", "&") + "&k=1&h=1" 
            for link in raw_links
        ]
        
        return list(set(full_links))

    except Exception as e:
        logging.error(f"搜索请求异常: {e}")
        return []

async def fetch_article_content(client: httpx.AsyncClient, link: str):
    try:
        # 1. 请求搜狗跳转页
        # follow_redirects=True 会自动处理 302 跳转
        # 如果是 JS 跳转，响应体里会有 url += '...'
        resp = await client.get(link, headers=get_headers(), follow_redirects=True, timeout=15)
        
        html = resp.text
        real_url = str(resp.url)
        
        # 处理搜狗的 JS 拼接跳转 (meta content 里的 url 或者 script 里的 url)
        if "weixin.sogou.com" in real_url:
            # 尝试提取 js 中的 url 拼接
            # 典型特征: url += 'http://mp.weixin.qq.com/s?src=...'
            fragments = re.findall(r"url \+= '([^']+)';", html)
            if fragments:
                real_url_fragment = "".join(fragments)
                # 简单清洗
                if "http" not in real_url_fragment:
                    # 有时候是相对路径或者缺协议头，这里简单处理，大不了丢弃
                    return None
                real_url = real_url_fragment.replace("@", "") # 有时候会有垃圾字符
                
                # 再次请求真实微信链接
                resp = await client.get(real_url, headers=get_headers(), follow_redirects=True, timeout=15)
                html = resp.text

        # 2. 验证是否为微信文章页
        if "var msg_title" not in html and "rich_media" not in html:
            return None

        # 3. 提取数据
        title_match = re.search(r'var msg_title = "([^"]+)";', html)
        title = title_match.group(1) if title_match else "无标题"
        
        # 微信文章正文通常在 id="js_content" 的 div 里，但直接正则去标签更通用
        # 先把 <script> 和 <style> 去掉，防止提取到 js 代码里的中文
        clean_html = re.sub(r'<(script|style).*?</\1>', '', html, flags=re.DOTALL)
        text_only = re.sub(r'<[^>]+>', '', clean_html)
        
        # 提取中文字符串 (保留标点以保持可读性)
        # 过滤掉过短的行（导航栏、广告）
        lines = [line.strip() for line in text_only.split('\n') if len(line.strip()) > 10]
        content = "\n".join(lines)
        
        if len(content) < 50:
            return None
            
        logging.info(f"抓取成功: {title} ({len(content)}字)")
        
        return {
            "title": title,
            "url": real_url,
            "content": content
        }

    except Exception as e:
        return None


async def worker(link_queue: asyncio.Queue, file_obj):
    while True:
        link = await link_queue.get()
        if link is None:
            break
        
        proxy = get_proxy()
        async with httpx.AsyncClient(proxies=proxy, verify=False) as client:
            article_data = await fetch_article_content(client, link)
            
            if article_data:
                record = f"Title: {article_data['title']}\nURL: {article_data['url']}\nContent:\n{article_data['content'][:200]}...\n{'-'*60}\n"
                file_obj.write(record)
                file_obj.flush()
        
        link_queue.task_done()

async def main():
    hot_words = await fetch_realtime_hotwords_agent()
    if not hot_words:
        logging.error("没有获取到热词，退出。")
        return

    logging.info(f"开始根据热词进行采集，目标词数: {len(hot_words)}")
    
    link_queue = asyncio.Queue()
    
    async def search_task(word):
        proxy = get_proxy()
        async with httpx.AsyncClient(proxies=proxy, verify=False) as client:
            for page in range(1, FETCH_PAGES_PER_WORD + 1):
                links = await search_sogou(client, word, page)
                for l in links:
                    await link_queue.put(l)
                await asyncio.sleep(random.uniform(1, 2))

    search_tasks = [search_task(w) for w in hot_words]
    await asyncio.gather(*search_tasks)
    
    logging.info(f"搜索结束，待抓取文章总数: {link_queue.qsize()}")

    # 采集 Worker
    f = open(OUTPUT_FILE, "a", encoding="utf-8")
    workers = [asyncio.create_task(worker(link_queue, f)) for _ in range(CONCURRENCY)]
    
    await link_queue.join()
    
    # 停止 Workers
    for _ in range(CONCURRENCY):
        await link_queue.put(None)
    await asyncio.gather(*workers)
    
    f.close()
    logging.info("全部任务完成。")

if __name__ == "__main__":
    asyncio.run(main())