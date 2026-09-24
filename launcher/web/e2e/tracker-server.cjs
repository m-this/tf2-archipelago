// Tiny static server for the standalone tracker entry in browser tests.
const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '../../../dist/tracker');
const types = {
  '.css': 'text/css',
  '.html': 'text/html',
  '.js': 'text/javascript',
  '.json': 'application/json',
  '.png': 'image/png',
  '.svg': 'image/svg+xml',
  '.webp': 'image/webp',
};

http
  .createServer((request, response) => {
    const pathname = new URL(request.url, 'http://127.0.0.1:8472').pathname;
    const file = path.resolve(root, `.${pathname === '/' ? '/index.html' : pathname}`);
    if (!file.startsWith(`${root}${path.sep}`)) {
      response.writeHead(403).end();
      return;
    }
    fs.readFile(file, (error, bytes) => {
      if (error) {
        response.writeHead(error.code === 'ENOENT' ? 404 : 500).end();
        return;
      }
      response.writeHead(200, {
        'Content-Type': types[path.extname(file)] || 'application/octet-stream',
      });
      response.end(bytes);
    });
  })
  .listen(8472, '127.0.0.1');
