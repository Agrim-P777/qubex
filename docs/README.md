# Documentation source and publishing

`docs/` contains the GitHub Pages source: short Markdown tutorials, a Jekyll layout, and CSS. The JSON fixtures remain canonical in the repository’s `examples/` folder. The publishing workflow copies them into the site artifact as `examples/*.json` so the site’s download links work.

To publish, set **Settings → Pages → Build and deployment → Source** to **GitHub Actions**, then merge this folder and [the Pages workflow](../.github/workflows/docs-pages.yml) into the repository’s default branch. Changes under `docs/` or `examples/` trigger a build and deployment from that branch. The workflow can also be run manually from the default branch. It does not create or change the repository’s Pages setting for you.

The workflow uses GitHub’s supported Jekyll, Pages artifact, and deployment actions. It obtains the site’s base path from `configure-pages`, builds flat `*.html` pages from the Markdown source, and copies downloadable sample JSON into the artifact. Internal Markdown links work on GitHub and are rewritten to rendered pages by `jekyll-relative-links` during the site build.

Each tutorial page should:

- Include one short task and an expected result.
- Use one canonical `examples/*.json` fixture throughout the page.
- Start from a fresh session unless it explicitly continues a previous step.
- Explain a new shortcut or form value at first use and link to the reference for exhaustive lists.
- Keep its `nav_order`, `title`, `summary`, and `permalink` front matter current.

GitHub also supports publishing directly from a branch’s `docs/` folder. This project uses the Actions workflow to build the Jekyll site and include the shared JSON examples. See [GitHub’s publishing-source guide](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site).
