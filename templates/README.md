# LaTeX Templates

This directory contains the LaTeX templates used for PDF generation.

## Files

- **resume-template.latex** - Pandoc template that wraps the markdown content in a LaTeX document structure
- **resume.cls** - Custom LaTeX class defining the resume styling (fonts, spacing, margins, hyperlinks)

## Usage

Copy these templates to your config directory:

```bash
cp templates/resume-template.latex ~/.resume-tailor/
cp templates/resume.cls ~/.resume-tailor/
```

Then reference them in your `~/.resume-tailor/config.json`:

```json
{
  "pandoc": {
    "template_path": "~/.resume-tailor/resume-template.latex",
    "class_file": "~/.resume-tailor/resume.cls"
  }
}
```

## Customization

Feel free to customize these templates for your own styling preferences:

- **Fonts**: Edit `resume.cls` to change font family (currently Helvetica/sans-serif)
- **Margins**: Adjust geometry package settings in `resume.cls` (currently 0.5in all sides)
- **Colors**: Modify hyperlink colors in `resume.cls` (currently blue for URLs, black for links)
- **Spacing**: Adjust section/paragraph spacing in `resume.cls`
- **Document structure**: Modify `resume-template.latex` to change header/title formatting
