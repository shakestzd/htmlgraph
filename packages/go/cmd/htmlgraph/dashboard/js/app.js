/* ── Application state & data fetching ─────────────────────── */

var events = [];
var sessions = [];
var features = [];
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

function updateStatsBar() {
  var todayCount = 0;
  var now = Date.now();
  events.forEach(function(e) {
    if (!e.timestamp) return;
    var ts = e.timestamp.indexOf('T') >= 0 ? e.timestamp : e.timestamp.replace(' ', 'T') + 'Z';
    if (now - new Date(ts).getTime() < 86400000) todayCount++;
  });
  setVal('sv-today', todayCount || stats.total_events);
  setVal('sv-sessions', stats.active_sessions);
  setVal('sv-feat-ip', stats.features_in_progress);
  setVal('sv-feat-done', stats.features_done);
}

function renderTopTools() {
  var container = document.getElementById('sp-top-tools');
  container.textContent = '';
  if (events.length === 0) return;
  var toolMap = {};
  events.forEach(function(e) {
    var t = e.tool_name || e.event_type || 'other';
    toolMap[t] = (toolMap[t] || 0) + 1;
  });
  var sorted = Object.keys(toolMap).map(function(k) { return [k, toolMap[k]]; })
    .sort(function(a, b) { return b[1] - a[1]; }).slice(0, 3);
  var maxVal = sorted.length > 0 ? sorted[0][1] : 1;
  var frag = document.createDocumentFragment();
  sorted.forEach(function(pair) {
    var label = pair[0];
    var count = pair[1];
    var pct = Math.round((count / maxVal) * 100);
    var wrap = document.createElement('div');
    wrap.className = 'stat-mini-bar';
    wrap.title = label + ': ' + count;
    var track = document.createElement('div');
    track.className = 'stat-mini-bar-track';
    var fill = document.createElement('div');
    fill.className = 'stat-mini-bar-fill';
    fill.style.width = pct + '%';
    track.appendChild(fill);
    wrap.appendChild(track);
    var lbl = document.createElement('span');
    lbl.className = 'stat-mini-bar-label';
    lbl.textContent = label.length > 6 ? label.slice(0, 6) : label;
    wrap.appendChild(lbl);
    frag.appendChild(wrap);
  });
  container.appendChild(frag);
}

function renderAgentTooltip() {
  var tooltip = document.getElementById('agents-tooltip');
  tooltip.textContent = '';
  var agentMap = {};
  events.forEach(function(e) {
    var aid = e.agent_id || 'unknown';
    agentMap[aid] = (agentMap[aid] || 0) + 1;
  });
  var sorted = Object.keys(agentMap).map(function(k) { return [k, agentMap[k]]; })
    .sort(function(a, b) { return b[1] - a[1]; });
  var frag = document.createDocumentFragment();
  sorted.forEach(function(pair) {
    var item = document.createElement('div');
    item.className = 'stat-tooltip-item';
    var nameSpan = document.createElement('span');
    nameSpan.textContent = pair[0];
    var countSpan = document.createElement('span');
    countSpan.textContent = pair[1];
    item.appendChild(nameSpan);
    item.appendChild(countSpan);
    frag.appendChild(item);
  });
  tooltip.appendChild(frag);
}

function fetchEvents() {
  return fetch('/api/events/recent?limit=100').then(function(r) {
    if (!r.ok) return;
    return r.json().then(function(data) {
      events = data;
      events.forEach(function(e) { seenEventIds.add(e.event_id); });
      renderAgentCount();
      renderTopTools();
      renderAgentTooltip();
      updateStatsBar();
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

function renderAgentCount() {
  var agents = new Set();
  events.forEach(function(e) { if (e.agent_id) agents.add(e.agent_id); });
  setVal('sv-agents', agents.size);
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
});

Promise.all([fetchStats(), fetchEvents()]);
setInterval(fetchStats, 30000);
