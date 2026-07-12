import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { getBlogPost, listBlogPosts } from '../src/blog.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const publicDir = path.resolve(__dirname, '../public');
const blogDir = path.join(publicDir, 'blog');
fs.mkdirSync(blogDir, { recursive: true });

for (const post of listBlogPosts()) {
  const full = getBlogPost(post.slug);
  fs.writeFileSync(path.join(blogDir, `${post.slug}.md`), full.body, 'utf8');
  console.log('wrote', post.slug);
}

const origin = 'https://mergeos.shop';
const urls = [
  '/',
  '/product',
  '/marketplace',
  '/how-it-works',
  '/ledger',
  '/protocol',
  '/contracts',
  '/whitepaper',
  '/mergeide',
  '/blog',
  '/airdrop',
  '/presale',
  '/sdk',
  '/agents',
  '/contributors',
  '/system',
  '/customers',
  '/solutions',
  '/live-feed',
];
for (const post of listBlogPosts()) urls.push(`/blog/${post.slug}`);

const xml = [
  '<?xml version="1.0" encoding="UTF-8"?>',
  '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
];
for (const u of urls) {
  xml.push('  <url>');
  xml.push(`    <loc>${origin}${u}</loc>`);
  xml.push('    <changefreq>weekly</changefreq>');
  xml.push(`    <priority>${u === '/' ? '1.0' : u.startsWith('/blog') ? '0.8' : '0.7'}</priority>`);
  xml.push('  </url>');
}
xml.push('</urlset>');
fs.writeFileSync(path.join(publicDir, 'sitemap.xml'), `${xml.join('\n')}\n`, 'utf8');
fs.writeFileSync(
  path.join(publicDir, 'robots.txt'),
  [
    'User-agent: *',
    'Allow: /',
    'Sitemap: https://mergeos.shop/sitemap.xml',
    '',
    'Allow: /blog',
    'Allow: /marketplace',
    'Allow: /ledger',
    'Allow: /protocol',
    'Allow: /whitepaper',
    'Allow: /mergeide',
    '',
  ].join('\n'),
  'utf8',
);
console.log('sitemap+robots ok', urls.length);
