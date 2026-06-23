# Awesome Prompt

Compressed image and prompt archive for MeiGen prompt examples.

## Build

Install dependencies:

```bash
npm install
```

Build the compressed archive from the local scrape:

```bash
npm run build:meigen
```

Default source directory:

```text
E:/image-garllery-wesite/meigen_export
```

Useful options:

```bash
npm run build:meigen -- --quality=76 --max=1280 --concurrency=8
npm run build:meigen -- --raw-base=https://raw.githubusercontent.com/mrslimslim/awesome-prompt/main
npm run build:meigen -- --clean=false
```

If `origin` is a GitHub remote, the script detects the raw base automatically.
After adding a GitHub remote, regenerate only the JSON URL fields without recompressing:

```bash
npm run build:meigen -- --clean=false
```

## Output

- `images/` compressed prompt images
- `video-thumbnails/` compressed video thumbnails
- `data/prompts-all.json` prompt metadata with `cdnImages`
- `data/prompts-images.json` image prompt metadata
- `data/prompts-videos.json` video prompt metadata
- `data/prompts-all.csv` CSV export
- `data/summary.json` build and compression summary

## Preview

Start a tiny local static server:

```bash
npm run serve
```

Then open:

```text
http://127.0.0.1:4173/
```

Raw GitHub URL pattern:

```text
https://raw.githubusercontent.com/mrslimslim/awesome-prompt/main/images/2054962469810978916/0.webp
```

Convert a normal GitHub page URL:

```text
https://github.com/mrslimslim/awesome-prompt/blob/main/images/1924288320881750108/0.webp
```

to a raw image URL:

```text
https://raw.githubusercontent.com/mrslimslim/awesome-prompt/main/images/1924288320881750108/0.webp
```
