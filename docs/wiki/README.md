# Whatomate Documentation Wiki

Complete bilingual (English / Arabic) documentation for the Whatomate WhatsApp Business API platform, built with [MkDocs Material](https://squidfunk.github.io/mkdocs-material/).

## Structure

```
docs/wiki/
├── mkdocs.yml                    # MkDocs configuration with i18n
├── README.md                     # This file
├── en/                           # English documentation
│   ├── index.md                  # Home page
│   ├── overview.md               # Platform overview
│   ├── quickstart.md             # Quick start guide
│   ├── faq.md                    # Frequently asked questions
│   ├── release-notes.md          # Changelog / release notes
│   ├── users/                    # User Guide (11 pages)
│   ├── developers/               # Developer Guide (11 pages)
│   └── admins/                   # Admin Guide (8 pages)
├── ar/                           # Arabic documentation (mirror of en/)
│   ├── index.md
│   ├── faq.md
│   ├── release-notes.md
│   ├── users/
│   ├── developers/
│   └── admins/
└── overrides/                    # MkDocs theme overrides
    ├── partials/                 # Custom HTML partials
    └── assets/                   # Custom CSS/JS/assets
```

## Quick Start

### Prerequisites

- Python 3.8+
- pip

### Installation

```bash
cd docs/wiki
pip install mkdocs-material mkdocs-static-i18n
```

### Local Development

```bash
mkdocs serve
```

Open [http://127.0.0.1:8000/](http://127.0.0.1:8000/) in your browser.

### Build for Production

```bash
mkdocs build
```

Output is written to `site/` directory.

## Adding New Pages

### 1. Create the English Page

Create a new `.md` file in the appropriate section:

```bash
# Example: adding a new user guide page
touch en/users/new-feature.md
```

Add front-matter at the top:

```yaml
---
title: New Feature Guide
---
```

### 2. Create the Arabic Translation

Create the corresponding Arabic file in the `ar/` directory:

```bash
touch ar/users/new-feature.md
```

Add front-matter with RTL support:

```yaml
---
title: دليل الميزة الجديدة
rtl: true
lang: ar
---
```

### 3. Update Navigation

Edit `mkdocs.yml` and add the new page to the `nav` section in **both** language trees:

```yaml
nav:
  - User Guide:
      - en/users/existing.md
      - en/users/new-feature.md    # <-- Add English
  # Arabic nav is auto-generated from en/ by mkdocs-static-i18n
```

### 4. Add Cross-References

Update related pages with "See Also" links pointing to the new page.

## Translation Guidelines

### What to Translate

- All headings (H1, H2, H3, etc.)
- All paragraph text
- All bullet and numbered lists
- Table content (headers and cells)
- Admonition titles and content
- Navigation labels

### What to Keep in English

- Code snippets (Go, JSON, bash, SQL, TOML, etc.)
- API endpoints (`POST /api/auth/login`)
- File paths (`internal/handlers/auth_handlers.go`)
- Environment variable names (`WHATOMATE_JWT_SECRET`)
- Database column names (`whatsapp_account_id`)
- Technical identifiers and constants

### Translation Quality

- Use clear, professional Arabic
- Keep technical terms consistent (e.g., always use "واجهة برمجة التطبيقات" for "API")
- Use Arabic punctuation (، for comma, ؟ for question mark)
- Maintain the same document structure as the English version

## Updating Existing Pages

When updating an English page:

1. Edit the English file in `en/`
2. Update the corresponding Arabic file in `ar/`
3. If you add new sections, translate them in the Arabic version
4. If you change API endpoints or code, update both versions

> **Tip:** Keep English and Arabic versions in sync. An outdated translation is worse than no translation.

## MkDocs Configuration

### Key Settings in `mkdocs.yml`

| Setting | Purpose |
|---------|---------|
| `theme.name` | Material theme |
| `plugins.i18n` | Bilingual support (en/ar) |
| `plugins.search.lang` | Search in both English and Arabic |
| `extra.alternate` | Language switcher configuration |
| `markdown_extensions` | Admonitions, tables, code highlighting, tabs |

### Language Switcher

The language switcher is configured in `extra.alternate`:

```yaml
extra:
  alternate:
    - name: English
      link: /en/
      lang: en
      direction: ltr
    - name: العربية
      link: /ar/
      lang: ar
      direction: rtl
```

MkDocs Material automatically renders this as a dropdown in the header.

## Placeholder Sections

The following pages contain placeholder content for future expansion:

- `en/faq.md` / `ar/faq.md` — Look for `<!-- TODO: Add more FAQs -->`
- `en/release-notes.md` / `ar/release-notes.md` — Look for `<!-- TODO: Add release notes for new versions -->`

To add content to these sections, simply edit the file and replace the placeholder comments with actual content.

## Deployment

See [Deployment Guide](en/admins/deployment.md) for detailed instructions. Quick summary:

### GitHub Pages

```bash
mkdocs gh-deploy
```

### Docker

```bash
docker run --rm -it -p 8000:8000 -v ${PWD}:/docs squidfunk/mkdocs-material
```

### Static Host

```bash
mkdocs build
# Upload site/ directory to any static host (Netlify, Vercel, S3, etc.)
```

## Contributing

1. Fork the repository
2. Create a branch: `git checkout -b docs/my-change`
3. Make your changes (English + Arabic)
4. Preview locally: `mkdocs serve`
5. Submit a pull request

## License

Copyright © 2024-2026 Whatomate. All rights reserved.
