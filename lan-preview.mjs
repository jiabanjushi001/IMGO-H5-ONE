// 仅供局域网开发预览：只允许指定手机和本机访问本地 API / WebSocket。
import { createReadStream } from 'node:fs'
import { stat } from 'node:fs/promises'
import { createServer, request as httpRequest } from 'node:http'
import { connect as connectTcp, isIP } from 'node:net'
import { dirname, extname, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const [host, phoneAddress] = process.argv.slice(2)
const port = 8765
const apiHost = '127.0.0.1'
const apiPort = 8088
const siteRoot = resolve(dirname(fileURLToPath(import.meta.url)), 'unpackage/dist/build/h5')

const isPrivateIPv4 = (value) => isIP(value) === 4 &&
  /^(?:10(?:\.\d{1,3}){3}|192\.168(?:\.\d{1,3}){2}|172\.(?:1[6-9]|2\d|3[01])(?:\.\d{1,3}){2})$/.test(value)

if (!isPrivateIPv4(host) || !isPrivateIPv4(phoneAddress)) {
  throw new Error('请指定本机和手机的内网 IPv4：node lan-preview.mjs 192.168.1.123 192.168.1.42')
}

const allowedClients = new Set([host, phoneAddress])
const siteOrigin = `http://${host}:${port}`
const mimeTypes = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.ico': 'image/x-icon',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.svg': 'image/svg+xml',
  '.webp': 'image/webp',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2',
}

const isApiPath = (pathname) =>
  /^\/(?:common|enterprise|avatar|storage|wss|scan)(?:\/|$)/i.test(pathname) ||
  /^\/static\/img\/emoji(?:\/|$)/i.test(pathname)

function canUseApi(req) {
  return allowedClients.has(req.socket.remoteAddress) &&
    (!req.headers.origin || req.headers.origin === siteOrigin)
}

function proxyHttp(req, res, pathname) {
  if (!canUseApi(req)) {
    res.writeHead(403).end('Forbidden')
    return
  }

  const upstream = httpRequest({
    hostname: apiHost,
    port: apiPort,
    method: req.method,
    path: req.url,
    headers: { ...req.headers, host: `${apiHost}:${apiPort}` },
  }, (response) => {
    const headers = { ...response.headers }
    const match = req.method === 'GET' && response.statusCode === 302 &&
      /^\/scan\/([gu])\/([a-zA-Z0-9._-]+)$/.exec(pathname)
    if (match?.[1] === 'g') {
      const groupNumber = match[2].split('.')[0]
      if (/^\d+$/.test(groupNumber)) {
        headers.location = `${siteOrigin}/#/pages/message/group/info?group_id=group-${groupNumber}&token=${encodeURIComponent(match[2])}`
      }
    } else if (match?.[1] === 'u' && headers.location) {
      const userID = new URL(headers.location, siteOrigin).hash.match(/^#\/index\?user_id=(\d+)$/)?.[1]
      if (userID) headers.location = `${siteOrigin}/#/pages/contacts/detail?id=${userID}`
    }
    res.writeHead(response.statusCode || 502, headers)
    response.pipe(res)
  })

  upstream.on('error', (error) => {
    console.error('本地 API 转发失败:', error.message)
    if (!res.headersSent) res.writeHead(502, { 'content-type': 'text/plain; charset=utf-8' })
    res.end('本地 API 不可用，请检查 127.0.0.1:8088')
  })
  req.pipe(upstream)
}

async function serveStatic(req, res, pathname) {
  if (req.method !== 'GET' && req.method !== 'HEAD') {
    res.writeHead(405).end()
    return
  }

  let filePath
  try {
    filePath = resolve(siteRoot, '.' + decodeURIComponent(pathname === '/' ? '/index.html' : pathname))
  } catch {
    res.writeHead(400).end()
    return
  }
  if (!filePath.startsWith(siteRoot + sep)) {
    res.writeHead(403).end()
    return
  }

  try {
    const info = await stat(filePath)
    if (!info.isFile()) throw new Error('not a file')
    res.writeHead(200, {
      'content-type': mimeTypes[extname(filePath).toLowerCase()] || 'application/octet-stream',
      'content-length': info.size,
      'cache-control': 'no-cache',
    })
    if (req.method === 'HEAD') res.end()
    else createReadStream(filePath).pipe(res)
  } catch {
    res.writeHead(404).end('Not found')
  }
}

const server = createServer((req, res) => {
  let pathname
  try {
    pathname = new URL(req.url, siteOrigin).pathname
  } catch {
    res.writeHead(400).end()
    return
  }
  if (isApiPath(pathname)) proxyHttp(req, res, pathname)
  else void serveStatic(req, res, pathname)
})

server.on('upgrade', (req, client, head) => {
  let pathname
  try {
    pathname = new URL(req.url, siteOrigin).pathname
  } catch {
    client.destroy()
    return
  }
  if (pathname !== '/wss' || !canUseApi(req)) {
    client.write('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n')
    client.end()
    return
  }

  const upstream = connectTcp(apiPort, apiHost)
  upstream.on('connect', () => {
    // 已在入口校验手机 IP 和 H5 来源；上游只接受与其 Host 相同的 Origin。
    const headers = {
      ...req.headers,
      host: `${apiHost}:${apiPort}`,
      origin: `http://${apiHost}:${apiPort}`,
    }
    const lines = [`${req.method} ${req.url} HTTP/${req.httpVersion}`]
    for (const [name, value] of Object.entries(headers)) {
      if (Array.isArray(value)) value.forEach((item) => lines.push(`${name}: ${item}`))
      else if (value !== undefined) lines.push(`${name}: ${value}`)
    }
    upstream.write(lines.join('\r\n') + '\r\n\r\n')
    if (head.length) upstream.write(head)
    client.pipe(upstream)
    upstream.pipe(client)
  })
  upstream.on('error', (error) => {
    console.error('本地 WebSocket 转发失败:', error.message)
    client.destroy()
  })
  client.on('error', () => upstream.destroy())
})

server.listen(port, host, () => {
  console.log(`局域网 H5：${siteOrigin}/#/pages/login/index`)
  console.log(`仅允许 ${phoneAddress} 和本机访问本地 API / WebSocket`)
})
