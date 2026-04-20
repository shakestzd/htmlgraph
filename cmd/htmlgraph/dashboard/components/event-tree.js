/* ── <hg-event-tree> Web Component ─────────────────────────── */

class HgEventTree extends HTMLElement {
  constructor() {
    super();
    this.turns = [];
    this.featureTitles = {};
    this.expanded = new Set(JSON.parse(localStorage.getItem('hg-expanded') || '[]'));
    this._filterDebounce = null;
    // OTel data cache, keyed by session_id. Populated from /api/otel/prompts,
    // /api/otel/rollup, and /api/otel/spans after turns load. Absent when
    // the receiver is disabled or no signals have arrived — rendering
    // degrades silently to the non-OTel path.
    this.otelPromptsBySession = {};
    this.otelRollupBySession = {};
    this.otelSpansBySession = {};
  }

  connectedCallback() {
    // At the doorway landing page (root path, no /p/<id>/ prefix) the
    // server holds no per-project DB handles, so /api/events/* 404s.
    // Skip the load + SSE subscription entirely — the event tree only
    // belongs inside a per-project view.
    if (window.location.pathname.indexOf('/p/') !== 0) return;
    this.load();
    this.evtSource = new EventSource(buildProjectUrl('events/stream'));
    this.evtSource.onmessage = (msg) => this.handleSSE(JSON.parse(msg.data));
    this.evtSource.onopen = () => {
      var dot = document.getElementById('conn-dot');
      var label = document.getElementById('conn-label');
      if (dot) dot.className = 'conn-dot live';
      if (label) label.textContent = 'Live';
    };
    this.evtSource.onerror = () => {
      var dot = document.getElementById('conn-dot');
      var label = document.getElementById('conn-label');
      if (dot) dot.className = 'conn-dot dead';
      if (label) label.textContent = 'Disconnected';
      setTimeout(() => {
        if (this.evtSource && this.evtSource.readyState === EventSource.CONNECTING) {
          if (dot) dot.className = 'conn-dot retry';
          if (label) label.textContent = 'Reconnecting...';
        }
      }, 2000);
    };
    this._bindFilterListeners();
  }

  disconnectedCallback() {
    if (this.evtSource) this.evtSource.close();
  }

  _bindFilterListeners() {
    var textEl = document.getElementById('filter-text');
    var toolEl = document.getElementById('filter-tool');
    var agentEl = document.getElementById('filter-agent');
    if (textEl) {
      textEl.addEventListener('input', () => {
        clearTimeout(this._filterDebounce);
        this._filterDebounce = setTimeout(() => this.render(), 200);
      });
    }
    if (toolEl) toolEl.addEventListener('change', () => this.render());
    if (agentEl) agentEl.addEventListener('change', () => this.render());
  }

  async load() {
    var limit = this.dataset.limit || 50;
    try {
      var resp = await fetch(buildProjectUrl('events/tree', 'limit=' + limit));
      if (!resp.ok) return;
      this.turns = await resp.json();
    } catch(e) {
      this.turns = [];
    }
    await this.loadFeatureTitles();
    await this.loadOtelData();
    this.updateCount();
    this._populateDropdowns();
    this.render();
  }

  // loadOtelData fetches per-prompt and per-session OTel aggregates for
  // every session currently in the turn list. One request per session.
  // Silently treats 404 / network errors as "no OTel data" — rendering
  // degrades to the non-OTel path without logging.
  async loadOtelData() {
    var sessionIds = new Set();
    this.turns.forEach(function(t) {
      if (t.user_query && t.user_query.session_id) {
        sessionIds.add(t.user_query.session_id);
      }
    });
    if (sessionIds.size === 0) return;

    var self = this;
    await Promise.all(Array.from(sessionIds).map(async function(sid) {
      try {
        var pResp = await fetch(buildProjectUrl('otel/prompts', 'session_id=' + encodeURIComponent(sid)));
        if (pResp.ok) {
          var body = await pResp.json();
          self.otelPromptsBySession[sid] = body.prompts || [];
        }
      } catch(_) {}
      try {
        var rResp = await fetch(buildProjectUrl('otel/rollup', 'session_id=' + encodeURIComponent(sid)));
        if (rResp.ok) {
          self.otelRollupBySession[sid] = await rResp.json();
        }
      } catch(_) {}
      try {
        var sResp = await fetch(buildProjectUrl('otel/spans', 'session_id=' + encodeURIComponent(sid)));
        if (sResp.ok) {
          var sBody = await sResp.json();
          self.otelSpansBySession[sid] = self._indexSpans(sBody.spans || []);
        }
      } catch(_) {}
    }));
  }

  // _indexSpans groups spans by trace_id and builds parent→children lookup.
  // Returns { byTrace: { traceId: [roots...] }, allSpans: [span with .children] }
  // so we can render either a trace rooted at its interaction span or drill
  // in from any parent. Each span entry gets a .children array populated.
  _indexSpans(spans) {
    var byId = {};
    spans.forEach(function(s) { s.children = []; byId[s.span_id] = s; });
    var roots = [];
    spans.forEach(function(s) {
      if (s.parent_span && byId[s.parent_span]) {
        byId[s.parent_span].children.push(s);
      } else {
        roots.push(s);
      }
    });
    var byTrace = {};
    roots.forEach(function(s) {
      (byTrace[s.trace_id] = byTrace[s.trace_id] || []).push(s);
    });
    return { byTrace: byTrace, roots: roots, byId: byId };
  }

  // _spansForTurn returns the root spans from the trace whose first span's
  // start timestamp is nearest the turn's user_query timestamp. This is the
  // same heuristic as _otelForTurn — a stand-in until native prompt_id ↔
  // trace_id correlation ships. Returns [] when no match.
  _spansForTurn(turn) {
    var uq = turn.user_query;
    if (!uq || !uq.session_id) return [];
    var idx = this.otelSpansBySession[uq.session_id];
    if (!idx || !idx.roots || idx.roots.length === 0) return [];
    if (!uq.timestamp) return idx.roots;
    var ts = Date.parse(uq.timestamp) * 1000;
    if (!ts) return idx.roots;
    // Pick the trace whose earliest root span is nearest to the turn ts.
    var bestTrace = null;
    var bestDiff = Infinity;
    Object.keys(idx.byTrace).forEach(function(tid) {
      var traceRoots = idx.byTrace[tid];
      var earliest = Math.min.apply(null, traceRoots.map(function(r) { return r.ts_micros; }));
      var d = Math.abs(earliest - ts);
      if (d < bestDiff) { bestTrace = tid; bestDiff = d; }
    });
    return bestTrace ? idx.byTrace[bestTrace] : [];
  }

  // _otelForTurn returns the OTel prompt breakdown nearest (by wall-clock)
  // to the turn's user_query timestamp, or null if no OTel data exists for
  // this session. Matching by nearest-timestamp is a stand-in until hooks
  // and OTel events can be joined on a native prompt_id (later phase).
  _otelForTurn(turn) {
    var uq = turn.user_query;
    if (!uq || !uq.session_id) return null;
    var prompts = this.otelPromptsBySession[uq.session_id];
    if (!prompts || prompts.length === 0) return null;
    if (!uq.timestamp) return prompts[0];
    var ts = Date.parse(uq.timestamp) * 1000; // ms → micros
    if (!ts) return prompts[0];
    var best = null;
    var bestDiff = Infinity;
    for (var i = 0; i < prompts.length; i++) {
      var d = Math.abs((prompts[i].first_ts_micros || 0) - ts);
      if (d < bestDiff) { best = prompts[i]; bestDiff = d; }
    }
    return best;
  }

  // _otelBadges returns an HTML fragment with cost/token/retry badges
  // for a turn when OTel data is available; empty string otherwise.
  _otelBadges(turn) {
    var p = this._otelForTurn(turn);
    if (!p) return '';
    var parts = [];
    if (p.cost_usd > 0) parts.push('$' + p.cost_usd.toFixed(4));
    var totalTokens = (p.tokens_in || 0) + (p.tokens_out || 0) + (p.tokens_cache_read || 0) + (p.tokens_cache_creation || 0);
    if (totalTokens > 0) parts.push(this._fmtTokens(totalTokens) + ' tok');
    if (p.api_errors > 0) parts.push(p.api_errors + ' err');
    if (parts.length === 0) return '';
    return '<span class="badge badge-otel" title="From OTel api_request events">'
      + parts.map(esc).join(' · ')
      + '</span>';
  }

  _fmtTokens(n) {
    if (n < 1000) return '' + n;
    if (n < 1000000) return (n / 1000).toFixed(1) + 'k';
    return (n / 1000000).toFixed(2) + 'M';
  }

  async loadFeatureTitles() {
    var ids = new Set();
    this.turns.forEach(function collectIds(t) {
      if (t.user_query && t.user_query.feature_id) ids.add(t.user_query.feature_id);
      (t.children || []).forEach(function walk(c) {
        if (c.feature_id) ids.add(c.feature_id);
        (c.children || []).forEach(walk);
      });
    });
    if (ids.size === 0) return;
    try {
      var resp = await fetch(buildProjectUrl('features'));
      if (!resp.ok) return;
      var features = await resp.json();
      var self = this;
      features.forEach(function(f) { if (ids.has(f.id)) self.featureTitles[f.id] = f.title; });
    } catch(e) { /* non-fatal */ }
  }

  _collectFromChildren(children, tools, agents) {
    (children || []).forEach((c) => {
      if (c.tool_name) tools.add(c.tool_name);
      if (c.agent_id && c.agent_id !== 'claude-code') agents.add(c.agent_id);
      this._collectFromChildren(c.children, tools, agents);
    });
  }

  _populateDropdowns() {
    var tools = new Set();
    var agents = new Set();
    this.turns.forEach((t) => {
      this._collectFromChildren(t.children, tools, agents);
    });

    var toolEl = document.getElementById('filter-tool');
    var agentEl = document.getElementById('filter-agent');
    if (toolEl) {
      var prevTool = toolEl.value;
      toolEl.innerHTML = '<option value="">All Tools</option>';
      Array.from(tools).sort().forEach(function(t) {
        var opt = document.createElement('option');
        opt.value = t;
        opt.textContent = t;
        if (t === prevTool) opt.selected = true;
        toolEl.appendChild(opt);
      });
    }
    if (agentEl) {
      var prevAgent = agentEl.value;
      agentEl.innerHTML = '<option value="">All Agents</option>';
      Array.from(agents).sort().forEach(function(a) {
        var opt = document.createElement('option');
        opt.value = a;
        opt.textContent = a;
        if (a === prevAgent) opt.selected = true;
        agentEl.appendChild(opt);
      });
    }
  }

  getFilterValues() {
    var textEl = document.getElementById('filter-text');
    var toolEl = document.getElementById('filter-tool');
    var agentEl = document.getElementById('filter-agent');
    return {
      text: textEl ? textEl.value.trim().toLowerCase() : '',
      tool: toolEl ? toolEl.value : '',
      agent: agentEl ? agentEl.value : ''
    };
  }

  _turnMatchesFilters(turn, filters) {
    if (!filters.text && !filters.tool && !filters.agent) return true;

    var uq = turn.user_query || {};
    if (filters.text) {
      var summary = (uq.input_summary || '').toLowerCase();
      if (!this._childrenContainText(turn.children, filters.text) && !summary.includes(filters.text)) {
        return false;
      }
    }
    if (filters.tool && !this._childrenContainTool(turn.children, filters.tool)) {
      return false;
    }
    if (filters.agent && !this._childrenContainAgent(turn.children, filters.agent)) {
      return false;
    }
    return true;
  }

  _childrenContainText(children, text) {
    return (children || []).some((c) => {
      var s = ((c.input_summary || '') + ' ' + (c.output_summary || '') + ' ' + (c.tool_name || '')).toLowerCase();
      if (s.includes(text)) return true;
      return this._childrenContainText(c.children, text);
    });
  }

  _childrenContainTool(children, tool) {
    return (children || []).some((c) => {
      if (c.tool_name === tool) return true;
      return this._childrenContainTool(c.children, tool);
    });
  }

  _childrenContainAgent(children, agent) {
    return (children || []).some((c) => {
      if (c.agent_id === agent) return true;
      return this._childrenContainAgent(c.children, agent);
    });
  }

  parseBadgeCategory(title) {
    var prefixes = {
      'Dashboard:': 'badge-dashboard',
      'Fix:': 'badge-fix',
      'Plan ': 'badge-plan',
      'Plan:': 'badge-plan',
      'CLI:': 'badge-cli',
      'Refactor:': 'badge-refactor',
      'Test:': 'badge-test'
    };
    for (var prefix in prefixes) {
      if (title.startsWith(prefix)) {
        return {
          text: title.substring(prefix.length).trim(),
          className: prefixes[prefix]
        };
      }
    }
    return { text: title, className: 'badge-feature' };
  }

  agentBadgeColor(agentType) {
    var type = (agentType || '').toLowerCase();
    if (type.indexOf('researcher') !== -1) return '#06b6d4'; // cyan
    if (type.indexOf('haiku') !== -1) return '#22c55e';      // green
    if (type.indexOf('sonnet') !== -1) return '#3b82f6';     // blue
    if (type.indexOf('opus') !== -1) return '#a855f7';       // purple
    if (type.indexOf('test-runner') !== -1) return '#eab308'; // yellow
    return '#d29922'; // default gold
  }

  featureBadge(featureId, featureTitle) {
    if (!featureId) return '';
    var title = featureTitle || this.featureTitles[featureId] || '';
    // Treat degenerate "title == id" rows as missing — fall back to a
    // short hash label instead of uppercasing the full ID. This happens
    // when the feature was created without a title or the title field
    // was never populated (tracked in a separate data-side bug).
    if (title && title.toLowerCase() === featureId.toLowerCase()) {
      title = '';
    }
    var parsed = this.parseBadgeCategory(title);
    var shortId = featureId.replace(/^(feat|bug|spk|plan|trk)-/, '').substring(0, 6);
    var label = parsed.text
      ? (parsed.text.length > 25 ? parsed.text.substring(0, 22) + '...' : parsed.text)
      : 'untitled ' + shortId;
    return '<span class="badge ' + parsed.className + '" title="' + esc(featureId) + '">' + esc(label) + '</span>';
  }

  updateCount() {
    var countEl = document.getElementById('activity-count');
    if (countEl) countEl.textContent = this.turns.length;
  }

  saveExpanded() {
    localStorage.setItem('hg-expanded', JSON.stringify([...this.expanded]));
  }

  toggle(eventId) {
    if (this.expanded.has(eventId)) {
      this.expanded.delete(eventId);
    } else {
      this.expanded.add(eventId);
    }
    this.saveExpanded();
    this.render();
  }

  handleSSE(data) {
    if (stats.total_events != null) {
      stats.total_events++;
      setVal('sv-events', stats.total_events);
    }
    if (this._reloadTimer) clearTimeout(this._reloadTimer);
    this._reloadTimer = setTimeout(() => this.load(), 500);
  }

  render() {
    if (!this.turns || this.turns.length === 0) {
      this.innerHTML = '<div class="empty-state">No activity yet. Start a Claude Code session to see activity.</div>';
      this._updateFilterCount(0, 0);
      return;
    }

    var filters = this.getFilterValues();
    var filtered = this.turns.filter((t) => this._turnMatchesFilters(t, filters));
    this._updateFilterCount(filtered.length, this.turns.length);
    this.innerHTML = filtered.map(t => this.renderTurn(t)).join('');
  }

  _updateFilterCount(shown, total) {
    var countEl = document.getElementById('filter-count');
    if (!countEl) return;
    countEl.textContent = (shown < total) ? shown + ' of ' + total : '';
  }

  renderTurn(turn) {
    var uq = turn.user_query;
    var isExp = this.expanded.has(uq.event_id);
    var hasChildren = turn.children && turn.children.length > 0;
    var expandIcon = hasChildren
      ? '<span class="expand-icon ' + (isExp ? 'expanded' : '') + '" data-toggle="' + esc(uq.event_id) + '">\u25B6</span>'
      : '<span class="expand-icon-spacer"></span>';

    var s = turn.stats || {};
    var statsHtml = '<span class="turn-stats">' + (s.tool_count || 0) + ' tools' + (s.error_count ? ', ' + s.error_count + ' errors' : '') + '</span>';
    var featureBdg = this.featureBadge(uq.feature_id, uq.feature_title);
    var otelBdg = this._otelBadges(turn);

    var html = '<div class="turn-group">'
      + '<div class="event-row depth-0 user-query-row"'
      + ' data-event-id="' + esc(uq.event_id) + '"'
      + ' data-timestamp="' + esc(uq.timestamp || '') + '">'
      + expandIcon
      + '<span class="event-time">' + formatTime(uq.timestamp) + '</span>'
      + '<span class="event-summary">' + esc(uq.input_summary || '') + '</span>'
      + featureBdg
      + otelBdg
      + statsHtml
      + '</div>';

    if (isExp && turn.children) {
      html += turn.children.map(c => this.renderEvent(c, 1)).join('');
    }

    // OTel span subtree — only when expanded, to avoid bloating collapsed
    // rows. Rendered after hook children so hook-derived hierarchy stays
    // primary and trace detail is supplementary. One rendered tree per
    // turn; rooted at the nearest matching trace's root spans.
    if (isExp) {
      var rootSpans = this._spansForTurn(turn);
      if (rootSpans && rootSpans.length > 0) {
        html += rootSpans.map(s => this.renderSpan(s, 1)).join('');
      }
    }

    html += '</div>';
    return html;
  }

  renderEvent(evt, depth) {
    if (depth > 3) return '';
    var hasChildren = evt.children && evt.children.length > 0;
    var isExp = this.expanded.has(evt.event_id);
    var expandIcon = hasChildren
      ? '<span class="expand-icon ' + (isExp ? 'expanded' : '') + '" data-toggle="' + esc(evt.event_id) + '">\u25B6</span>'
      : '<span class="expand-icon-spacer"></span>';

    var isTask = evt.tool_name === 'Task' || evt.tool_name === 'Agent' || evt.event_type === 'task_delegation';
    var isError = evt.event_type === 'error' || evt.status === 'failed';
    var borderClass = isTask ? 'border-task' : isError ? 'border-error' : '';

    var subagentBadge = '';
    if (isTask && evt.subagent_type) {
      var color = this.agentBadgeColor(evt.subagent_type);
      subagentBadge = '<span class="badge badge-subagent" style="background-color: ' + color + '">' + esc(evt.subagent_type) + '</span>';
    }
    var statusBdg = '<span class="badge badge-status-' + (evt.status || 'unknown') + '">' + esc(evt.status || 'unknown') + '</span>';

    var padLeft = (depth + 1) * 1.25;
    var bgAlpha = 0.05 + depth * 0.08;

    var html = '<div class="event-row depth-' + depth + ' ' + borderClass + ' clickable-row"'
      + ' data-event-id="' + esc(evt.event_id) + '"'
      + (evt.session_id ? ' data-session="' + esc(evt.session_id) + '"' : '')
      + ' data-tool-use-id="' + esc(evt.tool_use_id || '') + '"'
      + ' data-tool-name="' + esc(evt.tool_name || '') + '"'
      + (evt.agent_id ? ' data-agent="' + esc(evt.agent_id) + '"' : '')
      + ' data-timestamp="' + esc(evt.timestamp || '') + '"'
      + ' style="padding-left: ' + padLeft + 'rem; background: rgba(0,0,0,' + bgAlpha + ')">'
      + expandIcon
      + '<span class="event-time">' + formatTime(evt.timestamp) + '</span>'
      + '<span class="tool-chip tool-' + esc(evt.tool_name) + '">' + esc(evt.tool_name) + toolChipRange(evt) + '</span>'
      + subagentBadge
      + '<span class="event-summary">' + esc(evt.input_summary || evt.output_summary || '') + '</span>'
      + this.featureBadge(evt.feature_id, evt.feature_title)
      + statusBdg
      + '</div>';

    if (isExp && evt.children) {
      html += evt.children.map(c => this.renderEvent(c, depth + 1)).join('');
    }
    return html;
  }

  // renderSpan emits a tree row for an OTel span and recursively renders
  // its children. Uses distinct styling (.event-row-otel-span + trace chip)
  // so OTel-sourced rows are visually separable from hook-sourced rows.
  // Span IDs are used as toggle keys, namespaced with "span:" so they
  // don't collide with event_id-keyed toggles from the hook-derived tree.
  renderSpan(span, depth) {
    if (depth > 5) return '';
    if (!span) return '';
    var toggleKey = 'span:' + span.span_id;
    var hasChildren = span.children && span.children.length > 0;
    var isExp = this.expanded.has(toggleKey);
    var expandIcon = hasChildren
      ? '<span class="expand-icon ' + (isExp ? 'expanded' : '') + '" data-toggle="' + esc(toggleKey) + '">\u25B6</span>'
      : '<span class="expand-icon-spacer"></span>';

    // Label: use tool_name when available, otherwise the canonical span
    // name (interaction / api_request / tool_execution / tool_blocked_on_user).
    var label = span.tool_name
      ? span.tool_name
      : (span.canonical || span.native_name || 'span');
    var dur = span.duration_ms ? (span.duration_ms >= 1000 ? (span.duration_ms / 1000).toFixed(2) + 's' : span.duration_ms + 'ms') : '';

    var errBorder = (span.success === false) ? 'border-error' : '';
    var padLeft = (depth + 1) * 1.25;
    var bgAlpha = 0.04 + depth * 0.05;

    var costBdg = (span.cost_usd > 0)
      ? '<span class="badge badge-otel">$' + span.cost_usd.toFixed(4) + '</span>'
      : '';
    var modelBdg = span.model ? '<span class="badge badge-otel">' + esc(span.model) + '</span>' : '';
    var durBdg = dur ? '<span class="turn-stats">' + dur + '</span>' : '';
    var traceChip = '<span class="tool-chip tool-otel-trace" title="OTel span: ' + esc(span.native_name) + '">trace</span>';

    var html = '<div class="event-row event-row-otel-span depth-' + depth + ' ' + errBorder + '"'
      + ' data-span-id="' + esc(span.span_id) + '"'
      + ' data-trace-id="' + esc(span.trace_id) + '"'
      + (span.parent_span ? ' data-parent-span="' + esc(span.parent_span) + '"' : '')
      + ' style="padding-left: ' + padLeft + 'rem; background: rgba(56,139,253,' + bgAlpha + ')">'
      + expandIcon
      + traceChip
      + '<span class="tool-chip tool-otel">' + esc(label) + '</span>'
      + modelBdg
      + costBdg
      + '<span class="event-summary">' + esc(span.native_name) + '</span>'
      + durBdg
      + '</div>';

    if (isExp && hasChildren) {
      html += span.children.map(c => this.renderSpan(c, depth + 1)).join('');
    }
    return html;
  }
}

customElements.define('hg-event-tree', HgEventTree);

// Delegate click events
document.addEventListener('click', function(e) {
  // Expand/collapse toggle takes priority
  var toggle = e.target.closest('[data-toggle]');
  if (toggle) {
    var tree = document.querySelector('hg-event-tree');
    if (tree) tree.toggle(toggle.dataset.toggle);
    return;
  }

  // Clickable event row → drill down to transcript
  var row = e.target.closest('.clickable-row[data-session]');
  if (row) {
    var sid = row.dataset.session;
    var scrollHint = {
      toolUseId: row.dataset.toolUseId || '',
      toolName: row.dataset.toolName || '',
      timestamp: row.dataset.timestamp || ''
    };
    currentView = 'sessions';
    document.querySelectorAll('.nav-btn').forEach(function(b) {
      b.classList.toggle('active', b.dataset.view === 'sessions');
    });
    document.querySelectorAll('.view').forEach(function(v) {
      v.classList.toggle('active', v.id === 'v-sessions');
    });
    openTranscript(sid, scrollHint);
    return;
  }
});
