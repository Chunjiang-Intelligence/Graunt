use anyhow::{Context, Result};
use base64::{engine::general_purpose, Engine as _};
use bytes::Bytes;
use futures::{StreamExt, TryStreamExt};
use governor::{Quota, RateLimiter};
use governor::state::direct::NotKeyed;
use governor::clock::DefaultClock;
use governor::state::InMemoryState;
use nonzero_ext::nonzero;
use redis::AsyncCommands;
use reqwest::{Client, Proxy};
use std::env;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::mpsc;
use url::Url;

#[global_allocator]
static GLOBAL: jemallocator::Jemalloc = jemallocator::Jemalloc;

/*************************************************************
                        配置区域开始
*************************************************************/
const REDIS_URL: &str = "redis://127.0.0.1:6379/";
const REDIS_QUEUE_KEY: &str = "spider:target_urls";
const WEBDAV_ENDPOINT: &str = "http://192.168.1.100:5005/remote.php/dav/files/user/";
const WEBDAV_USER: &str = "admin";
const WEBDAV_PASS: &str = "password";
const SOCKS5_PROXY: Option<&str> = Some("socks5://127.0.0.1:1080");
const MAX_CONCURRENCY: usize = 256;
const RATE_LIMIT_PER_SEC: u32 = 1024;
const TIMEOUT_SECS: u64 = 30;
const REDIS_BATCH_SIZE: isize = 100;
/*************************************************************
                        配置区域结束
*************************************************************/

struct AppState {
    http_client: Client,
    webdav_client: Client,
    webdav_auth_header: String,
    limiter: RateLimiter<NotKeyed, InMemoryState, DefaultClock>,
}

fn main() -> Result<()> {
    tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .worker_threads(num_cpus::get() * 2)
        .build()
        .unwrap()
        .block_on(async_main())
}

async fn async_main() -> Result<()> {
    env_logger::init_from_env(env_logger::Env::default().default_filter_or("info"));
    log::info!("spider man!");

    let auth_raw = format!("{}:{}", WEBDAV_USER, WEBDAV_PASS);
    let webdav_auth_header = format!("Basic {}", general_purpose::STANDARD.encode(auth_raw));
    let mut http_builder = Client::builder()
        .tcp_nodelay(true)
        .tcp_keepalive(Duration::from_secs(60))
        .timeout(Duration::from_secs(TIMEOUT_SECS))
        .pool_idle_timeout(Duration::from_secs(90))
        .pool_max_idle_per_host(MAX_CONCURRENCY * 2)
        .danger_accept_invalid_certs(true);

    if let Some(proxy_url) = SOCKS5_PROXY {
        let proxy = Proxy::all(proxy_url).context("SOCKS5 Proxy Invalid")?;
        http_builder = http_builder.proxy(proxy);
        log::info!("SOCKS5 Proxy Enabled: {}", proxy_url);
    }
    let http_client = http_builder.build().context("HTTP Client Build Failed")?;
    let webdav_client = Client::builder()
        .tcp_nodelay(true)
        .tcp_keepalive(Duration::from_secs(60))
        .pool_max_idle_per_host(MAX_CONCURRENCY * 2)
        .danger_accept_invalid_certs(true)
        .build()
        .context("WebDAV Client Build Failed")?;
    let quota = Quota::per_second(nonzero!(RATE_LIMIT_PER_SEC));
    let limiter = RateLimiter::direct(quota);

    let state = Arc::new(AppState {
        http_client,
        webdav_client,
        webdav_auth_header,
        limiter,
    });
    let (tx, rx) = mpsc::channel::<String>(MAX_CONCURRENCY * 4);
    let rx = Arc::new(tokio::sync::Mutex::new(rx));
    for _ in 0..MAX_CONCURRENCY {
        let state_clone = state.clone();
        let rx_clone = rx.clone();
        tokio::spawn(async move {
            loop {
                let url = {
                    let mut rx_lock = rx_clone.lock().await;
                    match rx_lock.recv().await {
                        Some(u) => u,
                        None => break,
                    }
                };
                
                if let Err(_e) = process_url(state_clone.clone(), url).await {
                    // 深度学习不需要错误处理
                }
            }
        });
    }

    log::info!("starting redis batch pumper on: {}", REDIS_QUEUE_KEY);
    let redis_client = redis::Client::open(REDIS_URL).context("Redis Connection Failed")?;
    let mut redis_conn = redis_client.get_async_connection().await?;

    loop {
        let urls: Vec<String> = redis::cmd("LPOP")
            .arg(REDIS_QUEUE_KEY)
            .arg(REDIS_BATCH_SIZE)
            .query_async(&mut redis_conn)
            .await
            .unwrap_or_default();

        if urls.is_empty() {
            tokio::time::sleep(Duration::from_millis(50)).await;
            continue;
        }

        for url in urls {
            if tx.send(url).await.is_err() {
                log::error!("Internal Channel Closed!");
                return Ok(());
            }
        }
    }
}

async fn process_url(state: Arc<AppState>, url_str: String) -> Result<()> {
    state.limiter.until_ready().await;
    let response = state.http_client.get(&url_str).send().await?;
    if !response.status().is_success() {
        return Err(anyhow::anyhow!("Source HTTP error: {}", response.status()));
    }
    let file_name = get_filename_from_url(&url_str);
    let upload_url = format!("{}{}", WEBDAV_ENDPOINT.trim_end_matches('/'), file_name);
    let byte_stream = response.bytes_stream().map_err(|e| std::io::Error::new(std::io::ErrorKind::Other, e));
    let body = reqwest::Body::wrap_stream(byte_stream);
    let upload_res = state.webdav_client.put(&upload_url)
        .header("Authorization", &state.webdav_auth_header)
        .body(body)
        .send()
        .await?;
    if upload_res.status().is_success() || upload_res.status().as_u16() == 201 || upload_res.status().as_u16() == 204 {
        log::info!("swap: {} -> NAS", file_name);
    } else {
        return Err(anyhow::anyhow!("NAS upload error: {}", upload_res.status()));
    }
    Ok(())
}

fn get_filename_from_url(url_str: &str) -> String {
    if let Ok(parsed) = Url::parse(url_str) {
        if let Some(segments) = parsed.path_segments() {
            if let Some(last) = segments.last() {
                if !last.is_empty() {
                    return format!("/{}", last);
                }
            }
        }
    }
    format!("/file_{}.dat", uuid::Uuid::new_v4())
}