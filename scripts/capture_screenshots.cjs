const path = require('path');
const puppeteer = require(path.resolve(__dirname, '../web/node_modules/puppeteer'));
const fs = require('fs');
const { execSync } = require('child_process');

async function main() {
  const imagesDir = path.resolve('B:/workgit/embdb/docs/images');
  fs.mkdirSync(imagesDir, { recursive: true });

  const tempFramesDir = path.resolve('B:/workgit/embdb/docs/images/frames');
  fs.mkdirSync(tempFramesDir, { recursive: true });

  // 1. Seed some Knowledge Graph links via API
  const edges = [
    { from: 'legal_tos_01', to: 'sec_auth_01', relation: 'GOVERNS' },
    { from: 'legal_nda_02', to: 'sec_audit_02', relation: 'COMPLIES_WITH' },
    { from: 'ai_hnsw_01', to: 'infra_simd_03', relation: 'ACCELERATED_BY' },
    { from: 'ai_bm25_02', to: 'ai_rrf_03', relation: 'FUSED_BY' },
    { from: 'ai_hnsw_01', to: 'ai_rrf_03', relation: 'FUSED_BY' },
    { from: 'ai_mcp_04', to: 'ai_rag_05', relation: 'INTEGRATES' },
    { from: 'dev_go_01', to: 'infra_mmap_01', relation: 'IMPLEMENTS' },
  ];

  for (const e of edges) {
    try {
      await fetch('http://localhost:8383/api/graph/edges', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(e),
      });
    } catch (err) {}
  }

  console.log('Launching browser with Puppeteer...');
  const browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });

  console.log('Navigating to http://localhost:8383...');
  await page.goto('http://localhost:8383', { waitUntil: 'networkidle0' });
  await new Promise(r => setTimeout(r, 1000));

  // --- SCREENSHOT 1: HYBRID SEARCH BENCH (DESKTOP) ---
  console.log('Capturing Screenshot 1: Dashboard & Search Bench...');
  const allBtns = await page.$$('button');
  for (const b of allBtns) {
    const txt = await page.evaluate(el => el.textContent, b);
    if (txt && txt.trim() === 'Search') {
      await b.click();
      break;
    }
  }
  await new Promise(r => setTimeout(r, 1000));

  await page.screenshot({
    path: path.join(imagesDir, 'embdb_studio_dashboard.png'),
    fullPage: false,
  });
  console.log('✓ Captured docs/images/embdb_studio_dashboard.png');

  // --- SCREENSHOT 2: 3D VECTOR SCATTER ---
  console.log('Capturing Screenshot 2: 3D Vector Scatter...');
  const tabs = await page.$$('header button');
  for (const t of tabs) {
    const text = await page.evaluate(el => el.textContent, t);
    if (text.includes('3D Vector Scatter')) {
      await t.click();
      break;
    }
  }
  await new Promise(r => setTimeout(r, 1000));

  await page.screenshot({
    path: path.join(imagesDir, 'embdb_studio_3d.png'),
    fullPage: false,
  });
  console.log('✓ Captured docs/images/embdb_studio_3d.png');

  // --- SCREENSHOT 3: KNOWLEDGE GRAPH ---
  console.log('Capturing Screenshot 3: Knowledge Graph...');
  for (const t of tabs) {
    const text = await page.evaluate(el => el.textContent, t);
    if (text.includes('Knowledge Graph')) {
      await t.click();
      break;
    }
  }
  await new Promise(r => setTimeout(r, 1000));

  await page.screenshot({
    path: path.join(imagesDir, 'embdb_studio_graph.png'),
    fullPage: false,
  });
  console.log('✓ Captured docs/images/embdb_studio_graph.png');

  // --- SCREENSHOT 4: MOBILE RESPONSIVE (390x844) ---
  console.log('Capturing Screenshot 4: Mobile Viewport...');
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2 });
  for (const t of tabs) {
    const text = await page.evaluate(el => el.textContent, t);
    if (text.includes('Hybrid Search Bench')) {
      await t.click();
      break;
    }
  }
  await new Promise(r => setTimeout(r, 800));

  await page.screenshot({
    path: path.join(imagesDir, 'embdb_mobile.png'),
    fullPage: false,
  });
  console.log('✓ Captured docs/images/embdb_mobile.png');

  // --- CAPTURE ANIMATED FRAMES FOR DEMO GIF ---
  console.log('Recording frames for animated demo GIF...');
  await page.setViewport({ width: 1200, height: 750, deviceScaleFactor: 1 });
  await page.goto('http://localhost:8383', { waitUntil: 'networkidle0' });
  await new Promise(r => setTimeout(r, 600));

  let frameIdx = 0;
  const captureFrame = async () => {
    const filename = `frame_${String(frameIdx).padStart(4, '0')}.png`;
    await page.screenshot({ path: path.join(tempFramesDir, filename) });
    frameIdx++;
  };

  // Frame 0-5: initial state
  for (let i = 0; i < 5; i++) {
    await captureFrame();
    await new Promise(r => setTimeout(r, 100));
  }

  // Type query
  const input = await page.$('input[placeholder*="query"]');
  if (input) {
    await input.click({ clickCount: 3 });
    await page.keyboard.press('Backspace');
    const newQ = 'vector SIMD acceleration and HNSW graphs';
    for (const char of newQ) {
      await page.keyboard.type(char);
      await captureFrame();
      await new Promise(r => setTimeout(r, 30));
    }
    await page.keyboard.press('Enter');
    for (let i = 0; i < 8; i++) {
      await captureFrame();
      await new Promise(r => setTimeout(r, 100));
    }
  }

  // Switch to 3D tab
  const newTabs = await page.$$('header button');
  for (const t of newTabs) {
    const text = await page.evaluate(el => el.textContent, t);
    if (text.includes('3D Vector Scatter')) {
      await t.click();
      break;
    }
  }
  for (let i = 0; i < 8; i++) {
    await captureFrame();
    await new Promise(r => setTimeout(r, 100));
  }

  // Switch to Knowledge Graph tab
  for (const t of newTabs) {
    const text = await page.evaluate(el => el.textContent, t);
    if (text.includes('Knowledge Graph')) {
      await t.click();
      break;
    }
  }
  for (let i = 0; i < 8; i++) {
    await captureFrame();
    await new Promise(r => setTimeout(r, 100));
  }

  await browser.close();

  // --- COMPILE GIF USING FFMPEG ---
  console.log(`Compiling ${frameIdx} frames into docs/images/embdb_demo.gif with ffmpeg...`);
  const gifPath = path.join(imagesDir, 'embdb_demo.gif');
  const palettePath = path.join(tempFramesDir, 'palette.png');

  // Palette generation for high visual quality
  execSync(`ffmpeg -y -framerate 10 -i "${path.join(tempFramesDir, 'frame_%04d.png')}" -vf "fps=10,scale=960:-1:flags=lanczos,palettegen" "${palettePath}"`);
  execSync(`ffmpeg -y -framerate 10 -i "${path.join(tempFramesDir, 'frame_%04d.png')}" -i "${palettePath}" -lavfi "fps=10,scale=960:-1:flags=lanczos [x]; [x][1:v] paletteuse=dither=bayer:bayer_scale=3" "${gifPath}"`);

  console.log('✓ Successfully created docs/images/embdb_demo.gif');

  // Cleanup frames
  fs.rmSync(tempFramesDir, { recursive: true, force: true });
  console.log('✓ Cleaned up temporary frames.');
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
