/* ── Application state & data fetching ─────────────────────── */

var events = [];
var sessions = [];
var features = [];
var stats = {};
var currentView = 'activity';
var seenEventIds = new Set();

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
  setVal('sv-events', stats.total_events);
  setVal('sv-sessions', stats.active_sessions);
  setVal('sv-feat-ip', stats.features_in_progress);
  setVal('sv-feat-done', stats.features_done);
}

function fetchEvents() {
  return fetch('/api/events/recent?limit=100').then(function(r) {
    if (!r.ok) return;
    return r.json().then(function(data) {
      events = data;
      events.forEach(function(e) { seenEventIds.add(e.event_id); });
      renderAgentCount();
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
      renderFeatures();
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

/* ── Rendering: Work ───────────────────────────────────────── */
function renderFeatures() {
  var body = document.getElementById('work-body');
  var empty = document.getElementById('work-empty');
  document.getElementById('work-count').textContent = features.length;
  body.textContent = '';
  if (features.length === 0) { empty.style.display = ''; return; }
  empty.style.display = 'none';

  var frag = document.createDocumentFragment();
  features.forEach(function(f) {
    var tr = document.createElement('tr');
    tr.appendChild(td(f.id, { className: 'mono', style: 'color:var(--accent)' }));
    tr.appendChild(tdWithChild(createStatusBadge(f.status)));
    tr.appendChild(tdWithChild(createPriorityBadge(f.priority)));
    tr.appendChild(td(f.title || f.id, { className: 'ellipsis', title: true }));

    var progCell = document.createElement('td');
    if (f.steps_total > 0) {
      var pct = Math.round((f.steps_completed / f.steps_total) * 100);
      var wrapper = document.createElement('div');
      wrapper.setAttribute('style', 'display:flex;align-items:center;gap:6px');
      var track = document.createElement('div');
      track.className = 'bar-track';
      track.setAttribute('style', 'width:60px;height:10px');
      var fill = document.createElement('div');
      fill.className = 'bar-fill';
      fill.style.width = pct + '%';
      track.appendChild(fill);
      wrapper.appendChild(track);
      var pctLabel = document.createElement('span');
      pctLabel.className = 'mono';
      pctLabel.textContent = pct + '%';
      wrapper.appendChild(pctLabel);
      progCell.appendChild(wrapper);
    } else {
      progCell.className = 'mono';
      progCell.textContent = '--';
    }
    tr.appendChild(progCell);
    frag.appendChild(tr);
  });
  body.appendChild(frag);
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
Promise.all([fetchStats(), fetchEvents()]);
setInterval(fetchStats, 30000);
