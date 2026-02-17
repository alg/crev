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
      html += `<td class="line-content ${lineClass}">${escapeHtml(line.content)}</td>`;
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

  html += `<div class="comment-text">${escapeHtml(comment.text)}</div>`;
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

function isImageFile(path) {
  const ext = path.substring(path.lastIndexOf('.')).toLowerCase();
  return IMAGE_EXTENSIONS.has(ext);
}

// --- Start ---

init();
