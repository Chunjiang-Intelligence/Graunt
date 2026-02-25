> [!CAUTION]
> 该目录下的程序包含高并发网络请求逻辑。在未取得授权的情况下，严禁对非自有设施进行测试！
>
> 1. 法律风险：大规模抓取 Z-Library 或 Reddit 可能违反当地法律、版权法及ToS。
> 2. 技术风险：本程序的极速模式可能被识别为 DDoS 攻击。
> 3. 责任界定：开发者不对任何因使用本代码造成的服务中断、数据泄露或法律纠纷承担责任。请务必遵守 `robots.txt` 协议并控制请求速率。

---

1.  对于 Z-Library

    Rust 端的 `process_url` 在下载 PDF 时，必须正确处理 302 重定向。`reqwest` 默认会自动处理。
    你需要在 Rust 代码的 `http_client` 构建时，添加从浏览器抓取到的 `Cookie`，否则 `z-library` 链接会跳转到登录页而不是下载文件。
    
    解决方案（一）：
    
    在 Rust 的 `main` 函数 `Client::builder()` 后添加：
    ```rust
    let mut headers = reqwest::header::HeaderMap::new();
    headers.insert("Cookie", "remix_userid=...; remix_userkey=...".parse().unwrap()); // 填入你的 Z-Lib Cookie
    http_builder = http_builder.default_headers(headers);
    ```
    
    解决方案（二）：
    
    使用 CloudFlare Workers 搭建特殊的反向代理。

2.  对于 Reddit

    Rust 端接收到的将是 `.../comment/xxxx.json` 这种 URL。
    Spider 它会把 JSON 当作文件下载下来，直接 Stream 到 WebDAV（NAS）里。文件后缀名为 `.json`。

3.  联动

    先运行 Python 脚本，将 Redis 队列填满（比如先填入 100万条）。
    然后启动 Rust 程序。Rust 程序会利用 `LPOP` 批量消费。
