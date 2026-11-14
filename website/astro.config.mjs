import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';
import starlight from '@astrojs/starlight';
import sitemap from '@astrojs/sitemap';

export default defineConfig({
  vite: { plugins: [tailwindcss()] },
  site: 'https://elrefai99.github.io',
  base: process.env.VERCEL ? '/' : process.env.NODE_ENV === 'production' ? '/Leoxy' : '/',
  integrations: [sitemap(), starlight({
    title: 'Leoxy Documentation',
    description: 'Guides and reference for the Leoxy reverse proxy.',
    disable404Route: true,
    favicon: '/favicon.svg',
    customCss: ['./src/styles/docs.css'],
    social: [{ icon: 'github', label: 'GitHub', href: 'https://github.com/elrefai99/Leoxy' }],
    sidebar: [
      { label: 'Getting started', link: '/docs/' },
      { label: 'Installation', link: '/docs/installation/' },
      { label: 'Configuration', link: '/docs/configuration/' },
      { label: 'Configuration reference', link: '/docs/config-reference/' },
      { label: 'Usage', link: '/docs/usage/' },
      { label: 'Reference', link: '/docs/reference/' },
      { label: 'FAQ', link: '/docs/faq/' },
    ],
    components: {
      SiteTitle: './src/components/SiteTitle.astro',
      SocialIcons: './src/components/SiteNavigation.astro',
      ThemeSelect: './src/components/Empty.astro',
      Footer: './src/components/SiteFooter.astro',
    },
    head: [
      { tag: 'meta', attrs: { property: 'og:image', content: 'https://elrefai99.github.io/Leoxy/social-card.png' } },
      { tag: 'meta', attrs: { name: 'twitter:card', content: 'summary_large_image' } },
      { tag: 'meta', attrs: { name: 'twitter:image', content: 'https://elrefai99.github.io/Leoxy/social-card.png' } },
      { tag: 'meta', attrs: { property: 'og:site_name', content: 'Leoxy' } },
    ],
  })],
});
