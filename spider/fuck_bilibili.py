import asyncio
import aiohttp
import aiofiles
import hashlib
import time
import urllib.parse
import json
import random
from functools import reduce

START_AV = 1
END_AV = 1000000
CONCURRENCY = 50  # 并发数
OUTPUT_FILE = "bilibili_comments.jsonl"

mixinKeyEncTab = [
    46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49,
    33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40,
    61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11,
    36, 20, 34, 44, 52
]

def getMixinKey(orig: str):
    """对 imgKey 和 subKey 进行字符顺序打乱编码"""
    return reduce(lambda s, i: s + orig[i], mixinKeyEncTab, "")[:32]

def encWbi(params: dict, img_key: str, sub_key: str):
    """为请求参数进行 WBI 签名"""
    mixin_key = getMixinKey(img_key + sub_key)
    curr_time = round(time.time())
    params['wts'] = curr_time  # 添加时间戳
    # 按照 key 重排参数
    params = dict(sorted(params.items()))
    # 过滤不用签名的字符
    query = urllib.parse.urlencode({
        k: ''.join(filter(lambda x: x not in "!'()*", str(v)))
        for k, v in params.items()
    })
    # 计算 w_rid
    w_rid = hashlib.md5((query + mixin_key).encode(encoding='utf-8')).hexdigest()
    params['w_rid'] = w_rid
    return params

class BiliSpider:
    def __init__(self):
        self.headers = {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
            'Referer': 'https://www.bilibili.com/',
        }
        self.img_key = None
        self.sub_key = None
        self.semaphore = asyncio.Semaphore(CONCURRENCY)

    async def get_proxy(self):
        """
        这里接入你的代理池
        返回格式: 'http://user:pass@ip:port' 或 None (直连)
        """
        # 示例：return "http://127.0.0.1:7890"
        return None 

    async def init_wbi_keys(self, session):
        """
        获取最新的 WBI 密钥。
        这些密钥位于 nav 接口中，虽然我们不登录，但需要这俩 key 来做签名。
        """
        try:
            url = 'https://api.bilibili.com/x/web-interface/nav'
            async with session.get(url, headers=self.headers, ssl=False) as resp:
                data = await resp.json()
                wbi_img = data['data']['wbi_img']
                self.img_key = wbi_img['img_url'].split("/")[-1].split(".")[0]
                self.sub_key = wbi_img['sub_url'].split("/")[-1].split(".")[0]
                print(f"WBI Key 初始化成功: {self.img_key[:5]}... / {self.sub_key[:5]}...")
                return True
        except Exception as e:
            print(f"WBI Key 初始化失败: {e}")
            return False

    async def fetch_comment(self, session, aid):
        """
        爬取单个视频的评论区 (仅第一页热门评论，为了性能)
        av号即 oid
        """
        if not self.img_key:
            print("缺少 WBI Key，跳过")
            return

        params = {
            'oid': aid,       # 视频的 AV 号
            'type': 1,        # 1 代表视频
            'mode': 2,        # 3 代表热度排序 (2代表时间排序)
            'pagination_str': '{"offset":""}',
            'plat': 1,
            'web_location': 1315875
        }
        
        # 进行 WBI 签名计算 w_rid 和 wts
        signed_params = encWbi(params, self.img_key, self.sub_key)
        
        target_url = 'https://api.bilibili.com/x/v2/reply/wbi/main'
        
        async with self.semaphore: # 限制并发
            try:
                proxy = await self.get_proxy()
                async with session.get(target_url, params=signed_params, headers=self.headers, proxy=proxy, ssl=False, timeout=10) as resp:
                    if resp.status != 200:
                        print(f"Http Error {aid}: {resp.status}")
                        return None
                    
                    content = await resp.json()
                    
                    # 检查 B 站 API 返回码
                    code = content.get('code')
                    if code == 0:
                        # 成功获取数据
                        replies = content['data'].get('replies', [])
                        top = content['data'].get('top', {})
                        
                        result_data = {
                            "av": aid,
                            "count": content['data']['cursor']['all_count'],
                            "top_comment": top.get('upper', {}).get('content', {}).get('message', '') if top and top.get('upper') else None,
                            "hot_comments": [r['content']['message'] for r in replies if r] if replies else []
                        }
                        
                        print(f"AV{aid} 抓取成功, 评论数: {result_data['count']}")
                        return result_data
                    
                    elif code == 12002:
                        pass
                    elif code == -404:
                        pass
                    elif code == -412:
                        print(f"AV{aid} 触发风控 (412)！请更换代理！")
                    else:
                        print(f"AV{aid} API错误: {code} - {content.get('message')}")
                        
            except Exception as e:
                pass
        return None

    async def writer(self, queue):
        async with aiofiles.open(OUTPUT_FILE, mode='a', encoding='utf-8') as f:
            while True:
                item = await queue.get()
                if item is None:
                    break
                await f.write(json.dumps(item, ensure_ascii=False) + "\n")
                queue.task_done()

    async def main(self):
        async with aiohttp.ClientSession() as session:
            if not await self.init_wbi_keys(session):
                return

            queue = asyncio.Queue()
            writer_task = asyncio.create_task(self.writer(queue))

            tasks = []
            batch_size = 1000 
            
            for i in range(START_AV, END_AV + 1):
                task = asyncio.create_task(self.fetch_comment(session, i))
                tasks.append(task)
                
                if len(tasks) >= batch_size:
                    results = await asyncio.gather(*tasks)
                    for res in results:
                        if res:
                            await queue.put(res)
                    tasks = []
                    
                    if i % 10000 == 0: await self.init_wbi_keys(session)

            if tasks:
                results = await asyncio.gather(*tasks)
                for res in results:
                    if res:
                        await queue.put(res)

            await queue.put(None)
            await writer_task
            print("所有任务完成")

if __name__ == '__main__':
    spider = BiliSpider()
    try:
        import uvloop
        asyncio.set_event_loop_policy(uvloop.EventLoopPolicy())
    except ImportError:
        pass
        
    asyncio.run(spider.main())