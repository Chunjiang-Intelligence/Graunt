import asyncio
import uvloop
import redis.asyncio as redis
import logging

asyncio.set_event_loop_policy(uvloop.EventLoopPolicy())

REDIS_URL = "redis://127.0.0.1:6379"
REDIS_QUEUE_KEY = "spider:target_urls"
START_ID = 1
END_ID = 20_000_000
BATCH_SIZE = 10_000
# 请使用带有cookie的私有镜像
BASE_URL_TEMPLATE = "https://z-library.sk/dl/{}" 

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(message)s')
logger = logging.getLogger("zlib_pusher")

async def main():
    r = redis.from_url(REDIS_URL, decode_responses=True)
    logger.info(f"Z-Library URL: ID {START_ID} -> {END_ID}")

    pipeline = r.pipeline()
    count = 0
    total_pushed = 0

    for book_id in range(START_ID, END_ID + 1):
        url = BASE_URL_TEMPLATE.format(book_id)
        pipeline.lpush(REDIS_QUEUE_KEY, url)
        count += 1
        if count >= BATCH_SIZE:
            await pipeline.execute()
            total_pushed += count
            logger.info(f"已注入: {total_pushed} 条 (当前 ID: {book_id})")
            count = 0

    if count > 0:
        await pipeline.execute()
        total_pushed += count

    logger.info(f"共注入 {total_pushed} 条任务。")
    await r.aclose()

if __name__ == "__main__":
    asyncio.run(main())