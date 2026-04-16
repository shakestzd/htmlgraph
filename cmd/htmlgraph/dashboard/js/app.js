/* ── Application state & data fetching ─────────────────────── */

var events = [];
var sessions = [];
var features = [];
var plans = [];
var stats = {};
var currentView = 'activity';
var seenEventIds = new Set();
var groupByTrack = localStorage.getItem('htmlgraph-kanban-group-by-track') === 'true';

// Global mode state — populated by detectMode() on init. In single-project
// mode both values stay unset and buildProjectUrl() returns plain URLs.
window.htmlgraphMode = 'single';
window.htmlgraphProjects = [];
window.htmlgraphProjectId = '';

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
  if (view === 'graph') fetchGraph();
});

/* ── Data fetching ─────────────────────────────────────────── */
function fetchStats() {
  // In global mode with no project selected, show aggregate stats.
  var url = (window.htmlgraphMode === 'global' && !window.htmlgraphProjectId)
    ? '/api/projects/all/stats'
    : buildProjectUrl('stats');
  return fetch(url).then(function(r) {
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
  return fetch(buildProjectUrl('events/recent', 'limit=100')).then(function(r) {
    if (!r.ok) return;
    return r.json().then(function(data) {
      events = data;
      events.forEach(function(e) { seenEventIds.add(e.event_id); });
    });
  }).catch(function() {});
}

function fetchSessions() {
  return fetch(buildProjectUrl('sessions')).then(function(r) {
    if (!r.ok) return;
    return r.json().then(function(data) {
      sessions = data;
      renderSessions();
    });
  }).catch(function() {});
}

function fetchFeatures() {
  return fetch(buildProjectUrl('features')).then(function(r) {
    if (!r.ok) return;
    return r.json().then(function(data) {
      features = data;
      renderKanban();
    });
  }).catch(function() {});
}

function fetchPlans() {
  fetch(buildProjectUrl('plans'))
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
            fetch(buildProjectUrl('plans/' + planId + '/delete'), { method: 'DELETE' })
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
    tr.setAttribute('data-session-id', s.session_id);
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
    if (s.plan_id) {
      var planBadge = document.createElement('span');
      planBadge.className = 'badge-plan';
      planBadge.textContent = 'PLAN';
      planBadge.style.marginLeft = '6px';
      planBadge.title = s.plan_id;
      planBadge.addEventListener('click', function(e) {
        e.stopPropagation();
        navigateToPlan(s.plan_id, null);
      });
      titleTd.appendChild(planBadge);
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
  if (viewTitle) viewTitle.style.display = '';
  // If features haven't been loaded yet (e.g. user navigated directly
  // from the graph view), fetch them now instead of showing an empty board.
  if (features.length === 0) {
    fetchFeatures();
  }
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

  fetch(buildProjectUrl('features/detail', 'id=' + encodeURIComponent(id)))
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

  fetch(buildProjectUrl('features/related', 'feature_id=' + encodeURIComponent(node.id)))
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

  // Activity data — async, three collapsible panels: Commits / Files / Activity
  fetch(buildProjectUrl('features/' + encodeURIComponent(node.id) + '/activity'))
    .then(function(r) { return r.ok ? r.json() : null; })
    .then(function(data) {
      if (!data) return;

      var hasCommits  = data.commits    && data.commits.length > 0;
      var hasFiles    = data.file_edits && data.file_edits.length > 0;
      var hasActivity = data.total_events > 0;
      if (!hasCommits && !hasFiles && !hasActivity) return;

      // helper: build a collapsible section panel
      function makePanel(headerText, count) {
        var section = document.createElement('div');
        section.className = 'work-detail-section activity-panel';

        var header = document.createElement('div');
        header.className = 'work-detail-section-title activity-panel-header';
        header.style.cssText = 'cursor:pointer;display:flex;align-items:center;justify-content:space-between;';

        var labelSpan = document.createElement('span');
        labelSpan.textContent = headerText + (count != null ? ' (' + count + ')' : '');
        header.appendChild(labelSpan);

        var chevron = document.createElement('span');
        chevron.textContent = '\u25BE';
        chevron.style.cssText = 'font-size:0.8em;margin-left:6px;transition:transform 0.15s;';
        header.appendChild(chevron);

        var body = document.createElement('div');
        body.className = 'activity-panel-body';

        var collapsed = false;
        header.addEventListener('click', function() {
          collapsed = !collapsed;
          body.style.display = collapsed ? 'none' : '';
          chevron.style.transform = collapsed ? 'rotate(-90deg)' : '';
        });

        section.appendChild(header);
        section.appendChild(body);
        return { section: section, body: body };
      }

      // Commits panel
      if (hasCommits) {
        var cp = makePanel('Commits', data.commits.length);
        data.commits.forEach(function(c) {
          var row = document.createElement('div');
          row.className = 'activity-commit-row';
          row.title = 'Click to copy SHA';
          row.style.cssText = 'display:flex;align-items:flex-start;gap:8px;padding:4px 0;cursor:pointer;';

          var shaEl = document.createElement('code');
          shaEl.className = 'activity-commit-sha';
          shaEl.textContent = (c.sha || '').slice(0, 7);
          shaEl.style.cssText = 'font-size:0.75rem;background:var(--bg-tertiary);padding:1px 5px;border-radius:3px;white-space:nowrap;flex-shrink:0;';
          row.appendChild(shaEl);

          var subjectEl = document.createElement('span');
          subjectEl.className = 'activity-commit-subject';
          subjectEl.textContent = c.subject || '';
          subjectEl.style.cssText = 'font-size:0.8rem;flex:1;word-break:break-word;';
          row.appendChild(subjectEl);

          var tsEl = document.createElement('span');
          tsEl.className = 'activity-commit-time';
          tsEl.textContent = relTime(c.timestamp);
          tsEl.style.cssText = 'font-size:0.7rem;color:var(--text-muted);white-space:nowrap;flex-shrink:0;';
          row.appendChild(tsEl);

          row.addEventListener('click', function() {
            navigator.clipboard && navigator.clipboard.writeText(c.sha || '').catch(function() {});
          });

          cp.body.appendChild(row);
        });
        container.appendChild(cp.section);
      }

      // Files panel
      if (hasFiles) {
        var fp = makePanel('Files', data.file_edits.length);
        data.file_edits.forEach(function(fe) {
          var row = document.createElement('div');
          row.className = 'activity-file-item';
          row.style.cssText = 'display:flex;align-items:center;gap:8px;padding:3px 0;';

          var pathEl = document.createElement('span');
          pathEl.className = 'activity-file-path';
          var parts = (fe.file_path || '').split('/');
          pathEl.textContent = parts.slice(-3).join('/') || fe.file_path;
          pathEl.title = fe.file_path;
          pathEl.style.cssText = 'font-size:0.8rem;flex:1;word-break:break-all;';
          row.appendChild(pathEl);

          var countBadge = document.createElement('span');
          countBadge.className = 'activity-file-count';
          countBadge.textContent = fe.edit_count + 'x';
          countBadge.style.cssText = 'font-size:0.7rem;background:var(--bg-tertiary);padding:1px 6px;border-radius:10px;white-space:nowrap;';
          row.appendChild(countBadge);

          var lastEl = document.createElement('span');
          lastEl.className = 'activity-file-last';
          lastEl.textContent = relTime(fe.last_edit);
          lastEl.style.cssText = 'font-size:0.7rem;color:var(--text-muted);white-space:nowrap;';
          row.appendChild(lastEl);

          row.addEventListener('click', function() {
            console.log('[htmlgraph] file trace:', fe.file_path);
          });

          fp.body.appendChild(row);
        });
        container.appendChild(fp.section);
      }

      // Activity (timeline) panel
      if (hasActivity) {
        var ap = makePanel('Activity', data.total_events);

        if (data.sessions && data.sessions.length > 0) {
          var sessDiv = document.createElement('div');
          sessDiv.className = 'activity-stat';
          sessDiv.style.marginBottom = '6px';
          sessDiv.innerHTML = 'across <strong>' + data.sessions.length + '</strong> session' + (data.sessions.length === 1 ? '' : 's');
          ap.body.appendChild(sessDiv);
        }

        if (data.events && data.events.length > 0) {
          var timeline = document.createElement('div');
          timeline.className = 'activity-timeline';

          data.events.forEach(function(ev) {
            var evRow = document.createElement('div');
            evRow.className = 'activity-event';
            evRow.title = ev.input_summary || ev.tool_name || '';

            var tsEl = document.createElement('span');
            tsEl.className = 'activity-event-time';
            var ts = ev.timestamp || '';
            var timePart = ts.indexOf('T') >= 0 ? ts.split('T')[1] : ts;
            tsEl.textContent = timePart ? timePart.slice(0, 8) : ts.slice(0, 10);
            evRow.appendChild(tsEl);

            var toolEl = document.createElement('span');
            var toolName = ev.tool_name || ev.event_type || '?';
            var toolKey = ['Edit','Read','Write','Bash','Glob','Grep'].indexOf(toolName) >= 0 ? toolName : 'default';
            toolEl.className = 'activity-event-tool activity-event-tool-' + toolKey;
            toolEl.textContent = toolName.slice(0, 12);
            evRow.appendChild(toolEl);

            var sumEl = document.createElement('span');
            sumEl.className = 'activity-event-summary';
            sumEl.textContent = (ev.input_summary || '').slice(0, 80);
            evRow.appendChild(sumEl);

            if (ev.session_id) {
              evRow.addEventListener('click', function() {
                openSessionDetail(ev.session_id);
              });
            }

            timeline.appendChild(evRow);
          });
          ap.body.appendChild(timeline);
        }
        container.appendChild(ap.section);
      }
    })
    .catch(function() { /* activity panels hidden on error */ });
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

/* ── Doorway mode: projects landing (root) vs single-project (/p/<id>/) ── */

// isDoorwayLanding returns true when the dashboard is loaded at the
// server root ("/") with no /p/<id>/ prefix. In this mode it shows the
// projects landing and clicking a card navigates to /p/<id>/ with a full
// page load — there is no SPA drill-in.
function isDoorwayLanding() {
  return window.location.pathname.indexOf('/p/') !== 0;
}

// detectMode calls /api/mode. When loaded at the doorway (root), it
// receives {"mode":"global"} and renders the projects landing. When
// loaded under /p/<id>/, it receives {"mode":"single"} (from the child)
// and proceeds with the regular single-project startup.
function detectMode() {
  // IMPORTANT: use buildProjectUrl so under /p/<id>/ this hits the child's
  // /api/mode (which carries projectName), not the parent's global mode.
  return fetch(buildProjectUrl('mode')).then(function(r) {
    if (!r.ok) return null;
    return r.json();
  }).then(function(data) {
    if (!data) return;
    window.htmlgraphMode = data.mode;
    if (data.mode === 'global' && isDoorwayLanding()) {
      return loadAndRenderProjectsLanding();
    }
    // Inside a project (single mode served by a child under /p/<id>/)
    // — label the header with the project name returned by /api/mode.
    if (data.mode === 'single' && data.projectName) {
      window.htmlgraphProjectName = data.projectName;
      var pe = document.getElementById('brand-project');
      if (pe) {
        pe.textContent = '/ ' + data.projectName;
        pe.style.display = '';
      }
      document.title = data.projectName + ' — HtmlGraph';
    }
  }).catch(function() {});
}

// loadAndRenderProjectsLanding fetches /api/projects (registry JSON
// only — no DB counts) and renders one card per project. Clicking a
// card navigates the browser to /p/<id>/ with a full page load.
function loadAndRenderProjectsLanding() {
  return fetch('/api/projects').then(function(r) {
    if (!r.ok) return [];
    return r.json();
  }).then(function(projects) {
    if (!Array.isArray(projects)) projects = [];
    window.htmlgraphProjects = projects;

    // Hide the per-project side nav — the landing is a level above.
    var nav = document.querySelector('.nav');
    if (nav) nav.style.display = 'none';

    // Activate the projects landing view.
    document.querySelectorAll('.view').forEach(function(v) { v.classList.remove('active'); });
    var landing = document.getElementById('v-projects');
    if (landing) landing.classList.add('active');

    renderProjectsLanding(projects);
  });
}

// renderProjectsLanding builds one card per registered project. Cards
// are simple metadata blocks (name, path, git remote, last seen) with a
// visible "Open →" affordance. Clicking or pressing Enter navigates to
// /p/<id>/ with a full page load. No SPA state management.
function renderProjectsLanding(projects) {
  var grid = document.getElementById('project-grid');
  if (!grid) return;
  grid.innerHTML = '';
  var empty = document.getElementById('projects-empty');
  var count = document.getElementById('projects-count');
  if (count) count.textContent = String(projects.length);
  if (empty) empty.style.display = projects.length === 0 ? 'block' : 'none';

  projects.forEach(function(p) {
    var card = document.createElement('div');
    card.className = 'project-card';
    card.setAttribute('role', 'button');
    card.setAttribute('tabindex', '0');
    var navigate = function() { window.location.href = '/p/' + p.id + '/'; };
    card.addEventListener('click', navigate);
    card.addEventListener('keydown', function(e) {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); navigate(); }
    });

    var header = document.createElement('div');
    header.className = 'project-card-header';
    var name = document.createElement('div');
    name.className = 'project-card-name';
    name.textContent = p.name || '(unnamed)';
    var dir = document.createElement('div');
    dir.className = 'project-card-dir';
    dir.textContent = p.dir || '';
    dir.title = p.dir || '';
    header.appendChild(name);
    header.appendChild(dir);

    var meta = document.createElement('div');
    meta.className = 'project-card-meta';
    var last = document.createElement('span');
    last.textContent = p.lastSeen ? relTime(p.lastSeen) : 'never seen';
    meta.appendChild(last);
    if (p.gitRemoteURL) {
      var remote = document.createElement('span');
      var short = p.gitRemoteURL.replace(/^https?:\/\/(github\.com|gitlab\.com|bitbucket\.org)\//, '').replace(/\.git$/, '');
      remote.textContent = short;
      remote.title = p.gitRemoteURL;
      meta.appendChild(remote);
    }

    var open = document.createElement('div');
    open.className = 'project-card-open';
    open.textContent = 'Open \u2192';

    card.appendChild(header);
    card.appendChild(meta);
    card.appendChild(open);
    grid.appendChild(card);
  });
}

// Startup: detect mode, then either render the landing (at root) or run
// the single-project startup (under /p/<id>/).
detectMode().then(function() {
  if (isDoorwayLanding()) {
    // Landing: hide the stats bar (no aggregate data in the doorway).
    var sb = document.getElementById('stats-bar');
    if (sb) sb.style.display = 'none';
    return;
  }
  // Inside a project via /p/<id>/ — inject a back link at the top of
  // the nav so the user can return to the projects doorway.
  var nav = document.querySelector('.nav');
  if (nav && !document.getElementById('doorway-back')) {
    var back = document.createElement('a');
    back.id = 'doorway-back';
    back.href = '/';
    back.className = 'nav-btn';
    back.innerHTML = '<span style="font-size:13px;margin-right:4px;">&larr;</span> All Projects';
    back.style.cssText = 'margin-bottom:12px;border-bottom:1px solid var(--border);padding-bottom:12px;display:flex;align-items:center;text-decoration:none;color:var(--text-dim);font-size:.82rem;';
    nav.insertBefore(back, nav.firstChild);
  }
  Promise.all([fetchStats(), fetchEvents()]);
});
setInterval(function() {
  if (!isDoorwayLanding()) fetchStats();
}, 30000);

// Auto-refresh the sessions list while it is the active view so an
// in-progress session's message count and LIVE badge stay current
// without manual reloads (bug-af5d048b). Cadence is 15s — half the
// stats interval, fast enough to feel live but well above the
// autoIngest sweep window so we don't waste cycles between sweeps.
// Gated on currentView so we only fetch when the user is looking,
// avoiding background work for the activity/work/graph/plans views.
var SESSIONS_REFRESH_MS = 15000;
setInterval(function() {
  if (currentView === 'sessions' && !isDoorwayLanding()) {
    fetchSessions();
  }
}, SESSIONS_REFRESH_MS);

/* ── Plan detail panel ────────────────────────────────────── */

// navigateToWorkDetail switches the dashboard to the Work view and opens
// the given work item's detail panel. Called from cross-view badge
// clicks (transcript stats, session list) so the behaviour matches
// clicking a work-item node in the graph view. Without this helper,
// callers would have to duplicate the nav-btn / .view class toggling
// and still end up with the detail panel rendered inside a hidden tab
// (roborev finding on job 886).
function navigateToWorkDetail(id) {
  if (!id) return;
  currentView = 'work';
  document.querySelectorAll('.nav-btn').forEach(function(b) {
    b.classList.toggle('active', b.dataset.view === 'work');
  });
  document.querySelectorAll('.view').forEach(function(v) {
    v.classList.toggle('active', v.id === 'v-work');
  });
  if (typeof openWorkDetail === 'function') {
    openWorkDetail(id);
  }
}

// navigateToPlan switches to the plans view and opens the given plan.
// Called from cross-view badge clicks (e.g. session list, transcript view).
function navigateToPlan(planId, title) {
  // Resolve title from already-loaded plans list when not provided.
  if (!title && plans.length > 0) {
    var found = plans.find(function(p) { return p.id === planId; });
    title = found ? found.title : planId;
  }

  // Switch nav to plans view (the click handler calls fetchPlans if needed).
  var planBtn = document.querySelector('.nav-btn[data-view="plans"]');
  if (planBtn && currentView !== 'plans') planBtn.click();

  // Open the plan detail immediately — openPlanDetail fetches its own content.
  openPlanDetail(planId, title || planId);
}

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

  fetch(buildProjectUrl('plans/' + planId + '/render'))
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

/* ── Graph View ────────────────────────────────────────────── */

var graphSimulation = null;
// Observer that watches the root <html> element for data-theme
// attribute changes so we can repaint existing graph nodes and the
// legend without tearing down the D3 simulation. Disconnected on
// every renderGraph call and re-created against the new selection.
var graphThemeObserver = null;

// GRAPH_LAYOUT centralizes every tunable constant in the graph view so
// values can be adjusted in one place instead of scattered across
// renderGraph. FNV_* drives the per-node deterministic position seeding;
// TYPE_BAND_Y is the vertical fraction of the viewport each node type
// anchors to (tracks near the top, sessions near the bottom); SIM_*
// governs the force-layout cooldown.
var GRAPH_LAYOUT = {
  // FNV-1a hash constants for deterministic (x, y) seeding.
  FNV_OFFSET_BASIS: 2166136261,
  FNV_PRIME: 16777619,
  // Hash bit ranges used to spread nodes within a band.
  HASH_X_MODULUS: 1000,          // low bits drive x position
  HASH_Y_MODULUS: 200,           // next 8 bits drive y jitter
  HASH_Y_SHIFT: 10,              // bit shift before the y modulus
  BAND_Y_JITTER_FRACTION: 0.15,  // fraction of viewport height a node can drift within its band
  // Vertical anchors per node type, expressed as a fraction of viewport
  // height. Chosen to match the eventual force-layout clusters so the
  // relaxation pass has almost no work to do.
  TYPE_BAND_Y: {
    track:   0.20,
    plan:    0.30,
    agent:   0.35,
    feature: 0.50,
    bug:     0.55,
    spike:   0.60,
    commit:  0.70,
    session: 0.80,
    file:    0.90
  },
  // Force-simulation cooldown. Lower starting alpha + faster decay
  // because nodes are pre-seeded near their final positions.
  SIM_INITIAL_ALPHA: 0.3,
  SIM_ALPHA_DECAY:   0.05,
  // Inter-node repulsion. More negative = more spread. Tuned by eye
  // on a 500+ node graph; -220 gives the dense center room to breathe
  // without blowing the whole graph outside the viewport.
  CHARGE_STRENGTH: -220,
  // Per-type fill, keyed to design-system tokens so the graph inherits
  // the active theme automatically. Resolved live via
  // getComputedStyle(document.documentElement) at every getGraphPalette()
  // call — CSS `var(...)` cannot be assigned directly to a d3 `fill`
  // attribute, and the computed value flips when the user toggles theme.
  // Track is the brand accent; feature/plan are the grayscale tier
  // (plan is differentiated with a dashed stroke in the node render);
  // bug/spike/session reuse semantic status/priority tokens.
  TYPE_TOKEN: {
    track:   '--accent',
    feature: '--text-secondary',
    plan:    '--text-muted',
    bug:     '--status-blocked',
    spike:   '--priority-high',
    agent:   '--graph-agent',   // purple — was amber, collided with spike
    commit:  '--status-done',
    session: '--status-ip',
    file:    '--graph-file'     // slate — was muted grey, collided with plan
  },
  // Fill opacity for non-session nodes. Sessions stay at their
  // existing 0.6 — they're secondary. 0.88 takes a little more
  // edge off the primary nodes without making them look washed out.
  NODE_FILL_OPACITY: 0.88
};

// getGraphPalette resolves GRAPH_LAYOUT.TYPE_TOKEN into a flat map of
// concrete color strings using the live computed values of the root
// element. Called on every render and on every data-theme mutation so
// the result always reflects the active theme.
function getGraphPalette() {
  var cs = getComputedStyle(document.documentElement);
  var out = {};
  var tokens = GRAPH_LAYOUT.TYPE_TOKEN;
  for (var key in tokens) {
    if (Object.prototype.hasOwnProperty.call(tokens, key)) {
      out[key] = cs.getPropertyValue(tokens[key]).trim() || '#888';
    }
  }
  return out;
}

// colorToRGB parses any CSS color string getComputedStyle might return
// (rgb, rgba, hex) into a numeric [r,g,b] triple. Named colors and hsl
// are not expected on our tokens but fall through to a neutral gray so
// YIQ still produces a reasonable pick.
function colorToRGB(c) {
  if (!c) return [128, 128, 128];
  if (c[0] === '#') {
    if (c.length === 4) c = '#' + c[1]+c[1] + c[2]+c[2] + c[3]+c[3];
    return [parseInt(c.slice(1,3), 16), parseInt(c.slice(3,5), 16), parseInt(c.slice(5,7), 16)];
  }
  var m = c.match(/\d+(\.\d+)?/g);
  if (!m || m.length < 3) return [128, 128, 128];
  return [parseFloat(m[0]) | 0, parseFloat(m[1]) | 0, parseFloat(m[2]) | 0];
}

// pickLabelColor picks a near-black or near-white ink for a given node
// fill using YIQ luminance, so labels stay legible on every palette
// entry regardless of active theme. The paint-order stroke layered on
// top of the label adds a second line of defense for edge-case fills.
function pickLabelColor(fill) {
  var rgb = colorToRGB(fill);
  var yiq = (rgb[0] * 299 + rgb[1] * 587 + rgb[2] * 114) / 1000;
  return yiq >= 140 ? '#0a0a0a' : '#f0f0f0';
}

// paintGraphLegend applies the current-theme colors to every legend
// entry carrying a data-graph-type attribute. Called on initial render
// and again whenever the theme toggles.
function paintGraphLegend() {
  var palette = getGraphPalette();
  var spans = document.querySelectorAll('[data-graph-type]');
  for (var i = 0; i < spans.length; i++) {
    var t = spans[i].getAttribute('data-graph-type');
    if (palette[t]) spans[i].style.color = palette[t];
  }
}

// Active type filter state — null means "show all".
var graphActiveTypes = null;

// Race-proofing: track the current fetch so stale responses from a rapid
// sequence of toggles don't overwrite newer graph state. Each call bumps the
// token and aborts the previous request; the .then() checks the token before
// rendering.
var graphFetchToken = 0;
var graphFetchController = null;

function fetchGraph(types) {
  var url = buildProjectUrl('graph');
  if (types && types.length > 0) url += (url.indexOf('?') >= 0 ? '&' : '?') + 'types=' + types.join(',');

  // Cancel any in-flight request before starting a new one.
  if (graphFetchController) {
    try { graphFetchController.abort(); } catch (e) {}
  }
  graphFetchController = typeof AbortController === 'function' ? new AbortController() : null;
  var myToken = ++graphFetchToken;
  var signal = graphFetchController ? graphFetchController.signal : undefined;

  fetch(url, { signal: signal })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      // Drop stale responses — only the latest token is allowed to render.
      if (myToken !== graphFetchToken) return;
      document.getElementById('graph-count').textContent = data.nodes ? data.nodes.length : 0;
      var empty = document.getElementById('graph-empty');
      if (!data.nodes || data.nodes.length === 0) {
        empty.style.display = '';
        return;
      }
      empty.style.display = 'none';
      renderGraph(data);
    })
    .catch(function(err) {
      if (err && err.name === 'AbortError') return;
      if (myToken !== graphFetchToken) return;
      document.getElementById('graph-empty').style.display = '';
    });
}

function renderGraph(data) {
  var container = document.getElementById('graph-container');
  // Remove any previous SVG but keep the legend and empty overlay.
  var oldSvg = container.querySelector('svg');
  if (oldSvg) oldSvg.remove();
  var oldTip = container.querySelector('.graph-tooltip');
  if (oldTip) oldTip.remove();

  // Build or update the filter toolbar.
  var oldToolbar = container.querySelector('.graph-filter-toolbar');
  if (oldToolbar) oldToolbar.remove();
  var allTypes = ['track', 'agent', 'feature', 'bug', 'spike', 'plan', 'session', 'commit', 'file'];
  var typeCounts = {};
  (data.nodes || []).forEach(function(n) { typeCounts[n.type] = (typeCounts[n.type] || 0) + 1; });
  var toolbar = document.createElement('div');
  toolbar.className = 'graph-filter-toolbar';
  allTypes.forEach(function(type) {
    if (!typeCounts[type] && !(graphActiveTypes && graphActiveTypes.indexOf(type) < 0)) return;
    var btn = document.createElement('button');
    var active = !graphActiveTypes || graphActiveTypes.indexOf(type) >= 0;
    btn.className = 'graph-filter-btn' + (active ? ' active' : '');
    btn.dataset.type = type;
    var label = type.charAt(0).toUpperCase() + type.slice(1);
    var count = typeCounts[type] || 0;
    var capText = data.caps && data.caps[type] && data.caps[type].total > data.caps[type].shown
      ? ' of ' + data.caps[type].total : '';
    btn.innerHTML = '<span class="filter-dot" data-graph-type="' + type + '">\u25CF</span> ' + label +
      ' <span style="opacity:0.6">' + count + capText + '</span>';
    btn.onclick = function() {
      if (!graphActiveTypes) {
        // First click: drop the clicked type, keep all others.
        graphActiveTypes = allTypes.filter(function(t) { return t !== type; });
      } else {
        var idx = graphActiveTypes.indexOf(type);
        if (idx >= 0) {
          // Refuse to deselect the last remaining active type — a request
          // with an empty list would silently reload the full graph and
          // desync the toolbar from the canvas. The user can use Reset to
          // return to the default view.
          if (graphActiveTypes.length <= 1) return;
          graphActiveTypes.splice(idx, 1);
        } else {
          graphActiveTypes.push(type);
        }
        if (graphActiveTypes.length === allTypes.length) graphActiveTypes = null;
      }
      fetchGraph(graphActiveTypes);
    };
    toolbar.appendChild(btn);
  });
  var resetBtn = document.createElement('button');
  resetBtn.className = 'graph-filter-btn';
  resetBtn.textContent = 'Reset';
  resetBtn.onclick = function() { graphActiveTypes = null; fetchGraph(); };
  toolbar.appendChild(resetBtn);
  container.insertBefore(toolbar, container.firstChild);

  if (graphSimulation) {
    graphSimulation.stop();
    graphSimulation = null;
  }
  if (graphThemeObserver) {
    graphThemeObserver.disconnect();
    graphThemeObserver = null;
  }

  var width = container.clientWidth || 800;
  var height = container.clientHeight || 600;

  var svg = d3.select('#graph-container').append('svg')
    .attr('width', width)
    .attr('height', height)
    .style('display', 'block');

  // Zoom / pan layer.
  var g = svg.append('g');
  svg.call(d3.zoom()
    .scaleExtent([0.1, 4])
    .on('zoom', function(e) { g.attr('transform', e.transform); })
  );

  // Colour by node type — theme-aware via getGraphPalette(). The
  // variable is reassigned inside the MutationObserver below so a
  // theme toggle repaints existing nodes without rebuilding the
  // simulation.
  var typeColor = getGraphPalette();
  paintGraphLegend();

  // Node size combines edges (structural weight) and activity (usage weight).
  // Log scale spreads small nodes more and compresses large ones so hubs
  // don't all blob together at max size.
  function nodeRadius(d) {
    var edges = d.edges || 0;
    var activity = d.activity || 0;
    // Weighted combination — edges matter more than raw activity.
    var weight = edges * 2 + Math.sqrt(activity);
    // Log scale with minimum floor and max cap.
    var r = 4 + Math.log(1 + weight) * 4;
    return Math.max(4, Math.min(28, r));
  }

  // visualRadius is the ACTUAL rendered radius of the circle, after any
  // type-specific scaling. This is the value to use for hit testing,
  // text wrapping, and label fit decisions so a single source of truth
  // governs "how big is this node on screen." Previously the code used
  // nodeRadius() directly for labels while the circle renderer scaled
  // session nodes to 60% — the labels thought they had more room than
  // the circle actually provided and spilled onto the background.
  function visualRadius(d) {
    if (d.type === 'session') return Math.max(3, nodeRadius(d) * 0.6);
    if (d.type === 'commit' || d.type === 'file') return Math.max(3, nodeRadius(d) * 0.5);
    if (d.type === 'agent') return Math.max(4, nodeRadius(d) * 0.8);
    return nodeRadius(d);
  }

  // Make a shallow copy so D3 can mutate positions without polluting our cache.
  var nodes = data.nodes.map(function(n) { return Object.assign({}, n); });
  var edges = data.edges.map(function(e) { return Object.assign({}, e); });

  // Seed deterministic starting positions so nodes appear in roughly stable
  // locations instead of random chaos. Without this, D3 assigns every node a
  // random (x, y) and the simulation bounces everything into place — visible
  // as a jarring "explosion and settle" on every load. With seeded positions,
  // the simulation relaxes from a near-final layout, producing a small shiver.
  // All tunables live in GRAPH_LAYOUT at the top of the Graph View section.
  function hashNodeId(s) {
    var h = GRAPH_LAYOUT.FNV_OFFSET_BASIS;
    for (var i = 0; i < s.length; i++) {
      h ^= s.charCodeAt(i);
      // Math.imul performs true 32-bit integer multiplication; plain
      // `h * FNV_PRIME` would overflow a 64-bit float past 2^53 and
      // silently corrupt the hash distribution, causing avoidable
      // clustering in the seeded layout (roborev finding on f0b9d8aa).
      h = Math.imul(h, GRAPH_LAYOUT.FNV_PRIME) >>> 0;
    }
    return h;
  }
  var DEFAULT_BAND_Y = 0.5;
  nodes.forEach(function(n) {
    var h = hashNodeId(n.id);
    var bandFraction = GRAPH_LAYOUT.TYPE_BAND_Y[n.type];
    if (bandFraction === undefined) bandFraction = DEFAULT_BAND_Y;
    var bandY = bandFraction * height;
    var jitterRange = height * GRAPH_LAYOUT.BAND_Y_JITTER_FRACTION;
    n.x = ((h % GRAPH_LAYOUT.HASH_X_MODULUS) / GRAPH_LAYOUT.HASH_X_MODULUS) * width;
    n.y = bandY + ((((h >>> GRAPH_LAYOUT.HASH_Y_SHIFT) % GRAPH_LAYOUT.HASH_Y_MODULUS) / GRAPH_LAYOUT.HASH_Y_MODULUS) - 0.5) * jitterRange;
  });

  // Balanced forces: clusters visible but not overlapping.
  // Link strength varies by type: structural edges pull tighter than activity.
  // SIM_INITIAL_ALPHA and SIM_ALPHA_DECAY are lowered from the D3 defaults
  // (1.0 and 0.0228 respectively) because nodes are pre-seeded near their
  // final positions — the simulation only needs a short relaxation pass.
  graphSimulation = d3.forceSimulation(nodes)
    .alpha(GRAPH_LAYOUT.SIM_INITIAL_ALPHA)
    .alphaDecay(GRAPH_LAYOUT.SIM_ALPHA_DECAY)
    .force('link', d3.forceLink(edges).id(function(d) { return d.id; })
      .distance(function(d) {
        return d.type === 'worked_on' ? 70 : 45;
      })
      .strength(function(d) {
        // Structural edges dominate the layout; activity edges are loose.
        if (d.type === 'worked_on') return 0.2;
        if (d.type === 'part_of') return 0.9;
        return 0.6;
      }))
    .force('charge', d3.forceManyBody().strength(GRAPH_LAYOUT.CHARGE_STRENGTH).distanceMax(400))
    .force('center', d3.forceCenter(width / 2, height / 2))
    .force('x', d3.forceX(width / 2).strength(0.015))
    .force('y', d3.forceY(height / 2).strength(0.015))
    .force('collision', d3.forceCollide().radius(function(d) { return visualRadius(d) + 3; }));

  // Edge color by relationship type for visual variety.
  var edgeColor = {
    part_of:       '#4b5563',
    blocked_by:    '#dc2626',
    caused_by:     '#f59e0b',
    implements:    '#3b82f6',
    contains:      '#22c55e',
    co_session:    '#8b5cf6',
    worked_on:     '#06b6d4',
    committed_for: '#10b981',
    produced_by:   '#0ea5e9',
    produced_in:   '#a78bfa',
    touched_by:    '#6b7280',
    spawned:       '#f97316',
    ran_as:        '#f59e0b'
  };

  // Edge lines — structural edges bolder, activity edges subtle.
  var link = g.append('g').selectAll('line')
    .data(edges).enter().append('line')
    .attr('stroke', function(d) { return edgeColor[d.type] || '#6b7280'; })
    .attr('stroke-opacity', function(d) {
      if (d.type === 'worked_on' || d.type === 'committed_for' || d.type === 'touched_by') return 0.25;
      return 0.6;
    })
    .attr('stroke-width', function(d) {
      return d.type === 'worked_on' ? 0.7 : 1.2;
    })
    .attr('stroke-dasharray', function(d) {
      return d.type === 'spawned' ? '6,3' : null;
    });

  // Node circles. Radius flows through visualRadius so the on-screen
  // size matches the value used by label wrapping and the collision
  // force below — one source of truth for "how big is this node."
  var node = g.append('g').selectAll('circle')
    .data(nodes).enter().append('circle')
    .attr('r', visualRadius)
    .attr('fill', function(d) { return typeColor[d.type] || '#888'; })
    .attr('fill-opacity', function(d) {
      if (d.type === 'session') return 0.6;
      if (d.type === 'commit' || d.type === 'file') return 0.5;
      return GRAPH_LAYOUT.NODE_FILL_OPACITY;
    })
    .attr('stroke', 'var(--bg-primary)')
    .attr('stroke-width', 1.5)
    // Plans share the grayscale tier with features. The dashed outline
    // signals "blueprint, not built yet" so the two tiers stay distinct
    // even when their fill tokens resolve to similar neutrals.
    .attr('stroke-dasharray', function(d) { return d.type === 'plan' ? '4,2' : null; })
    .style('cursor', 'pointer')
    .call(d3.drag()
      .on('start', function(e, d) {
        if (!e.active) graphSimulation.alphaTarget(0.3).restart();
        d.fx = d.x; d.fy = d.y;
      })
      .on('drag', function(e, d) { d.fx = e.x; d.fy = e.y; })
      .on('end', function(e, d) {
        if (!e.active) graphSimulation.alphaTarget(0);
        d.fx = null; d.fy = null;
      })
    );

  // Icon overlay — draw an SVG icon inside nodes that are large enough to
  // read it. Small high-cardinality nodes (sessions, commits, files under
  // ~10px radius) stay as colored circles only; icons there would be pixel
  // mush. Icons inherit the node fill via currentColor so they follow the
  // theme. Pointer-events disabled so drag/click still target the circle.
  var ICON_MIN_RADIUS = 10;
  var iconTypes = { track:1, plan:1, feature:1, bug:1, spike:1, agent:1, commit:1, session:1, file:1 };
  var icons = g.append('g')
    .attr('pointer-events', 'none')
    .selectAll('use')
    .data(nodes.filter(function(d) { return iconTypes[d.type] && visualRadius(d) >= ICON_MIN_RADIUS; }))
    .enter().append('use')
    .attr('href', function(d) { return '#icon-' + d.type; })
    .attr('color', 'var(--bg-primary)')   // icon stroke/fill inherits via currentColor
    .attr('opacity', 0.95);

  // Repaint nodes, labels, and legend on theme toggle without tearing
  // down the simulation. The closure captures `node` / the label
  // selections and reassigns `typeColor` so subsequent fill reads stay
  // in sync (used by drag/hover handlers that reuse typeColor).
  graphThemeObserver = new MutationObserver(function() {
    typeColor = getGraphPalette();
    node.attr('fill', function(d) { return typeColor[d.type] || '#888'; });
    if (typeof trackLabels !== 'undefined') {
      trackLabels.attr('fill', function(d) { return pickLabelColor(typeColor[d.type] || '#888'); });
    }
    if (typeof hubLabels !== 'undefined') {
      hubLabels.attr('fill', function(d) { return pickLabelColor(typeColor[d.type] || '#888'); });
    }
    paintGraphLegend();
  });
  graphThemeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-theme']
  });

  // Tooltip.
  var tooltip = d3.select('#graph-container').append('div')
    .attr('class', 'graph-tooltip')
    .style('position', 'absolute')
    .style('background', 'rgba(15, 23, 42, 0.9)')
    .style('backdrop-filter', 'blur(4px)')
    .style('-webkit-backdrop-filter', 'blur(4px)')
    .style('border', '1px solid var(--border)')
    .style('padding', '8px 12px')
    .style('border-radius', '6px')
    .style('font-size', '12px')
    .style('pointer-events', 'none')
    .style('opacity', 0)
    .style('color', 'var(--text-primary)')
    .style('max-width', '240px')
    .style('z-index', 20)
    .style('box-shadow', '0 4px 12px rgba(0,0,0,0.4)');

  node.on('mouseover', function(e, d) {
    var rect = container.getBoundingClientRect();
    // Tooltip content is built with DOM text nodes instead of .html(...)
    // because d.title can originate from a user prompt (session nodes
    // now use sessions.title or the first user message as their label)
    // and passing that through innerHTML would let a crafted prompt
    // inject <script> into the dashboard (roborev finding on job 886).
    var tipEl = tooltip.node();
    tipEl.textContent = '';
    var titleEl = document.createElement('strong');
    titleEl.textContent = d.title || '';
    tipEl.appendChild(titleEl);
    tipEl.appendChild(document.createElement('br'));
    var meta = (d.type || '') + ' · ' + (d.status || '') +
               ' · ' + (d.edges || 0) + ' edge' + (d.edges !== 1 ? 's' : '');
    tipEl.appendChild(document.createTextNode(meta));
    tooltip.style('opacity', 1)
      .style('left', (e.clientX - rect.left + 12) + 'px')
      .style('top', (e.clientY - rect.top - 10) + 'px');
    // Highlight connected nodes.
    var connected = new Set();
    edges.forEach(function(edge) {
      var src = typeof edge.source === 'object' ? edge.source.id : edge.source;
      var tgt = typeof edge.target === 'object' ? edge.target.id : edge.target;
      if (src === d.id) connected.add(tgt);
      if (tgt === d.id) connected.add(src);
    });
    node.attr('opacity', function(n) {
      return n.id === d.id || connected.has(n.id) ? 1 : 0.25;
    });
    link.attr('stroke-opacity', function(edge) {
      var src = typeof edge.source === 'object' ? edge.source.id : edge.source;
      var tgt = typeof edge.target === 'object' ? edge.target.id : edge.target;
      return (src === d.id || tgt === d.id) ? 0.9 : 0.05;
    });
  }).on('mousemove', function(e) {
    var rect = container.getBoundingClientRect();
    tooltip.style('left', (e.clientX - rect.left + 12) + 'px').style('top', (e.clientY - rect.top - 10) + 'px');
  }).on('mouseout', function() {
    tooltip.style('opacity', 0);
    node.attr('opacity', 1);
    link.attr('stroke-opacity', 0.5);
  }).on('click', function(e, d) {
    // All node types open the provenance panel for causal-chain drill-down.
    openProvenancePanel(d.id);
  });

  // Wrap text inside a circle using real SVG measurement via getComputedTextLength.
  // Uses binary iteration: tries to fit text, shrinks font if needed, hides if too small.
  function wrapTextInCircle(textEl, title, radius) {
    textEl.text(null);
    var words = title.split(/\s+/).filter(function(w) { return w.length > 0; });
    if (words.length === 0) return;

    // Start with a font size proportional to radius, then shrink if needed.
    var minFont = 6;
    var maxFont = Math.max(minFont, Math.min(12, radius * 0.32));
    var fontSize = maxFont;

    // Try shrinking font until the text fits, or give up and truncate.
    for (var attempt = 0; attempt < 4; attempt++) {
      textEl.text(null).attr('font-size', fontSize + 'px');
      var lineHeight = fontSize * 1.15;
      // Reserve inner area — circle chord at top/bottom is narrower.
      var innerRadius = radius * 0.92;
      var maxLines = Math.max(1, Math.floor((innerRadius * 2) / lineHeight));

      // Greedy word wrap, measuring actual rendered width per line.
      var lines = [];
      var i = 0;
      var fit = true;
      while (i < words.length && lines.length < maxLines) {
        // Compute the chord width at this line's y-offset.
        var lineIdx = lines.length;
        var yOffset = (lineIdx - (maxLines - 1) / 2) * lineHeight;
        var chord = 2 * Math.sqrt(Math.max(0, innerRadius * innerRadius - yOffset * yOffset));
        if (chord <= 0) break;

        // Create a temp tspan to measure word fit.
        var tspan = textEl.append('tspan').attr('x', 0).attr('dy', 0);
        var line = words[i];
        tspan.text(line);
        // If even a single word doesn't fit, we need a smaller font.
        if (tspan.node().getComputedTextLength() > chord) {
          fit = false;
          tspan.remove();
          break;
        }
        i++;
        // Add words while they fit.
        while (i < words.length) {
          tspan.text(line + ' ' + words[i]);
          if (tspan.node().getComputedTextLength() > chord) {
            tspan.text(line);
            break;
          }
          line = line + ' ' + words[i];
          i++;
        }
        lines.push(line);
      }

      if (!fit) {
        // Single word too wide for any line — shrink font and retry.
        fontSize = Math.max(minFont, fontSize - 1);
        if (fontSize === minFont) {
          // Last resort: truncate the long word with ellipsis.
          textEl.text(null);
          textEl.append('tspan').attr('x', 0).attr('dy', 0).text(words[0].substring(0, 4) + '\u2026');
          return;
        }
        continue;
      }

      // Successfully laid out. Rebuild tspans with correct dy offsets.
      textEl.text(null);
      var startY = -((lines.length - 1) * lineHeight) / 2;
      var anyTruncated = i < words.length;
      if (anyTruncated && lines.length > 0) {
        // Append ellipsis to last line if we couldn't fit all words.
        var last = lines[lines.length - 1];
        // Try to append an ellipsis that still fits.
        var testSpan = textEl.append('tspan').attr('x', 0).attr('dy', 0);
        var yOffset2 = ((lines.length - 1) - (maxLines - 1) / 2) * lineHeight;
        var chord2 = 2 * Math.sqrt(Math.max(0, innerRadius * innerRadius - yOffset2 * yOffset2));
        testSpan.text(last + '\u2026');
        if (testSpan.node().getComputedTextLength() > chord2 && last.length > 1) {
          lines[lines.length - 1] = last.substring(0, last.length - 1) + '\u2026';
        } else {
          lines[lines.length - 1] = last + '\u2026';
        }
        textEl.text(null);
      }

      for (var k = 0; k < lines.length; k++) {
        textEl.append('tspan')
          .attr('x', 0)
          .attr('dy', k === 0 ? startY : lineHeight)
          .text(lines[k]);
      }
      return;
    }
  }

  // Labels inside track nodes using SVG text + tspan (no foreignObject).
  // Fill is contrast-aware via pickLabelColor so labels stay legible
  // regardless of which palette token the node resolved to. No
  // paint-order stroke — labels wrap inside the node radius, never
  // cross onto the background, and a dark halo would visibly thicken
  // and blur the small font sizes that fit inside sub-20px nodes.
  var trackLabelNodes = nodes.filter(function(d) { return d.type === 'track'; });
  var trackLabelGroup = g.append('g');
  var trackLabels = trackLabelGroup.selectAll('text.track-label')
    .data(trackLabelNodes)
    .enter().append('text')
    .attr('class', 'track-label')
    .attr('text-anchor', 'middle')
    .attr('dominant-baseline', 'central')
    .attr('fill', function(d) { return pickLabelColor(typeColor[d.type] || '#888'); })
    .attr('font-weight', 'bold')
    .attr('pointer-events', 'none');

  trackLabels.each(function(d) {
    wrapTextInCircle(d3.select(this), d.title, visualRadius(d));
  });

  // Hub node labels — fit inside the circle when node is large enough.
  // Uses visualRadius (not nodeRadius) so the "is this big enough to
  // label?" test matches the ACTUAL on-screen size, which for session
  // nodes is 60% of nodeRadius. Without this, session labels thought
  // they had 67% more space than the circle actually provided and
  // spilled onto the background.
  var hubNodes = nodes.filter(function(d) {
    return d.type !== 'track' && (d.edges || 0) >= 3 && visualRadius(d) >= 10;
  });

  var hubLabels = g.append('g').selectAll('text.hub-label')
    .data(hubNodes)
    .enter().append('text')
    .attr('class', 'hub-label')
    .attr('text-anchor', 'middle')
    .attr('dominant-baseline', 'central')
    .attr('fill', function(d) { return pickLabelColor(typeColor[d.type] || '#888'); })
    .attr('font-weight', '600')
    .attr('pointer-events', 'none');

  hubLabels.each(function(d) {
    wrapTextInCircle(d3.select(this), d.title, visualRadius(d));
  });

  graphSimulation.on('tick', function() {
    link
      .attr('x1', function(d) { return d.source.x; })
      .attr('y1', function(d) { return d.source.y; })
      .attr('x2', function(d) { return d.target.x; })
      .attr('y2', function(d) { return d.target.y; });
    node
      .attr('cx', function(d) { return d.x; })
      .attr('cy', function(d) { return d.y; });
    // Icons sit at 60% of the node's visual radius so they don't touch the
    // ring. Anchored via x/y = center - size/2 since <use> honors the symbol
    // viewBox as its own coordinate space.
    icons
      .attr('width', function(d) { return visualRadius(d) * 1.2; })
      .attr('height', function(d) { return visualRadius(d) * 1.2; })
      .attr('x', function(d) { return d.x - visualRadius(d) * 0.6; })
      .attr('y', function(d) { return d.y - visualRadius(d) * 0.6; });
    trackLabels
      .attr('transform', function(d) { return 'translate(' + d.x + ',' + d.y + ')'; });
    hubLabels
      .attr('transform', function(d) { return 'translate(' + d.x + ',' + d.y + ')'; });
  });
}

// openProvenancePanel fetches and displays the causal chain for a graph node
// in the fixed right-side drawer. Each upstream/downstream item is clickable
// to drill into that node's own provenance.
//
// Race-proofed: rapid clicks on a chain of nodes would otherwise let a slow
// earlier response overwrite the newer drawer. The token/abort pair mirrors
// fetchGraph above.
var provenanceFetchToken = 0;
var provenanceFetchController = null;

function openProvenancePanel(nodeId) {
  var panel = document.getElementById('provenance-panel');
  var titleEl = document.getElementById('provenance-title');
  var badge = document.getElementById('provenance-type-badge');
  var upstreamEl = document.getElementById('provenance-upstream');
  var downstreamEl = document.getElementById('provenance-downstream');

  if (provenanceFetchController) {
    try { provenanceFetchController.abort(); } catch (e) {}
  }
  provenanceFetchController = typeof AbortController === 'function' ? new AbortController() : null;
  var myToken = ++provenanceFetchToken;
  var signal = provenanceFetchController ? provenanceFetchController.signal : undefined;

  fetch(buildProjectUrl('provenance/' + encodeURIComponent(nodeId)), { signal: signal })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (myToken !== provenanceFetchToken) return;
      titleEl.textContent = data.node.title || data.node.id;
      badge.textContent = data.node.type;
      badge.className = 'type-badge type-' + data.node.type;

      upstreamEl.innerHTML = '';
      (data.upstream || []).forEach(function(link) {
        var li = document.createElement('li');
        var rel = document.createElement('span');
        rel.className = 'provenance-rel';
        rel.textContent = link.relationship;
        var label = document.createElement('span');
        label.textContent = link.title || link.id;
        li.appendChild(rel);
        li.appendChild(label);
        li.onclick = function() { openProvenancePanel(link.id); };
        upstreamEl.appendChild(li);
      });

      downstreamEl.innerHTML = '';
      (data.downstream || []).forEach(function(link) {
        var li = document.createElement('li');
        var rel = document.createElement('span');
        rel.className = 'provenance-rel';
        rel.textContent = link.relationship;
        var label = document.createElement('span');
        label.textContent = link.title || link.id;
        li.appendChild(rel);
        li.appendChild(label);
        li.onclick = function() { openProvenancePanel(link.id); };
        downstreamEl.appendChild(li);
      });

      panel.classList.remove('hidden');
    })
    .catch(function(err) {
      if (err && err.name === 'AbortError') return;
      if (myToken !== provenanceFetchToken) return;
      console.error('provenance fetch failed', err);
    });
}

(function() {
  var closeBtn = document.getElementById('provenance-close');
  if (closeBtn) {
    closeBtn.addEventListener('click', function() {
      document.getElementById('provenance-panel').classList.add('hidden');
    });
  }
})();

// openSessionDetail switches to the sessions view and highlights a specific session.
function openSessionDetail(sessionId) {
  currentView = 'sessions';
  document.querySelectorAll('.nav-btn').forEach(function(b) {
    b.classList.toggle('active', b.dataset.view === 'sessions');
  });
  document.querySelectorAll('.view').forEach(function(v) {
    v.classList.toggle('active', v.id === 'v-sessions');
  });
  // Open the transcript directly — don't just highlight the list row.
  if (sessions.length === 0) {
    fetchSessions().then(function() { openTranscript(sessionId); });
  } else {
    openTranscript(sessionId);
  }
}

// highlightSession scrolls to and briefly highlights a session row by ID.
function highlightSession(sessionId) {
  var el = document.querySelector('[data-session-id="' + sessionId + '"]');
  if (!el) return;
  el.scrollIntoView({ behavior: 'smooth', block: 'center' });
  el.style.outline = '2px solid var(--accent)';
  setTimeout(function() { el.style.outline = ''; }, 2000);
}
