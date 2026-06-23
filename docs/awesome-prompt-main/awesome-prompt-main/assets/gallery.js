const state = {
  prompts: [],
  assets: [],
  visible: 160,
  selected: null,
};

const formatter = new Intl.NumberFormat('zh-CN');
const gallery = document.querySelector('#gallery');
const searchInput = document.querySelector('#searchInput');
const modelSelect = document.querySelector('#modelSelect');
const typeSelect = document.querySelector('#typeSelect');
const matchCount = document.querySelector('#matchCount');
const promptCount = document.querySelector('#promptCount');
const imageCount = document.querySelector('#imageCount');
const loadMore = document.querySelector('#loadMore');
const viewer = document.querySelector('#viewer');
const viewerImage = document.querySelector('#viewerImage');
const viewerModel = document.querySelector('#viewerModel');
const viewerTitle = document.querySelector('#viewerTitle');
const viewerPrompt = document.querySelector('#viewerPrompt');
const sourceLink = document.querySelector('#sourceLink');
const copyPrompt = document.querySelector('#copyPrompt');
const closeViewer = document.querySelector('#closeViewer');

function format(value) {
  return formatter.format(value ?? 0);
}

async function copyText(text, feedbackButton) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
  } else {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.left = '-9999px';
    document.body.append(textarea);
    textarea.select();
    document.execCommand('copy');
    textarea.remove();
  }

  if (!feedbackButton) return;
  const original = feedbackButton.textContent;
  feedbackButton.textContent = 'Copied';
  window.setTimeout(() => {
    feedbackButton.textContent = original;
  }, 1200);
}

function getSourceUrl(item) {
  return item.author?.username ? `https://x.com/${item.author.username}/status/${item.id}` : '';
}

function buildAssets(prompts) {
  return prompts.flatMap((item) =>
    (item.cdnImages ?? item.localImages ?? []).map((image, index) => ({
      key: `${item.id}-${index}`,
      image,
      index,
      item,
    })),
  );
}

function filteredAssets() {
  const query = searchInput.value.trim().toLowerCase();
  const model = modelSelect.value;
  const type = typeSelect.value;

  return state.assets.filter(({ item }) => {
    const haystack = [item.id, item.title, item.prompt, item.model, item.author?.name, item.author?.username].join(' ').toLowerCase();
    return (!query || haystack.includes(query)) && (model === 'all' || item.model === model) && (type === 'all' || item.sourceCollection === type);
  });
}

function render() {
  const matches = filteredAssets();
  matchCount.textContent = format(matches.length);
  gallery.textContent = '';

  for (const asset of matches.slice(0, state.visible)) {
    const card = document.createElement('article');
    card.className = 'tile';

    const imageButton = document.createElement('button');
    imageButton.className = 'thumb-button';
    imageButton.type = 'button';
    imageButton.addEventListener('click', () => openViewer(asset));

    const image = document.createElement('img');
    image.src = asset.image;
    image.alt = asset.item.title || asset.item.id;
    image.loading = 'lazy';
    imageButton.append(image);

    const info = document.createElement('div');
    info.className = 'tile-info';

    const meta = document.createElement('div');
    meta.className = 'tile-meta';

    const tag = document.createElement('span');
    tag.className = 'model-tag';
    tag.textContent = asset.item.model || asset.item.mediaType || 'Unknown';

    const copyButton = document.createElement('button');
    copyButton.className = 'copy-card';
    copyButton.type = 'button';
    copyButton.textContent = 'Copy';
    copyButton.addEventListener('click', () => copyText(asset.item.prompt || '', copyButton));

    meta.append(tag, copyButton);

    const excerpt = document.createElement('p');
    excerpt.className = 'prompt-excerpt';
    excerpt.textContent = asset.item.prompt || asset.item.title || 'No prompt available';
    excerpt.title = excerpt.textContent;

    info.append(meta, excerpt);
    card.append(imageButton, info);
    gallery.append(card);
  }

  loadMore.hidden = state.visible >= matches.length;
}

function openViewer(asset) {
  state.selected = asset;
  viewerImage.src = asset.image;
  viewerImage.alt = asset.item.title || asset.item.id;
  viewerModel.textContent = asset.item.model || 'Unknown model';
  viewerTitle.textContent = asset.item.title || asset.item.id;
  viewerPrompt.textContent = asset.item.prompt || 'No prompt available';
  const sourceUrl = getSourceUrl(asset.item);
  sourceLink.href = sourceUrl || '#';
  sourceLink.hidden = !sourceUrl;
  copyPrompt.textContent = 'Copy prompt';
  viewer.showModal();
}

async function init() {
  const [prompts, summary] = await Promise.all([
    fetch('./data/prompts-all.json').then((response) => response.json()),
    fetch('./data/summary.json').then((response) => response.json()),
  ]);

  state.prompts = prompts;
  state.assets = buildAssets(prompts);
  promptCount.textContent = format(summary.promptCount ?? prompts.length);
  imageCount.textContent = format(summary.assetCount ?? state.assets.length);

  for (const model of Array.from(new Set(prompts.map((item) => item.model).filter(Boolean))).sort()) {
    const option = document.createElement('option');
    option.value = model;
    option.textContent = model;
    modelSelect.append(option);
  }

  render();
}

searchInput.addEventListener('input', () => {
  state.visible = 160;
  render();
});

modelSelect.addEventListener('change', () => {
  state.visible = 160;
  render();
});

typeSelect.addEventListener('change', () => {
  state.visible = 160;
  render();
});

loadMore.addEventListener('click', () => {
  state.visible += 160;
  render();
});

closeViewer.addEventListener('click', () => viewer.close());

copyPrompt.addEventListener('click', async () => {
  if (!state.selected) return;
  await copyText(state.selected.item.prompt || '', copyPrompt);
});

init().catch((error) => {
  gallery.textContent = `Failed to load gallery: ${error.message}`;
});
