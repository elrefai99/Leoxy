import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';

const favicon = await readFile(new URL('../public/favicon.svg', import.meta.url));
const socialCard = await readFile(new URL('../public/social-card.svg', import.meta.url));

await Promise.all([
  sharp(favicon).resize(32, 32).png().toFile(fileURLToPath(new URL('../public/favicon-32x32.png', import.meta.url))),
  sharp(favicon).resize(180, 180).png().toFile(fileURLToPath(new URL('../public/apple-touch-icon.png', import.meta.url))),
  sharp(socialCard).png().toFile(fileURLToPath(new URL('../public/social-card.png', import.meta.url))),
]);
