# Leoxy website

The website is an Astro static site with Starlight documentation. It lives in its own folder and does not change the Go app or root package configuration. `npm run dev` serves the home page at `/`; production builds use `/Leoxy/` for the GitHub Pages project site.

## Develop and build

Install Node.js and npm, then run these commands from `website/`:

```sh
npm install
npm run dev
npm run build
npm run preview
```

The build runs Astro's TypeScript/content checks and generates static output in `website/dist/`. The prebuild script creates the PNG icons and social card from the SVG sources.

## GitHub Pages setup

The workflow in `.github/workflows/website.yml` builds on pushes and pull requests that change `website/**`. It deploys `main` to GitHub Pages. In repository Settings → Pages, set the build and deployment source to **GitHub Actions**. Astro is configured for `https://elrefai99.github.io/Leoxy/`; update `site` and `base` in `astro.config.mjs` if the repository owner or Pages URL changes.

The screenshots were supplied separately and used as the visual reference. The repository does not publish an official supported platform/version matrix, so the installation guide lists the Go toolchain version declared in `app/go.mod` without asserting a platform support guarantee.
