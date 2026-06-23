import { createReadStream, existsSync } from 'node:fs';
import { stat } from 'node:fs/promises';
import { createServer } from 'node:http';
import path from 'node:path';
import process from 'node:process';

const root = process.cwd();
const port = Number(process.env.PORT ?? 4173);

const types = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.webp': 'image/webp',
};

function resolvePath(requestUrl) {
  const url = new URL(requestUrl ?? '/', `http://127.0.0.1:${port}`);
  const requested = decodeURIComponent(url.pathname === '/' ? '/index.html' : url.pathname);
  const filePath = path.resolve(root, requested.replace(/^\/+/, ''));
  const relative = path.relative(root, filePath);
  if (relative.startsWith('..') || path.isAbsolute(relative)) return '';
  return filePath;
}

createServer(async (request, response) => {
  const filePath = resolvePath(request.url);
  if (!filePath || !existsSync(filePath)) {
    response.statusCode = 404;
    response.end('Not found');
    return;
  }

  const fileStat = await stat(filePath);
  if (!fileStat.isFile()) {
    response.statusCode = 404;
    response.end('Not found');
    return;
  }

  response.statusCode = 200;
  response.setHeader('content-type', types[path.extname(filePath)] ?? 'application/octet-stream');
  response.setHeader('content-length', String(fileStat.size));
  createReadStream(filePath).pipe(response);
}).listen(port, '127.0.0.1', () => {
  console.log(`Serving ${root} at http://127.0.0.1:${port}/`);
});
