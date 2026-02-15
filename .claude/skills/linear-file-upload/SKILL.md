---
name: linear-file-upload
description: Upload files (screenshots, logs, images) to Linear issues and comments. Use when attaching bug reports, proof-of-work, validation artifacts, visual references, or any file to a Linear ticket via the Linear MCP server.
allowed-tools: Bash
---

# Linear File Upload

## When to use

- When creating a bug report and need to attach a visual representation of the issue
- When completing a task and need to attach validation evidence (screenshots, logs, output files)
- When updating a ticket with visual context, references, or supporting artifacts
- Any time a file needs to be embedded in a Linear issue description or comment

## How it works

Linear automatically downloads and permanently stores images referenced in markdown
descriptions and comments. You upload the file to a temporary public URL, then
embed it in the Linear ticket markdown. Linear copies the image to its own storage
at creation time, so the temp URL expiring later doesn't matter.

## Steps

1. Capture or locate the file you want to attach (e.g., a screenshot at `./evidence/screenshot.png`)

2. Upload it using the bundled script:
```bash
   URL=$(bash .claude/skills/linear-file-upload/scripts/upload.sh ./evidence/screenshot.png)
```

3. Embed the returned URL in the Linear issue description or comment markdown:
```markdown
   ## Validation
   ![task validation screenshot](https://0x0.st/ABcd.png)
```

4. When you create or update the issue via the Linear MCP server, include this
   markdown in the description or comment body. Linear will automatically
   download the image and store it permanently.

## Supported file types

Images (png, jpg, gif, webp) are rendered inline by Linear. Other file types
(pdf, log, txt) will be linked but not previewed.

## Important notes

- The temp URL (0x0.st) has a 512MB limit and rate limits — only upload what's needed
- The temp URL is publicly accessible — do not upload files containing secrets,
  credentials, or customer PII
- For non-image files, consider pasting the content directly into the ticket
  description instead of uploading
