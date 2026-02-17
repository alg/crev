// crev web review app

const LINE_CONTEXT = 0;
const LINE_ADDED = 1;
const LINE_REMOVED = 2;

let state = {
  files: [],
  baseCommit: '',
  selectedFileIndex: 0,
  comments: [],  // {file, line_start, line_end, side, severity, text}
  editingComment: null,  // {file, lineStart, side} - identifies which comment form is open
};

// --- Init ---

async function init() {
  initTheme();
  initSidebarResize();

  const resp = await fetch('/api/diff');
  const data = await resp.json();
  state.files = data.files || [];
  state.baseCommit = data.base_commit || '';

  renderStats();
  renderFileTree();

  if (state.files.length > 0) {
    selectFile(0);
  } else {
    document.getElementById('diff-view').innerHTML = '<div class="empty-state">No changes to review</div>';
  }

  document.getElementById('btn-submit').addEventListener('click', () => submitReview(false));
  document.getElementById('btn-approve').addEventListener('click', () => submitReview(true));
}

// --- Theme ---

function initTheme() {
  const saved = localStorage.getItem('crev-theme');
  if (saved) {
    document.documentElement.dataset.theme = saved;
  }
  updateThemeIcon();

  document.getElementById('theme-toggle').addEventListener('click', () => {
    const current = document.documentElement.dataset.theme;
    const next = current === 'light' ? 'dark' : 'light';
    document.documentElement.dataset.theme = next;
    localStorage.setItem('crev-theme', next);
    updateThemeIcon();
  });
}

function updateThemeIcon() {
  const btn = document.getElementById('theme-toggle');
  const isLight = document.documentElement.dataset.theme === 'light';
  btn.innerHTML = isLight ? '&#9728;' : '&#9790;';
  btn.title = isLight ? 'Switch to dark theme' : 'Switch to light theme';
}

// --- Sidebar Resize ---

function initSidebarResize() {
  const saved = localStorage.getItem('crev-sidebar-width');
  if (saved) {
    document.documentElement.style.setProperty('--sidebar-width', saved + 'px');
  }

  const handle = document.getElementById('sidebar-resize-handle');
  let dragging = false;

  handle.addEventListener('mousedown', (e) => {
    e.preventDefault();
    dragging = true;
    handle.classList.add('dragging');
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  });

  document.addEventListener('mousemove', (e) => {
    if (!dragging) return;
    const sidebar = document.getElementById('sidebar');
    const min = 150;
    const max = window.innerWidth / 2;
    const width = Math.min(max, Math.max(min, e.clientX));
    document.documentElement.style.setProperty('--sidebar-width', width + 'px');
  });

  document.addEventListener('mouseup', () => {
    if (!dragging) return;
    dragging = false;
    handle.classList.remove('dragging');
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    const width = parseInt(getComputedStyle(document.getElementById('sidebar')).width);
    localStorage.setItem('crev-sidebar-width', width);
  });
}

// --- Stats ---

function renderStats() {
  let additions = 0, deletions = 0;
  for (const file of state.files) {
    for (const hunk of (file.hunks || [])) {
      for (const line of (hunk.lines || [])) {
        if (line.type === LINE_ADDED) additions++;
        if (line.type === LINE_REMOVED) deletions++;
      }
    }
  }

  const statsEl = document.getElementById('stats');
  statsEl.innerHTML = `${state.files.length} file${state.files.length !== 1 ? 's' : ''} changed, ` +
    `<span class="stat-added">+${additions}</span> ` +
    `<span class="stat-removed">-${deletions}</span>`;

  updateCommentCount();
}

function updateCommentCount() {
  const el = document.getElementById('comment-count');
  const n = state.comments.length;
  el.textContent = `${n} comment${n !== 1 ? 's' : ''}`;
}

// --- File Tree ---

function renderFileTree() {
  const tree = buildTree(state.files);
  const container = document.getElementById('file-tree');
  container.innerHTML = '';
  renderTreeNodes(container, tree, 0);
}

function buildTree(files) {
  const root = { children: {}, isDir: true };

  files.forEach((file, index) => {
    const parts = file.path.split('/');
    let current = root;
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      if (!current.children[part]) {
        current.children[part] = {
          name: part,
          children: {},
          isDir: i < parts.length - 1,
          fileIndex: i === parts.length - 1 ? index : -1,
          file: i === parts.length - 1 ? file : null,
          expanded: true,
        };
      }
      current = current.children[part];
    }
  });

  return flattenSingleChildDirs(root);
}

function flattenSingleChildDirs(node) {
  const children = Object.values(node.children);
  for (const child of children) {
    if (child.isDir) {
      flattenSingleChildDirs(child);
      const grandchildren = Object.values(child.children);
      if (grandchildren.length === 1 && grandchildren[0].isDir) {
        const gc = grandchildren[0];
        child.name = child.name + '/' + gc.name;
        child.children = gc.children;
      }
    }
  }
  return node;
}

function renderTreeNodes(container, node, depth) {
  const children = Object.values(node.children);
  // Sort: directories first, then files, both alphabetically
  children.sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
    return a.name.localeCompare(b.name);
  });

  for (const child of children) {
    const el = document.createElement('div');
    el.className = 'tree-item' + (child.isDir ? ' directory' : '');
    if (!child.isDir && child.fileIndex === state.selectedFileIndex) {
      el.classList.add('selected');
    }

    let html = '';
    for (let i = 0; i < depth; i++) html += '<span class="tree-indent"></span>';

    if (child.isDir) {
      html += `<span class="tree-icon">${child.expanded ? '&#9660;' : '&#9654;'}</span>`;
      html += `<span class="tree-name">${escapeHtml(child.name)}</span>`;
    } else {
      html += `<span class="tree-icon"></span>`;
      html += `<span class="file-status ${fileStatusClass(child.file)}">${fileStatusIcon(child.file)}</span>`;
      html += `<span class="tree-name">${escapeHtml(child.name)}</span>`;
      const commentCount = countCommentsForFile(child.file.path);
      if (commentCount > 0) {
        html += `<span class="tree-badge">${commentCount}</span>`;
      }
    }

    el.innerHTML = html;

    if (child.isDir) {
      el.addEventListener('click', () => {
        child.expanded = !child.expanded;
        renderFileTree();
      });
    } else {
      el.addEventListener('click', () => selectFile(child.fileIndex));
    }

    container.appendChild(el);

    if (child.isDir && child.expanded) {
      renderTreeNodes(container, child, depth + 1);
    }
  }
}

function fileStatusClass(file) {
  if (file.is_new || file.is_untracked) return 'added';
  if (file.is_deleted) return 'deleted';
  return 'modified';
}

function fileStatusIcon(file) {
  if (file.is_untracked) return '?';
  if (file.is_new) return 'A';
  if (file.is_deleted) return 'D';
  if (file.is_binary) return 'B';
  return 'M';
}

function countCommentsForFile(path) {
  return state.comments.filter(c => c.file === path).length;
}

// --- File Selection ---

function selectFile(index) {
  state.selectedFileIndex = index;
  renderFileTree();
  renderDiff();
}

// --- Diff Rendering ---

function renderDiff() {
  const file = state.files[state.selectedFileIndex];
  if (!file) return;

  // File header
  const headerEl = document.getElementById('file-header');
  let additions = 0, deletions = 0;
  for (const hunk of (file.hunks || [])) {
    for (const line of (hunk.lines || [])) {
      if (line.type === LINE_ADDED) additions++;
      if (line.type === LINE_REMOVED) deletions++;
    }
  }
  headerEl.innerHTML = `<span>${escapeHtml(file.path)}</span>` +
    `<span class="file-header-stats"><span class="stat-added">+${additions}</span> <span class="stat-removed">-${deletions}</span></span>`;

  // Diff content
  const diffView = document.getElementById('diff-view');

  if (isImageFile(file.path)) {
    if (file.is_deleted) {
      diffView.innerHTML = '<div class="empty-state">Deleted image</div>';
    } else {
      diffView.innerHTML = `<div class="image-preview"><img src="/api/file?path=${encodeURIComponent(file.path)}" alt="${escapeAttr(file.path)}"></div>`;
    }
    return;
  }

  if (file.is_binary) {
    diffView.innerHTML = '<div class="empty-state">Binary file</div>';
    return;
  }

  if (!file.hunks || file.hunks.length === 0) {
    diffView.innerHTML = '<div class="empty-state">No changes in this file</div>';
    return;
  }

  const language = getLanguageFromPath(file.path);

  let html = '<table class="diff-table">';

  for (const hunk of file.hunks) {
    // Hunk header
    html += `<tr class="diff-hunk-header"><td colspan="5">${escapeHtml(hunk.header)}</td></tr>`;

    for (const line of (hunk.lines || [])) {
      const lineClass = lineTypeClass(line.type);
      const side = line.type === LINE_REMOVED ? 'old' : 'new';
      const lineNum = line.type === LINE_REMOVED ? line.old_num : line.new_num;
      const hasComment = findComment(file.path, lineNum, side) !== null;

      html += `<tr class="diff-line ${lineClass}${hasComment ? ' has-comment' : ''}" ` +
        `data-file="${escapeAttr(file.path)}" data-line="${lineNum}" data-side="${side}">`;

      // Add comment button
      html += `<td class="line-add-comment"><button class="add-comment-btn" title="Add comment">+</button></td>`;

      // Line numbers
      html += `<td class="line-num ${line.type === LINE_REMOVED ? 'removed' : ''}">${line.old_num || ''}</td>`;
      html += `<td class="line-num ${line.type === LINE_ADDED ? 'added' : ''}">${line.new_num || ''}</td>`;

      // Prefix
      const prefix = line.type === LINE_ADDED ? '+' : line.type === LINE_REMOVED ? '-' : ' ';
      html += `<td class="line-prefix ${lineClass}">${prefix}</td>`;

      // Content
      html += `<td class="line-content ${lineClass}">${highlightLine(line.content, language)}</td>`;
      html += '</tr>';

      // Show existing comment or editing form below this line
      if (hasComment) {
        const comment = findComment(file.path, lineNum, side);
        const isEditing = state.editingComment &&
          state.editingComment.file === file.path &&
          state.editingComment.lineStart === lineNum &&
          state.editingComment.side === side;
        if (isEditing) {
          html += renderCommentForm(file.path, lineNum, side, comment);
        } else {
          html += renderSavedComment(comment, file.path, lineNum, side);
        }
      } else if (state.editingComment &&
        state.editingComment.file === file.path &&
        state.editingComment.lineStart === lineNum &&
        state.editingComment.side === side) {
        html += renderCommentForm(file.path, lineNum, side, null);
      }
    }
  }

  html += '</table>';
  diffView.innerHTML = html;

  // Attach event listeners
  attachDiffListeners();
}

function lineTypeClass(type) {
  if (type === LINE_ADDED) return 'added';
  if (type === LINE_REMOVED) return 'removed';
  return '';
}

function renderCommentForm(filePath, lineNum, side, existingComment) {
  const severity = existingComment ? existingComment.severity : 'suggestion';
  const text = existingComment ? existingComment.text : '';

  let html = `<tr class="comment-row"><td colspan="5"><div class="comment-card">`;

  // Severity pills
  html += `<div class="comment-header"><div class="severity-pills">`;
  for (const sev of ['suggestion', 'question', 'concern', 'blocker']) {
    html += `<button class="severity-pill ${sev}${severity === sev ? ' active' : ''}" ` +
      `data-severity="${sev}">${capitalize(sev)}</button>`;
  }
  html += `</div></div>`;

  // Textarea
  html += `<div class="comment-body">`;
  html += `<textarea class="comment-textarea" placeholder="Write a comment...">${escapeHtml(text)}</textarea>`;
  html += `</div>`;

  // Actions
  html += `<div class="comment-actions">`;
  html += `<button class="btn btn-sm btn-secondary cancel-comment-btn">Cancel</button>`;
  html += `<button class="btn btn-sm btn-primary save-comment-btn" ` +
    `data-file="${escapeAttr(filePath)}" data-line="${lineNum}" data-side="${side}">` +
    `${existingComment ? 'Update' : 'Comment'}</button>`;
  html += `</div>`;

  html += `</div></td></tr>`;
  return html;
}

function renderSavedComment(comment, filePath, lineNum, side) {
  let html = `<tr class="comment-row"><td colspan="5"><div class="comment-card saved">`;

  html += `<div class="comment-header">`;
  html += `<div><span class="comment-severity-indicator ${comment.severity}"></span>`;
  html += `<span class="comment-severity-label" style="color: var(--severity-${comment.severity})">${capitalize(comment.severity)}</span></div>`;
  html += `<div class="comment-edit-actions">`;
  html += `<button class="btn btn-sm btn-secondary edit-comment-btn" ` +
    `data-file="${escapeAttr(filePath)}" data-line="${lineNum}" data-side="${side}">Edit</button>`;
  html += `<button class="btn btn-sm btn-danger delete-comment-btn" ` +
    `data-file="${escapeAttr(filePath)}" data-line="${lineNum}" data-side="${side}">Delete</button>`;
  html += `</div></div>`;

  html += `<div class="comment-text">${formatCommentText(comment.text)}</div>`;
  html += `</div></td></tr>`;
  return html;
}

function attachDiffListeners() {
  // Add comment button clicks
  document.querySelectorAll('.add-comment-btn').forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.stopPropagation();
      const row = btn.closest('.diff-line');
      const filePath = row.dataset.file;
      const lineNum = parseInt(row.dataset.line);
      const side = row.dataset.side;

      // If there's already a comment, edit it
      const existing = findComment(filePath, lineNum, side);
      if (existing) {
        state.editingComment = { file: filePath, lineStart: lineNum, side };
      } else {
        state.editingComment = { file: filePath, lineStart: lineNum, side };
      }
      renderDiff();

      // Focus textarea
      setTimeout(() => {
        const textarea = document.querySelector('.comment-textarea');
        if (textarea) textarea.focus();
      }, 0);
    });
  });

  // Severity pill clicks
  document.querySelectorAll('.severity-pill').forEach(pill => {
    pill.addEventListener('click', () => {
      document.querySelectorAll('.severity-pill').forEach(p => p.classList.remove('active'));
      pill.classList.add('active');
    });
  });

  // Save comment
  document.querySelectorAll('.save-comment-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      const filePath = btn.dataset.file;
      const lineNum = parseInt(btn.dataset.line);
      const side = btn.dataset.side;
      const textarea = btn.closest('.comment-card').querySelector('.comment-textarea');
      const text = textarea.value.trim();
      if (!text) return;

      const activePill = btn.closest('.comment-card').querySelector('.severity-pill.active');
      const severity = activePill ? activePill.dataset.severity : 'suggestion';

      saveComment(filePath, lineNum, side, severity, text);
    });
  });

  // Cancel comment
  document.querySelectorAll('.cancel-comment-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      state.editingComment = null;
      renderDiff();
    });
  });

  // Edit comment
  document.querySelectorAll('.edit-comment-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      const filePath = btn.dataset.file;
      const lineNum = parseInt(btn.dataset.line);
      const side = btn.dataset.side;
      state.editingComment = { file: filePath, lineStart: lineNum, side };
      renderDiff();
      setTimeout(() => {
        const textarea = document.querySelector('.comment-textarea');
        if (textarea) textarea.focus();
      }, 0);
    });
  });

  // Delete comment
  document.querySelectorAll('.delete-comment-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      const filePath = btn.dataset.file;
      const lineNum = parseInt(btn.dataset.line);
      const side = btn.dataset.side;
      deleteComment(filePath, lineNum, side);
    });
  });

  // Auto-expand textareas
  document.querySelectorAll('.comment-textarea').forEach(textarea => {
    textarea.addEventListener('input', () => autoExpandTextarea(textarea));
    // Run once for pre-filled text (editing existing comment)
    autoExpandTextarea(textarea);
  });
}

function autoExpandTextarea(textarea) {
  textarea.style.height = 'auto';
  textarea.style.height = textarea.scrollHeight + 'px';
}

// --- Comment Management ---

function findComment(filePath, lineNum, side) {
  return state.comments.find(c =>
    c.file === filePath && c.line_start === lineNum && c.side === side
  ) || null;
}

function saveComment(filePath, lineNum, side, severity, text) {
  // Remove existing comment at this location if any
  state.comments = state.comments.filter(c =>
    !(c.file === filePath && c.line_start === lineNum && c.side === side)
  );

  state.comments.push({
    file: filePath,
    line_start: lineNum,
    line_end: lineNum,
    side: side,
    severity: severity,
    text: text,
  });

  state.editingComment = null;
  updateCommentCount();
  renderFileTree();
  renderDiff();
}

function deleteComment(filePath, lineNum, side) {
  state.comments = state.comments.filter(c =>
    !(c.file === filePath && c.line_start === lineNum && c.side === side)
  );
  state.editingComment = null;
  updateCommentCount();
  renderFileTree();
  renderDiff();
}

// --- Submit ---

async function submitReview(approved) {
  const summary = document.getElementById('summary-input').value.trim();

  const body = {
    comments: state.comments,
    summary: summary,
    approved: approved,
  };

  try {
    const resp = await fetch('/api/submit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    if (!resp.ok) {
      alert('Failed to submit review');
      return;
    }

    // Show submitted overlay
    const overlay = document.createElement('div');
    overlay.className = 'submitted-overlay';
    overlay.innerHTML = `<div class="submitted-message">
      <h2>${approved ? 'Review Approved' : 'Review Submitted'}</h2>
      <p>${state.comments.length} comment${state.comments.length !== 1 ? 's' : ''}${summary ? ' with summary' : ''}</p>
      <p style="margin-top: 12px; color: var(--text-subtle)">You can close this tab</p>
    </div>`;
    document.body.appendChild(overlay);
  } catch (err) {
    alert('Error submitting review: ' + err.message);
  }
}

// --- Text Formatting ---

function formatCommentText(text) {
  const parts = [];
  const regex = /```(\w*)\n([\s\S]*?)```/g;
  let lastIndex = 0;
  let match;

  while ((match = regex.exec(text)) !== null) {
    // Text before the code block
    if (match.index > lastIndex) {
      parts.push(`<span>${escapeHtml(text.slice(lastIndex, match.index))}</span>`);
    }
    const lang = match[1];
    const code = match[2];
    const cls = lang ? ` class="language-${escapeAttr(lang)}"` : '';
    parts.push(`<pre><code${cls}>${escapeHtml(code)}</code></pre>`);
    lastIndex = match.index + match[0].length;
  }

  // Remaining text after last code block
  if (lastIndex < text.length) {
    parts.push(`<span>${escapeHtml(text.slice(lastIndex))}</span>`);
  }

  return parts.join('');
}

// --- Utilities ---

function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

function escapeAttr(str) {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;');
}

function capitalize(str) {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

const IMAGE_EXTENSIONS = new Set(['.png', '.jpg', '.jpeg', '.gif', '.svg', '.webp', '.bmp', '.ico', '.avif']);

// --- Syntax Highlighting ---

const FILE_EXT_TO_LANGUAGE = {
  '.go': 'go', '.js': 'javascript', '.jsx': 'javascript', '.ts': 'typescript', '.tsx': 'typescript',
  '.py': 'python', '.rb': 'ruby', '.rs': 'rust', '.java': 'java', '.kt': 'kotlin',
  '.c': 'c', '.h': 'c', '.cpp': 'cpp', '.hpp': 'cpp', '.cs': 'csharp',
  '.swift': 'swift', '.m': 'objectivec', '.php': 'php', '.pl': 'perl',
  '.sh': 'bash', '.bash': 'bash', '.zsh': 'bash', '.fish': 'bash',
  '.html': 'xml', '.htm': 'xml', '.xml': 'xml', '.svg': 'xml',
  '.css': 'css', '.scss': 'scss', '.less': 'less',
  '.json': 'json', '.yaml': 'yaml', '.yml': 'yaml', '.toml': 'ini',
  '.md': 'markdown', '.sql': 'sql', '.graphql': 'graphql',
  '.proto': 'protobuf', '.lua': 'lua', '.r': 'r', '.R': 'r',
  '.ex': 'elixir', '.erl': 'erlang', '.hs': 'haskell', '.ml': 'ocaml',
  '.scala': 'scala', '.clj': 'clojure', '.mk': 'makefile', '.cmake': 'cmake',
};

const FILE_NAME_TO_LANGUAGE = {
  'Makefile': 'makefile', 'Dockerfile': 'dockerfile',
};

function getLanguageFromPath(filePath) {
  const name = filePath.split('/').pop();
  if (FILE_NAME_TO_LANGUAGE[name]) return FILE_NAME_TO_LANGUAGE[name];
  const dot = name.lastIndexOf('.');
  if (dot === -1) return null;
  return FILE_EXT_TO_LANGUAGE[name.substring(dot)] || null;
}

function highlightLine(content, language) {
  if (language && typeof hljs !== 'undefined') {
    try {
      return hljs.highlight(content, { language, ignoreIllegals: true }).value;
    } catch (e) {
      // Language not registered or other error — fall back
    }
  }
  return escapeHtml(content);
}

function isImageFile(path) {
  const ext = path.substring(path.lastIndexOf('.')).toLowerCase();
  return IMAGE_EXTENSIONS.has(ext);
}

// --- Start ---

init();
