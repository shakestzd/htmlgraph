/* ── Application state & data fetching ─────────────────────── */

var events = [];
var sessions = [];
var features = [];
var plans = [];
var stats = {};
var currentView = 'activity';
var seenEventIds = new Set();
var groupByTrack = localStorage.getItem('htmlgraph-kanban-group-by-track') === 'true';

/* ── Navigation ────────────────────────────────────────────── */
document.querySelector('.nav').addEventListener('click', function(e) {
  var btn = e.target.closest('.nav-btn');
  if (!btn) return;
  var view = btn.dataset.view;
  if (view === currentView) return;
  currentView = view;
  document.querySelectorAll('.nav-btn').forEach(function(b) { b.classList.toggle('active', b === btn); });
  document.querySelectorAll('.view').forEach(function(v) { v.classList.toggle('active', v.id === 'v-' + view); });
  if (view === 'sessions' && sessions.length === 0) fetchSessions();
  if (view === 'work' && features.length === 0) fetchFeatures();
  if (view === 'plans' && plans.length === 0) fetchPlans();
});

/* ── Data fetching ─────────────────────────────────────────── */
function fetchStats() {
  return fetch('/api/stats').then(function(r) {
    if (!r.ok) return;
    return r.json().then(function(data) {
      stats = data;
      updateStatsBar();
    });
  }).catch(function() {});
}

function formatCost(val) {
  if (val >= 1000) return '$' + (val / 1000).toFixed(1) + 'k';
  if (val >= 1) return '$' + val.toFixed(0);
  return '$' + val.toFixed(2);
}

function updateStatsBar() {
  setVal('sv-live', stats.live_sessions);
  setVal('sv-feat-ip', stats.features_in_progress);
  setVal('sv-done-today', '+' + (stats.done_today || 0));
  setVal('sv-cost', formatCost(stats.cost_today || 0));
  var errPill = document.getElementById('sp-errors');
  if (errPill) {
    if (stats.errors_today > 0) {
      errPill.style.display = '';
      setVal('sv-errors', stats.errors_today);
    } else {
      errPill.style.display = 'none';
    }
  }
}

function fetchEvents() {
  return fetch('/api/events/recent?limit=100').then(function(r) {
    if (!r.ok) return;
    return r.json().then(function(data) {
      events = data;
      events.forEach(function(e) { seenEventIds.add(e.event_id); });
    });
  }).catch(function() {});
}

function fetchSessions() {
  return fetch('/api/sessions').then(function(r) {
    if (!r.ok) return;
    return r.json().then(function(data) {
      sessions = data;
      renderSessions();
    });
  }).catch(function() {});
}

function fetchFeatures() {
  return fetch('/api/features').then(function(r) {
    if (!r.ok) return;
    return r.json().then(function(data) {
      features = data;
      renderKanban();
    });
  }).catch(function() {});
}

function fetchPlans() {
  fetch('/api/plans')
    .then(function(r) { return r.json(); })
    .then(function(data) {
      plans = data || [];
      renderPlans();
      var pending = plans.filter(function(p) { return p.status !== 'finalized'; }).length;
      var pill = document.getElementById('sp-plans');
      if (pill && pending > 0) {
        pill.style.display = '';
        document.getElementById('sv-plans').textContent = pending;
      }
    })
    .catch(function() {
      plans = [];
      renderPlans();
    });
}

function renderPlans(filteredPlans) {
  var items = filteredPlans || plans;
  var body = document.getElementById('plans-body');
  var empty = document.getElementById('plans-empty');
  document.getElementById('plans-count').textContent = plans.length;

  if (items.length === 0) {
    body.innerHTML = '';
    empty.style.display = 'block';
    return;
  }
  empty.style.display = 'none';

  body.innerHTML = '';
  items.forEach(function(p) {
    var tr = document.createElement('tr');
    tr.style.cursor = 'pointer';
    tr.addEventListener('click', function() {
      openPlanDetail(p.id, p.title);
    });

    // Title
    tr.appendChild(td(p.title));

    // Status badge
    var statusClass = p.status === 'finalized' ? 'badge-done' :
                      p.status === 'in-progress' ? 'badge-ip' : 'badge-todo';
    var statusText = p.status === 'finalized' ? 'Finalized' :
                     p.status === 'in-progress' ? 'In Progress' : 'Draft';
    tr.appendChild(tdWithChild(createBadge(statusText, statusClass)));

    // Progress bar
    var pct = p.total > 0 ? Math.round(p.approved / p.total * 100) : 0;
    var progTd = document.createElement('td');
    var progWrap = document.createElement('div');
    progWrap.style.cssText = 'display:flex;align-items:center;gap:8px;';
    var progTrack = document.createElement('div');
    progTrack.style.cssText = 'flex:1;height:6px;background:var(--bg-tertiary);border-radius:3px;overflow:hidden;';
    var progFill = document.createElement('div');
    progFill.style.cssText = 'height:100%;border-radius:3px;background:' +
      (pct === 100 ? 'var(--status-done)' : 'var(--accent)') + ';width:' + pct + '%;';
    progTrack.appendChild(progFill);
    progWrap.appendChild(progTrack);
    var progLabel = document.createElement('span');
    progLabel.style.cssText = 'font-size:0.75rem;color:var(--text-secondary);white-space:nowrap;';
    progLabel.textContent = p.approved + '/' + p.total;
    progWrap.appendChild(progLabel);
    progTd.appendChild(progWrap);
    tr.appendChild(progTd);

    // Linked feature/track
    tr.appendChild(td(p.feature_id || '\u2014'));

    // Updated
    tr.appendChild(td(relTime(p.updated_at)));

    // Delete button (only for non-finalized plans)
    var delTd = document.createElement('td');
    delTd.style.cssText = 'white-space:nowrap;';
    if (p.status !== 'finalized') {
      (function(planId, planTitle, td) {
        var btnStyle = 'background:transparent;border:1px solid var(--border);color:var(--text-muted);padding:3px 10px;border-radius:4px;font-size:0.7rem;cursor:pointer;font-family:var(--font-sans,inherit);';
        var delBtn = document.createElement('button');
        delBtn.textContent = 'Delete';
        delBtn.style.cssText = btnStyle;
        delBtn.addEventListener('mouseenter', function() { this.style.borderColor='#dc2626'; this.style.color='#dc2626'; });
        delBtn.addEventListener('mouseleave', function() { this.style.borderColor=''; this.style.color='var(--text-muted)'; });
        delBtn.addEventListener('click', function(e) {
          e.stopPropagation();
          td.innerHTML = '';
          var cancelBtn = document.createElement('button');
          cancelBtn.textContent = 'Cancel';
          cancelBtn.style.cssText = btnStyle + 'margin-right:6px;';
          cancelBtn.addEventListener('click', function(e2) {
            e2.stopPropagation();
            td.innerHTML = '';
            td.appendChild(delBtn);
          });
          var confirmBtn = document.createElement('button');
          confirmBtn.textContent = 'Confirm';
          confirmBtn.style.cssText = 'background:#dc2626;border:1px solid #dc2626;color:#fff;padding:3px 10px;border-radius:4px;font-size:0.7rem;cursor:pointer;font-family:var(--font-sans,inherit);';
          confirmBtn.addEventListener('click', function(e2) {
            e2.stopPropagation();
            confirmBtn.textContent = 'Deleting...';
            confirmBtn.disabled = true;
            fetch('/api/plans/' + planId + '/delete', { method: 'DELETE' })
              .then(function(r) { return r.json(); })
              .then(function() { plans = []; fetchPlans(); });
          });
          td.appendChild(cancelBtn);
          td.appendChild(confirmBtn);
        });
        td.appendChild(delBtn);
      })(p.id, p.title, delTd);
    }
    tr.appendChild(delTd);

    body.appendChild(tr);
  });
}

/* ── Rendering: Sessions ───────────────────────────────────── */
function sessionSparkline(msgCount) {
  var maxMsgs = 100;
  var w = Math.min(50, Math.max(4, (msgCount / maxMsgs) * 50));
  var svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
  svg.setAttribute('width', '50');
  svg.setAttribute('height', '16');
  svg.setAttribute('viewBox', '0 0 50 16');
  var rect = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
  rect.setAttribute('x', '0');
  rect.setAttribute('y', '4');
  rect.setAttribute('width', String(w));
  rect.setAttribute('height', '8');
  rect.setAttribute('rx', '2');
  rect.setAttribute('fill', 'var(--accent)');
  rect.setAttribute('opacity', '0.5');
  svg.appendChild(rect);
  return svg;
}

function renderSessions() {
  var body = document.getElementById('sessions-body');
  var empty = document.getElementById('sessions-empty');
  document.getElementById('sessions-count').textContent = sessions.length;
  body.textContent = '';
  if (sessions.length === 0) { empty.style.display = ''; return; }
  empty.style.display = 'none';

  // Pin live sessions to top, then sort by created_at DESC
  var sorted = sessions.slice().sort(function(a, b) {
    var aLive = a.status === 'active' ? 1 : 0;
    var bLive = b.status === 'active' ? 1 : 0;
    if (bLive !== aLive) return bLive - aLive;
    return (b.created_at || '') > (a.created_at || '') ? 1 : -1;
  });

  var frag = document.createDocumentFragment();
  sorted.forEach(function(s) {
    var tr = document.createElement('tr');
    tr.className = 'session-row' + (s.status === 'active' ? ' live' : '');
    tr.addEventListener('click', function() { openTranscript(s.session_id); });

    // Title cell
    var titleTd = document.createElement('td');
    var titleSpan = document.createElement('span');
    titleSpan.className = 'session-title';
    titleSpan.textContent = sessionDisplayTitle(s);
    titleSpan.title = s.first_message || s.session_id;
    titleTd.appendChild(titleSpan);
    if (s.launch_mode === 'yolo') {
      var yoloBadge = document.createElement('span');
      yoloBadge.className = 'badge-yolo';
      yoloBadge.textContent = 'YOLO';
      yoloBadge.style.marginLeft = '6px';
      titleTd.appendChild(yoloBadge);
    }
    tr.appendChild(titleTd);

    // Model cell
    var modelTd = document.createElement('td');
    modelTd.className = 'mono';
    modelTd.textContent = s.model || '--';
    tr.appendChild(modelTd);

    // Msgs cell
    tr.appendChild(td(s.message_count ? String(s.message_count) : '--', { className: 'mono' }));

    // Activity sparkline cell
    var sparkTd = document.createElement('td');
    sparkTd.appendChild(sessionSparkline(s.message_count || 0));
    tr.appendChild(sparkTd);

    // Status cell
    var statusTd = document.createElement('td');
    if (s.status === 'active') {
      var liveBadge = document.createElement('span');
      liveBadge.className = 'badge-live';
      liveBadge.textContent = 'LIVE';
      statusTd.appendChild(liveBadge);
    } else {
      var endedBadge = document.createElement('span');
      endedBadge.className = 'badge badge-ended';
      endedBadge.textContent = s.status || 'ended';
      statusTd.appendChild(endedBadge);
    }
    tr.appendChild(statusTd);

    // Time cell
    tr.appendChild(td(relTime(s.created_at), { className: 'mono' }));

    frag.appendChild(tr);
  });
  body.appendChild(frag);
}

/* ── Rendering: Work (Kanban) ──────────────────────────────── */
var PRIORITY_ORDER = { critical: 0, high: 1, medium: 2, low: 3 };

var TYPE_ICONS = {
  feat:  '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><polyline points="20 6 9 17 4 12"/></svg>',
  bug:   '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>',
  spk:   '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z"/></svg>',
  track: '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="2" y="3" width="20" height="4" rx="1"/><rect x="2" y="10" width="20" height="4" rx="1"/><rect x="2" y="17" width="20" height="4" rx="1"/></svg>'
};

var COL_DEFS = [
  { status: 'todo',        label: 'Todo' },
  { status: 'in-progress', label: 'In Progress' },
  { status: 'done',        label: 'Done' }
];

function itemTypeKey(id) {
  if (!id) return 'feat';
  if (id.startsWith('bug')) return 'bug';
  if (id.startsWith('spk')) return 'spk';
  if (id.startsWith('trk')) return 'track';
  return 'feat';
}

function sortItems(items) {
  return items.slice().sort(function(a, b) {
    var pa = PRIORITY_ORDER[a.priority] != null ? PRIORITY_ORDER[a.priority] : 2;
    var pb = PRIORITY_ORDER[b.priority] != null ? PRIORITY_ORDER[b.priority] : 2;
    if (pa !== pb) return pa - pb;
    return (b.created_at || '') > (a.created_at || '') ? 1 : -1;
  });
}

function buildKanbanCard(f) {
  var card = document.createElement('div');
  card.className = 'kanban-card';
  card.dataset.itemId = f.id;
  card.style.cursor = 'pointer';
  card.addEventListener('click', function() { openWorkDetail(f.id); });

  var titleEl = document.createElement('div');
  titleEl.className = 'kanban-card-title';
  titleEl.textContent = f.title || f.id;
  titleEl.title = f.title || f.id;
  card.appendChild(titleEl);

  var meta = document.createElement('div');
  meta.className = 'kanban-card-meta';

  var typeKey = itemTypeKey(f.id);
  var iconHtml = TYPE_ICONS[typeKey] || TYPE_ICONS.feat;
  var iconWrap = document.createElement('span');
  iconWrap.className = 'type-icon';
  iconWrap.innerHTML = iconHtml;
  meta.appendChild(iconWrap);

  var idSpan = document.createElement('span');
  idSpan.textContent = f.id ? f.id.slice(0, 12) : '--';
  meta.appendChild(idSpan);

  if (f.priority) {
    var priBadge = createPriorityBadge(f.priority);
    meta.appendChild(priBadge);
  }

  card.appendChild(meta);
  return card;
}

function buildKanbanColumns(items) {
  var buckets = { 'todo': [], 'in-progress': [], 'done': [] };
  items.forEach(function(f) {
    var s = f.status || 'todo';
    if (!buckets[s]) s = 'todo';
    buckets[s].push(f);
  });

  var frag = document.createDocumentFragment();
  COL_DEFS.forEach(function(col) {
    var sorted = sortItems(buckets[col.status] || []);
    var colEl = document.createElement('div');
    colEl.className = 'kanban-col';
    colEl.dataset.status = col.status;

    var header = document.createElement('div');
    header.className = 'kanban-col-header';
    var labelSpan = document.createElement('span');
    labelSpan.textContent = col.label;
    header.appendChild(labelSpan);
    var countBadge = document.createElement('span');
    countBadge.className = 'col-count';
    countBadge.textContent = sorted.length;
    header.appendChild(countBadge);
    colEl.appendChild(header);

    var cardsEl = document.createElement('div');
    cardsEl.className = 'kanban-cards';
    sorted.forEach(function(f) { cardsEl.appendChild(buildKanbanCard(f)); });
    colEl.appendChild(cardsEl);
    frag.appendChild(colEl);
  });
  return frag;
}

function buildTrackSection(trackId, trackTitle, items) {
  var doneCount = items.filter(function(f) { return f.status === 'done'; }).length;
  var collapseKey = 'htmlgraph-track-collapsed-' + trackId;
  var isCollapsed = localStorage.getItem(collapseKey) === 'true';

  var section = document.createElement('div');
  section.className = 'track-section';

  var sectionHeader = document.createElement('div');
  sectionHeader.className = 'track-section-header';

  var titleSpan = document.createElement('span');
  titleSpan.className = 'track-section-title';
  titleSpan.textContent = trackTitle || trackId;
  sectionHeader.appendChild(titleSpan);

  var progressSpan = document.createElement('span');
  progressSpan.className = 'track-section-progress';
  progressSpan.textContent = doneCount + '/' + items.length + ' done';
  sectionHeader.appendChild(progressSpan);

  var chevron = document.createElement('span');
  chevron.className = 'track-section-toggle' + (isCollapsed ? ' collapsed' : '');
  chevron.textContent = '\u25BE';
  sectionHeader.appendChild(chevron);

  var body = document.createElement('div');
  body.className = 'track-section-body kanban-board' + (isCollapsed ? ' collapsed' : '');
  body.appendChild(buildKanbanColumns(items));

  sectionHeader.addEventListener('click', function() {
    isCollapsed = !isCollapsed;
    chevron.classList.toggle('collapsed', isCollapsed);
    body.classList.toggle('collapsed', isCollapsed);
    localStorage.setItem(collapseKey, isCollapsed ? 'true' : 'false');
  });

  section.appendChild(sectionHeader);
  section.appendChild(body);
  return section;
}

function renderKanban() {
  var board = document.getElementById('kanban-board');
  var empty = document.getElementById('work-empty');
  document.getElementById('work-count').textContent = features.length;
  board.textContent = '';

  var toggleBtn = document.getElementById('track-group-toggle');
  if (toggleBtn) toggleBtn.classList.toggle('active', groupByTrack);

  if (features.length === 0) { empty.style.display = ''; return; }
  empty.style.display = 'none';

  var frag = document.createDocumentFragment();

  if (!groupByTrack) {
    frag.appendChild(buildKanbanColumns(features));
  } else {
    var trackMap = {};
    var trackOrder = [];
    var untracked = [];

    features.forEach(function(f) {
      if (!f.track_id) { untracked.push(f); return; }
      if (!trackMap[f.track_id]) {
        trackMap[f.track_id] = { title: f.track_title || f.track_id, items: [] };
        trackOrder.push(f.track_id);
      }
      trackMap[f.track_id].items.push(f);
    });

    trackOrder.forEach(function(tid) {
      var t = trackMap[tid];
      frag.appendChild(buildTrackSection(tid, t.title, t.items));
    });

    if (untracked.length > 0) {
      frag.appendChild(buildTrackSection('untracked', 'Untracked', untracked));
    }
  }

  board.appendChild(frag);
}

/* ── Rendering: Agents ─────────────────────────────────────── */
function renderAgents() {
  var body = document.getElementById('agents-body');
  var empty = document.getElementById('agents-empty');
  body.textContent = '';
  if (events.length === 0) {
    empty.style.display = '';
    document.getElementById('agents-count').textContent = '0';
    return;
  }
  empty.style.display = 'none';

  var agentMap = {};
  events.forEach(function(e) {
    var aid = e.agent_id || 'unknown';
    if (!agentMap[aid]) agentMap[aid] = { count: 0, lastTs: '', tools: {} };
    agentMap[aid].count++;
    if (e.timestamp > agentMap[aid].lastTs) agentMap[aid].lastTs = e.timestamp;
    var tool = e.tool_name || e.event_type || 'other';
    agentMap[aid].tools[tool] = (agentMap[aid].tools[tool] || 0) + 1;
  });

  var sorted = Object.keys(agentMap).map(function(k) { return [k, agentMap[k]]; })
    .sort(function(a, b) { return b[1].count - a[1].count; });
  document.getElementById('agents-count').textContent = sorted.length;

  var frag = document.createDocumentFragment();
  sorted.forEach(function(pair) {
    var aid = pair[0];
    var info = pair[1];
    var topTools = Object.keys(info.tools).map(function(t) { return [t, info.tools[t]]; })
      .sort(function(a, b) { return b[1] - a[1]; })
      .slice(0, 4)
      .map(function(pair) { return pair[0] + '(' + pair[1] + ')'; })
      .join(', ');
    var tr = document.createElement('tr');
    tr.appendChild(td(aid, { style: 'color:var(--text-primary);font-weight:500' }));
    tr.appendChild(td(String(info.count)));
    tr.appendChild(td(relTime(info.lastTs), { className: 'mono' }));
    tr.appendChild(td(topTools, { className: 'ellipsis', style: 'color:var(--text-muted)' }));
    frag.appendChild(tr);
  });
  body.appendChild(frag);
}

/* ── Rendering: Metrics ────────────────────────────────────── */
function renderMetrics() {
  var emptyEl = document.getElementById('metrics-empty');
  if (events.length === 0) { emptyEl.style.display = ''; return; }
  emptyEl.style.display = 'none';
  renderBarChart('chart-tools', bucketBy(events, function(e) { return e.tool_name || e.event_type || 'other'; }));
  renderBarChart('chart-agents', bucketBy(events, function(e) { return e.agent_id || 'unknown'; }));
  renderHoursChart();
}

function bucketBy(arr, keyFn) {
  var m = {};
  arr.forEach(function(e) { var k = keyFn(e); m[k] = (m[k] || 0) + 1; });
  return Object.keys(m).map(function(k) { return [k, m[k]]; })
    .sort(function(a, b) { return b[1] - a[1]; });
}

function renderBarChart(containerId, entries) {
  var el = document.getElementById(containerId);
  el.textContent = '';
  if (entries.length === 0) {
    var msg = document.createElement('div');
    msg.className = 'empty';
    msg.textContent = 'No data';
    el.appendChild(msg);
    return;
  }
  var maxVal = entries[0][1];
  var frag = document.createDocumentFragment();
  entries.slice(0, 15).forEach(function(pair) {
    var label = pair[0];
    var count = pair[1];
    var pct = maxVal > 0 ? (count / maxVal) * 100 : 0;
    var row = document.createElement('div');
    row.className = 'bar-row';
    var lblSpan = document.createElement('span');
    lblSpan.className = 'label';
    lblSpan.title = label;
    lblSpan.textContent = label;
    row.appendChild(lblSpan);
    var track = document.createElement('div');
    track.className = 'bar-track';
    var fill = document.createElement('div');
    fill.className = 'bar-fill';
    fill.style.width = pct + '%';
    track.appendChild(fill);
    row.appendChild(track);
    var valSpan = document.createElement('span');
    valSpan.className = 'val';
    valSpan.textContent = count;
    row.appendChild(valSpan);
    frag.appendChild(row);
  });
  el.appendChild(frag);
}

function renderHoursChart() {
  var now = Date.now();
  var keys = [];
  var buckets = {};
  for (var h = 23; h >= 0; h--) {
    var d = new Date(now - h * 3600000);
    var key = String(d.getHours()).padStart(2, '0') + ':00';
    keys.push(key);
    buckets[key] = 0;
  }
  events.forEach(function(e) {
    if (!e.timestamp) return;
    var d = new Date(e.timestamp.indexOf('T') >= 0 ? e.timestamp : e.timestamp.replace(' ', 'T') + 'Z');
    if (now - d.getTime() > 86400000) return;
    var key = String(d.getHours()).padStart(2, '0') + ':00';
    if (key in buckets) buckets[key]++;
  });
  var entries = keys.map(function(k) { return [k, buckets[k]]; });
  renderBarChart('chart-hours', entries);
}

/* ── Work item detail panel ────────────────────────────────── */
function closeWorkDetail() {
  var detail = document.getElementById('work-detail');
  var board = document.getElementById('kanban-board');
  var empty = document.getElementById('work-empty');
  var viewTitle = document.querySelector('#v-work .view-title');
  detail.classList.remove('active');
  board.style.display = '';
  if (features.length === 0) empty.style.display = '';
  if (viewTitle) viewTitle.style.display = '';
}

function openWorkDetail(id) {
  var detail = document.getElementById('work-detail');
  var board = document.getElementById('kanban-board');
  var empty = document.getElementById('work-empty');
  var content = document.getElementById('work-detail-content');
  var viewTitle = document.querySelector('#v-work .view-title');

  // Hide board, show detail panel
  board.style.display = 'none';
  empty.style.display = 'none';
  if (viewTitle) viewTitle.style.display = 'none';
  detail.classList.add('active');
  content.textContent = '';

  // Loading indicator
  var loading = document.createElement('div');
  loading.className = 'empty';
  loading.textContent = 'Loading...';
  content.appendChild(loading);

  fetch('/api/features/detail?id=' + encodeURIComponent(id))
    .then(function(r) {
      if (!r.ok) throw new Error('Not found');
      return r.json();
    })
    .then(function(node) {
      content.textContent = '';
      renderWorkDetail(content, node);
    })
    .catch(function() {
      content.textContent = '';
      var err = document.createElement('div');
      err.className = 'empty';
      err.textContent = 'Could not load item: ' + id;
      content.appendChild(err);
    });
}

function renderWorkDetail(container, node) {
  // Type badge + title
  var typeKey = itemTypeKey(node.id);
  var typeBadge = document.createElement('span');
  typeBadge.className = 'badge badge-' + (typeKey === 'feat' ? 'ip' : typeKey === 'bug' ? 'error' : 'todo');
  typeBadge.style.marginBottom = '8px';
  typeBadge.style.display = 'inline-block';
  typeBadge.textContent = typeKey.toUpperCase();
  container.appendChild(typeBadge);

  var titleEl = document.createElement('h2');
  titleEl.className = 'work-detail-title';
  titleEl.textContent = node.title || node.id;
  container.appendChild(titleEl);

  var idEl = document.createElement('div');
  idEl.className = 'work-detail-id';
  idEl.textContent = node.id;
  container.appendChild(idEl);

  // Status + priority badges
  var badges = document.createElement('div');
  badges.className = 'work-detail-badges';
  if (node.status) {
    var statusBadge = document.createElement('span');
    var statusClass = node.status === 'in-progress' ? 'ip' : node.status === 'done' ? 'done' : 'todo';
    statusBadge.className = 'badge badge-' + statusClass;
    statusBadge.textContent = node.status;
    badges.appendChild(statusBadge);
  }
  if (node.priority) {
    var priBadge = createPriorityBadge(node.priority);
    badges.appendChild(priBadge);
  }
  if (badges.childNodes.length > 0) container.appendChild(badges);

  // Content / findings
  if (node.content) {
    var contentSection = document.createElement('div');
    contentSection.className = 'work-detail-section';
    var contentTitle = document.createElement('div');
    contentTitle.className = 'work-detail-section-title';
    contentTitle.textContent = 'Findings';
    contentSection.appendChild(contentTitle);
    var contentBody = document.createElement('div');
    contentBody.className = 'work-detail-content';
    contentBody.textContent = node.content;
    contentSection.appendChild(contentBody);
    container.appendChild(contentSection);
  }

  // Track info
  if (node.track_id) {
    var trackSection = document.createElement('div');
    trackSection.className = 'work-detail-section';
    var trackLabel = document.createElement('div');
    trackLabel.className = 'work-detail-section-title';
    trackLabel.textContent = 'Track';
    trackSection.appendChild(trackLabel);
    var trackLink = document.createElement('div');
    trackLink.className = 'work-detail-track-link';
    trackLink.textContent = node.track_id;
    trackLink.addEventListener('click', function() { openWorkDetail(node.track_id); });
    trackSection.appendChild(trackLink);
    container.appendChild(trackSection);
  }

  // Steps
  if (node.steps && node.steps.length > 0) {
    var stepsSection = document.createElement('div');
    stepsSection.className = 'work-detail-section';
    var stepsLabel = document.createElement('div');
    stepsLabel.className = 'work-detail-section-title';
    stepsLabel.textContent = 'Steps (' + node.steps.filter(function(s) { return s.completed; }).length + '/' + node.steps.length + ')';
    stepsSection.appendChild(stepsLabel);
    var stepsList = document.createElement('ul');
    stepsList.className = 'work-detail-steps';
    node.steps.forEach(function(step) {
      var li = document.createElement('li');
      var icon = document.createElement('span');
      if (step.completed) {
        icon.className = 'step-done';
        icon.textContent = '\u2713';
      } else {
        icon.className = 'step-pending';
        icon.textContent = '\u25CB';
      }
      li.appendChild(icon);
      var text = document.createElement('span');
      text.textContent = step.description || step.step_id || '';
      if (step.completed) text.style.textDecoration = 'line-through';
      li.appendChild(text);
      stepsList.appendChild(li);
    });
    stepsSection.appendChild(stepsList);
    container.appendChild(stepsSection);
  }

  // Edges
  if (node.edges && Object.keys(node.edges).length > 0) {
    var edgesSection = document.createElement('div');
    edgesSection.className = 'work-detail-section';
    var edgesLabel = document.createElement('div');
    edgesLabel.className = 'work-detail-section-title';
    edgesLabel.textContent = 'Relationships';
    edgesSection.appendChild(edgesLabel);
    var edgesContainer = document.createElement('div');
    edgesContainer.className = 'work-detail-edges';
    Object.keys(node.edges).forEach(function(relType) {
      var edgeList = node.edges[relType];
      if (!Array.isArray(edgeList)) return;
      edgeList.forEach(function(edge) {
        var edgeEl = document.createElement('div');
        edgeEl.className = 'work-detail-edge';
        var typeSpan = document.createElement('span');
        typeSpan.className = 'edge-type';
        typeSpan.textContent = relType.replace(/_/g, ' ');
        edgeEl.appendChild(typeSpan);
        var targetSpan = document.createElement('span');
        targetSpan.className = 'edge-target';
        targetSpan.textContent = (edge.target_id || '').slice(0, 16);
        edgeEl.appendChild(targetSpan);
        if (edge.title) {
          var titleSpan = document.createElement('span');
          titleSpan.className = 'edge-title';
          titleSpan.textContent = edge.title;
          edgeEl.appendChild(titleSpan);
        }
        edgeEl.addEventListener('click', function() {
          if (edge.target_id) openWorkDetail(edge.target_id);
        });
        edgesContainer.appendChild(edgeEl);
      });
    });
    edgesSection.appendChild(edgesContainer);
    container.appendChild(edgesSection);
  }

  // Related features (async) — section only shown when results exist
  var relatedSection = document.createElement('div');
  relatedSection.className = 'work-detail-section';
  var relatedLabel = document.createElement('div');
  relatedLabel.className = 'work-detail-section-title';
  relatedLabel.textContent = 'Related (Shared Files)';
  relatedSection.appendChild(relatedLabel);
  var relatedContainer = document.createElement('div');
  relatedContainer.className = 'work-detail-related';
  relatedSection.appendChild(relatedContainer);

  fetch('/api/features/related?feature_id=' + encodeURIComponent(node.id))
    .then(function(r) { return r.ok ? r.json() : []; })
    .then(function(related) {
      if (!related || related.length === 0) {
        return;
      }
      container.appendChild(relatedSection);
      related.forEach(function(rel) {
        var relEl = document.createElement('div');
        relEl.className = 'work-detail-related-item';
        var idSpan = document.createElement('span');
        idSpan.className = 'rel-id';
        idSpan.textContent = (rel.feature_id || rel.id || '').slice(0, 16);
        relEl.appendChild(idSpan);
        var titleSpan = document.createElement('span');
        titleSpan.className = 'rel-title';
        titleSpan.textContent = rel.title || rel.feature_id || '';
        relEl.appendChild(titleSpan);
        var fid = rel.feature_id || rel.id;
        if (fid) relEl.addEventListener('click', function() { openWorkDetail(fid); });
        relatedContainer.appendChild(relEl);
      });
    })
    .catch(function() { /* no related section on error */ });
}

/* ── Init ──────────────────────────────────────────────────── */
document.addEventListener('DOMContentLoaded', function() {
  var toggleBtn = document.getElementById('track-group-toggle');
  if (toggleBtn) {
    toggleBtn.classList.toggle('active', groupByTrack);
    toggleBtn.addEventListener('click', function() {
      groupByTrack = !groupByTrack;
      localStorage.setItem('htmlgraph-kanban-group-by-track', groupByTrack ? 'true' : 'false');
      renderKanban();
    });
  }

  var backBtn = document.getElementById('work-detail-back');
  if (backBtn) {
    backBtn.addEventListener('click', closeWorkDetail);
  }

  var planBackBtn = document.getElementById('plan-detail-back');
  if (planBackBtn) {
    planBackBtn.addEventListener('click', closePlanDetail);
  }

  var planFilter = document.getElementById('plan-status-filter');
  if (planFilter) {
    planFilter.addEventListener('change', function() {
      var val = this.value;
      renderPlans(val === 'all' ? plans : plans.filter(function(p) { return p.status === val; }));
    });
  }
});

// Theme toggle — single source of truth for dashboard theme
(function() {
  // Clean up stale theme keys from the old plan review system
  localStorage.removeItem('crispi-theme');
  localStorage.removeItem('theme');

  var btn = document.getElementById('theme-toggle');
  if (!btn) return;
  var saved = localStorage.getItem('htmlgraph-theme');
  if (!saved) {
    saved = (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) ? 'light' : 'dark';
    localStorage.setItem('htmlgraph-theme', saved);
  }
  document.documentElement.dataset.theme = saved;
  btn.textContent = saved === 'light' ? '\u263E' : '\u2600';
  btn.addEventListener('click', function() {
    var current = document.documentElement.dataset.theme || 'dark';
    var next = current === 'dark' ? 'light' : 'dark';
    window._htmlgraphTheme = next;
    document.documentElement.dataset.theme = next;
    localStorage.setItem('htmlgraph-theme', next);
    btn.textContent = next === 'light' ? '\u263E' : '\u2600';
  });

  // Re-assert theme after any dynamic content injection (plan scripts may alter it)
  window._htmlgraphTheme = saved;
  var observer = new MutationObserver(function() {
    if (document.documentElement.dataset.theme !== window._htmlgraphTheme) {
      document.documentElement.dataset.theme = window._htmlgraphTheme;
    }
  });
  observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
})();

Promise.all([fetchStats(), fetchEvents()]);
setInterval(fetchStats, 30000);

/* ── Plan detail panel ────────────────────────────────────── */
function closePlanDetail() {
  var detail = document.getElementById('plan-detail');
  var listView = document.getElementById('plans-list-view');
  var viewTitle = document.querySelector('#v-plans .view-title');
  detail.classList.remove('active');
  listView.style.display = '';
  if (viewTitle) viewTitle.style.display = '';
  // Clear plan subnav
  var subnav = document.getElementById('plan-subnav');
  if (subnav) { subnav.classList.remove('active'); subnav.innerHTML = ''; }
}

function openPlanDetail(planId, title) {
  var detail = document.getElementById('plan-detail');
  var listView = document.getElementById('plans-list-view');
  var body = document.getElementById('plan-detail-body');
  var titleEl = document.getElementById('plan-detail-title');
  var viewTitle = document.querySelector('#v-plans .view-title');

  listView.style.display = 'none';
  if (viewTitle) viewTitle.style.display = 'none';
  detail.classList.add('active');
  titleEl.textContent = title || planId;
  body.innerHTML = '<div class="empty">Loading...</div>';

  fetch('/api/plans/' + planId + '/render')
    .then(function(r) {
      if (!r.ok) throw new Error('Not found');
      return r.text();
    })
    .then(function(html) {
      body.innerHTML = html;
      // Scripts via innerHTML don't execute. Load external scripts first
      // (D3, dagre-d3, hljs), then run inline scripts after they're ready.
      var scripts = Array.from(body.querySelectorAll('script'));
      var externals = scripts.filter(function(s) { return !!s.src; });
      var inlines = scripts.filter(function(s) { return !s.src && s.textContent.trim(); });

      // Remove all old script tags
      scripts.forEach(function(s) { s.remove(); });

      // Build plan subnav from section cards in the loaded content
      buildPlanSubnav(body);

      // Load external scripts sequentially, then run inlines
      function loadNext(i) {
        if (i >= externals.length) {
          inlines.forEach(function(oldScript) {
            var s = document.createElement('script');
            s.textContent = oldScript.textContent;
            body.appendChild(s);
          });
          return;
        }
        var s = document.createElement('script');
        s.src = externals[i].src;
        s.onload = function() { loadNext(i + 1); };
        s.onerror = function() { loadNext(i + 1); };
        body.appendChild(s);
      }
      loadNext(0);
    })
    .catch(function() {
      body.innerHTML = '<div class="empty">Could not load plan: ' + planId + '</div>';
    });
}

function buildPlanSubnav(container) {
  var subnav = document.getElementById('plan-subnav');
  if (!subnav) return;
  subnav.innerHTML = '';

  var sections = [];
  var graph = container.querySelector('.dep-graph');
  if (graph) sections.push({ id: graph.id || 'dep-graph', label: 'Graph' });

  container.querySelectorAll('.section-card[id]').forEach(function(el) {
    var summary = el.querySelector('summary span:first-child');
    var label = summary ? summary.textContent.trim() : el.id;
    sections.push({ id: el.id, label: label });
  });

  var progress = container.querySelector('.progress-zone');
  if (progress) sections.push({ id: progress.id || 'feedback-summary', label: 'Progress' });

  // Scroll container is .plan-content inside .plan-detail-body
  var scrollTarget = container.querySelector('.plan-content') || container;

  sections.forEach(function(sec) {
    var a = document.createElement('a');
    a.href = '#';
    a.textContent = sec.label;
    a.addEventListener('click', function(e) {
      e.preventDefault();
      var target = container.querySelector('#' + sec.id);
      if (target) target.scrollIntoView({ behavior: 'smooth', block: 'start' });
      subnav.querySelectorAll('a').forEach(function(l) { l.classList.remove('active'); });
      a.classList.add('active');
    });
    subnav.appendChild(a);
  });

  // Chat link — the chat sidebar is always visible on the right
  var chatSidebar = container.querySelector('.chat-sidebar');
  if (chatSidebar) {
    var chatLink = document.createElement('a');
    chatLink.href = '#';
    chatLink.textContent = 'Chat';
    chatLink.style.color = 'var(--accent)';
    chatLink.addEventListener('click', function(e) {
      e.preventDefault();
      var input = chatSidebar.querySelector('#chat-input');
      if (input) input.focus();
    });
    subnav.appendChild(chatLink);
  }

  subnav.classList.add('active');
}
