#!/usr/bin/env node
/**
 * Renders AEGIS_Architecture_Diagram.png and .pdf from the HTML source.
 *
 * Prerequisites:
 *   npm install --no-save puppeteer    (one-time, from repo root)
 *
 * Usage:
 *   node scripts/render-architecture-diagram.mjs
 *
 * Output:
 *   AEGIS_Architecture_Diagram.png   (2× retina, ~1100 KB)
 *   AEGIS_Architecture_Diagram.pdf   (vector, single page)
 *
 * The HTML source is scripts/architecture-diagram.html.
 * The logo is loaded from logo-small.png in the repo root.
 * Edit the HTML to update the diagram content, then re-run this script.
 */

import puppeteer from 'puppeteer';
import { fileURLToPath } from 'url';
import path from 'path';
import fs from 'fs';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '..');
const htmlPath = path.join(__dirname, 'architecture-diagram.html');

const launchOptions = {
  headless: true,
  args: ['--no-sandbox', '--disable-setuid-sandbox'],
};

const chromePath = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
if (fs.existsSync(chromePath)) {
  launchOptions.executablePath = chromePath;
}

const browser = await puppeteer.launch(launchOptions);
const page = await browser.newPage();
await page.setViewport({ width: 1660, height: 1200, deviceScaleFactor: 2 });
await page.goto(`file://${htmlPath}`, { waitUntil: 'networkidle0' });

const bodyHeight = await page.evaluate(() => document.body.scrollHeight);
await page.setViewport({ width: 1660, height: bodyHeight + 60, deviceScaleFactor: 2 });

// PNG (2× retina for crisp rendering)
await page.screenshot({
  path: path.join(repoRoot, 'AEGIS_Architecture_Diagram.png'),
  fullPage: true,
  omitBackground: false,
});
console.log('✓ AEGIS_Architecture_Diagram.png');

// PDF (single page, exact fit)
await page.pdf({
  path: path.join(repoRoot, 'AEGIS_Architecture_Diagram.pdf'),
  width: '1660px',
  height: (bodyHeight + 60) + 'px',
  printBackground: true,
  margin: { top: 0, right: 0, bottom: 0, left: 0 },
});
console.log('✓ AEGIS_Architecture_Diagram.pdf');

await browser.close();
