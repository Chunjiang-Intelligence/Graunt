/**
 * Z-Library Auth Injector & Reverse Proxy
 * 
 * Environment Variables:
 * TARGET_UPSTREAM: 目标域名 (不带 https://), 例: z-library.sk
 * ZLIB_COOKIES: 你的登录Cookie, 例: remix_userid=12345; remix_userkey=xxxx
 */

export default {
  async fetch(request, env, ctx) {
    const TARGET = env.TARGET_UPSTREAM || 'z-library.sk';
    const COOKIES = env.ZLIB_COOKIES || '';
    const url = new URL(request.url);
    const targetUrl = new URL(url.pathname + url.search, `https://${TARGET}`);
    const newHeaders = new Headers(request.headers);
    newHeaders.set('Cookie', COOKIES);
    newHeaders.set('Host', TARGET);
    newHeaders.set('Referer', `https://${TARGET}/`);
    if (!newHeaders.get('User-Agent')) {
        newHeaders.set('User-Agent', 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36');
    }
    newHeaders.delete('Cf-Connecting-Ip');
    newHeaders.delete('X-Forwarded-For');
    newHeaders.delete('X-Real-Ip');
    const newRequest = new Request(targetUrl, {
      method: request.method,
      headers: newHeaders,
      body: request.body,
      redirect: 'follow' // 让 Worker 自动跟随重定向
    });
    try {
      const response = await fetch(newRequest);
      const newResponseHeaders = new Headers(response.headers);
      newResponseHeaders.set('Access-Control-Allow-Origin', '*');
      return new Response(response.body, {
        status: response.status,
        statusText: response.statusText,
        headers: newResponseHeaders
      });

    } catch (e) {
      return new Response(JSON.stringify({ error: e.message, hint: "Z-Lib domain might be blocked or changed" }), { 
        status: 502,
        headers: { 'Content-Type': 'application/json' }
      });
    }
  },
};
