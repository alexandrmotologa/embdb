const fs = require('fs');
const path = require('path');
const { Resvg } = require('@resvg/resvg-js');

function buildLogoSvg() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="1024" height="1024">
  <defs>
    <clipPath id="squircle-clip">
      <rect x="24" y="24" width="976" height="976" rx="220" />
    </clipPath>

    <linearGradient id="cyan-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#00f5ff"/>
      <stop offset="100%" stop-color="#0284c7"/>
    </linearGradient>

    <linearGradient id="indigo-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#818cf8"/>
      <stop offset="100%" stop-color="#4f46e5"/>
    </linearGradient>

    <linearGradient id="facet-slate-light" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#475569"/>
      <stop offset="100%" stop-color="#334155"/>
    </linearGradient>

    <linearGradient id="facet-slate-dark" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#1e293b"/>
      <stop offset="100%" stop-color="#0f172a"/>
    </linearGradient>

    <linearGradient id="facet-obsidian" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" stop-color="#0f172a"/>
      <stop offset="100%" stop-color="#050811"/>
    </linearGradient>

    <filter id="subtle-shadow" x="-10%" y="-10%" width="120%" height="120%">
      <feDropShadow dx="0" dy="16" stdDeviation="20" flood-color="#000000" flood-opacity="0.14" />
    </filter>
  </defs>

  <!-- Luxury White Squircle Container -->
  <rect x="24" y="24" width="976" height="976" rx="220" fill="#ffffff" stroke="#e2e8f0" stroke-width="6" />

  <g clip-path="url(#squircle-clip)">
    <g transform="translate(512, 512)" filter="url(#subtle-shadow)">

      <!-- Hexagonal Architectural Gateway Frame -->
      <polygon points="
        0,-410
        355,-205
        355,205
        0,410
        -355,205
        -355,-205
      " fill="none" stroke="#0f172a" stroke-width="36" stroke-linejoin="round" />

      <polygon points="
        0,-375
        324,-187
        324,187
        0,375
        -324,187
        -324,-187
      " fill="none" stroke="#00f5ff" stroke-width="4" opacity="0.5" stroke-dasharray="16, 12" />

      <!-- Concentric Sonar Vector Radar Arc Accents -->
      <circle cx="0" cy="0" r="260" fill="none" stroke="#6366f1" stroke-width="2" opacity="0.25" stroke-dasharray="8, 8" />
      <circle cx="0" cy="0" r="320" fill="none" stroke="#00f5ff" stroke-width="1.5" opacity="0.2" />

      <!-- Solid Watertight Base Silhouette for Zero Background Bleed -->
      <polygon points="
        0,-240
        65,-140
        120,-260
        135,-110
        260,-40
        310,120
        210,180
        140,110
        0,230
        -140,110
        -210,180
        -310,120
        -260,-40
        -135,-110
        -120,-260
        -65,-140
      " fill="#0b0f19" />

      <!-- Left Wing Facets -->
      <polygon points="-260,-40 -135,-110 -150,20" fill="url(#facet-slate-light)" />
      <polygon points="-260,-40 -150,20 -310,120" fill="url(#facet-slate-dark)" />
      <polygon points="-150,20 -310,120 -210,180" fill="url(#facet-obsidian)" />
      <polygon points="-150,20 -210,180 -70,90" fill="url(#facet-slate-light)" />
      <polygon points="-135,-110 -150,20 -70,0" fill="url(#facet-slate-dark)" />

      <!-- Right Wing Facets -->
      <polygon points="260,-40 135,-110 150,20" fill="url(#facet-slate-dark)" />
      <polygon points="260,-40 150,20 310,120" fill="url(#facet-slate-light)" />
      <polygon points="150,20 310,120 210,180" fill="url(#facet-slate-dark)" />
      <polygon points="150,20 210,180 70,90" fill="url(#facet-obsidian)" />
      <polygon points="135,-110 150,20 70,0" fill="url(#facet-slate-light)" />

      <!-- Bat Ears (Left and Right Radar Pinnae) -->
      <polygon points="-65,-140 -120,-260 -135,-110" fill="url(#facet-slate-light)" />
      <polygon points="-65,-140 -105,-220 -100,-115" fill="#00f5ff" opacity="0.3" />
      <polygon points="65,-140 120,-260 135,-110" fill="url(#facet-slate-dark)" />
      <polygon points="65,-140 105,-220 100,-115" fill="#0284c7" opacity="0.3" />

      <!-- Craniofacial Diamond Mask (Head) -->
      <polygon points="0,-240 -65,-140 0,-110" fill="url(#facet-slate-light)" />
      <polygon points="0,-240 65,-140 0,-110" fill="url(#facet-slate-dark)" />
      <polygon points="-65,-140 0,-110 -70,0" fill="url(#facet-obsidian)" />
      <polygon points="65,-140 0,-110 70,0" fill="url(#facet-slate-light)" />
      <polygon points="0,-110 -70,0 0,60" fill="url(#facet-slate-light)" />
      <polygon points="0,-110 70,0 0,60" fill="url(#facet-slate-dark)" />

      <!-- Sonar Core / Lower Torso -->
      <polygon points="0,60 -70,90 0,230" fill="url(#facet-slate-dark)" />
      <polygon points="0,60 70,90 0,230" fill="url(#facet-slate-light)" />

      <!-- Center Acoustic Hex Nexus Emblem -->
      <polygon points="
        0,15
        35,35
        35,75
        0,95
        -35,75
        -35,35
      " fill="#090d16" stroke="#00f5ff" stroke-width="4" />
      <circle cx="0" cy="55" r="12" fill="url(#cyan-glow)" />

      <!-- Predatory Almond Optics (Cyan Slit Eyes) -->
      <polygon points="-45,-75 -20,-85 -15,-65 -40,-60" fill="url(#cyan-glow)" />
      <polygon points="45,-75 20,-85 15,-65 40,-60" fill="url(#cyan-glow)" />
      <polygon points="-36,-76 -22,-82 -25,-68 -38,-65" fill="#ffffff" opacity="0.9" />
      <polygon points="36,-76 22,-82 25,-68 38,-65" fill="#ffffff" opacity="0.9" />

      <!-- Micro Vector Nodes on Wing Tips -->
      <circle cx="-310" cy="120" r="7" fill="#00f5ff" />
      <circle cx="310" cy="120" r="7" fill="#00f5ff" />
      <circle cx="0" cy="-240" r="6" fill="#818cf8" />
      <circle cx="0" cy="230" r="6" fill="#818cf8" />

    </g>
  </g>
</svg>`;
}

async function main() {
  const svg = buildLogoSvg();
  const outputDir = path.resolve('B:/workgit/embdb/docs/images');
  fs.mkdirSync(outputDir, { recursive: true });

  const svgPath = path.join(outputDir, 'logo.svg');
  const pngPath = path.join(outputDir, 'logo.png');

  fs.writeFileSync(svgPath, svg);

  const resvg = new Resvg(svg, { fitTo: { mode: 'width', value: 1024 } });
  const pngBuffer = resvg.render().asPng();
  fs.writeFileSync(pngPath, pngBuffer);

  console.log('✓ Successfully rendered logo.svg and logo.png at 1024x1024 to ' + outputDir);
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
