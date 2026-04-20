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
  //
  // Handles a real-life quirk: Claude Code emits the root `interaction`
  // span only at turn rollup, while child spans (api_request, tool_result,
  // tool.execution, tool.blocked_on_user) export continuously. For
  // in-flight sessions this leaves us with orphan children whose parent
  // span_id references a span we haven't received yet. Treating every
  // orphan as its own root explodes the tree horizontally.
  //
  // Fix: synthesize one placeholder parent per unique orphan parent_span,
  // flagged with is_pending=true so renderSpan can label it "pending
  // root — interaction span not yet received". All orphan children group
  // under their shared synthetic root, and the rendered tree looks right.
  _indexSpans(spans) {
    var byId = {};
    spans.forEach(function(s) { s.children = []; byId[s.span_id] = s; });

    // Identify orphan parents (parent_span non-empty but not in byId).
    var orphanParents = {};
    spans.forEach(function(s) {
      if (s.parent_span && !byId[s.parent_span]) {
        orphanParents[s.parent_span] = {
          span_id: s.parent_span,
          parent_span: '',
          trace_id: s.trace_id,
          canonical: 'interaction',
          native_name: 'claude_code.interaction (pending)',
          tool_name: '',
          model: '',
          ts_micros: s.ts_micros,
          duration_ms: 0,
          tokens_in: 0, tokens_out: 0, cost_usd: 0,
          decision: '',
          details: {},
          children: [],
          _pending: true,
        };
      }
    });
    // Merge synthetic parents into byId so the parent-link step below
    // sees them.
    Object.keys(orphanParents).forEach(function(id) {
      if (!byId[id]) byId[id] = orphanParents[id];
    });

    var roots = [];
    Object.values(byId).forEach(function(s) {
      if (s.parent_span && byId[s.parent_span]) {
        byId[s.parent_span].children.push(s);
      } else {
        roots.push(s);
      }
    });

    // Absorb each api_request span into the tool span that immediately
    // FOLLOWS it in its parent's child list. The api_request is the
    // LLM turn that decided to call that tool, so its model/cost/tokens
    // morally attribute to the tool call. The trailing api_request in a
    // turn (the final response) has no following tool and stays as its
    // own row. Done BEFORE reverse so "preceding" means chronologically
    // earlier.
    Object.values(byId).forEach(function(parent) {
      if (!parent.children || parent.children.length < 2) return;
      var kids = parent.children;
      for (var i = 0; i < kids.length - 1; i++) {
        var cur = kids[i], nxt = kids[i + 1];
        if (cur.canonical === 'api_request' && nxt.tool_name) {
          nxt._precedingApi = cur;
          cur._absorbedInto = nxt.span_id;
        }
      }
      // Drop absorbed api_requests from the child list in place.
      parent.children = kids.filter(function(c) { return !c._absorbedInto; });
    });

    // Reverse every children array so the most recent span renders first,
    // matching the activity feed's overall "newest at top" ordering.
    // The endpoint returns spans in ascending ts_micros order; reversing
    // here keeps the server query simple (it's still chronologically
    // ordered, which helps span-to-span joining) while the UI surface
    // flips to reverse-chronological.
    Object.values(byId).forEach(function(s) {
      if (s.children.length > 1) s.children.reverse();
    });
    roots.reverse();

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
  // its children. Uses the same tool-chip classes as hook-sourced rows
  // (tool-Bash/Read/Edit/etc.) so Bash is green, Read is blue, Agent is
  // pink regardless of where the row came from. The "trace" chip appears
  // only on trace roots — every descendant already inherits the blue
  // left-border + tinted background, which is enough provenance.
  //
  // Summary text prefers tool-specific detail (bash full_command, Read
  // file_path, Grep pattern, Agent subagent_type) over the native span
  // name. This mirrors how hook rows show input_summary rather than
  // "tool_call".
  //
  // Span IDs are used as toggle keys, namespaced with "span:" so they
  // don't collide with event_id-keyed toggles from the hook-derived tree.
  renderSpan(span, depth) {
    if (depth > 5) return '';
    if (!span) return '';

    var toggleKey = 'span:' + span.span_id;
    var hasChildren = span.children && span.children.length > 0;
    // Synthetic pending roots default to expanded so the actual tool
    // activity is visible without an extra click — users shouldn't have
    // to discover a placeholder node just to see their own tool calls.
    var isExp = this.expanded.has(toggleKey) || (span._pending && !this.expanded.has('collapse:' + span.span_id));
    var expandIcon = hasChildren
      ? '<span class="expand-icon ' + (isExp ? 'expanded' : '') + '" data-toggle="' + esc(toggleKey) + '">\u25B6</span>'
      : '<span class="expand-icon-spacer"></span>';

    var d = span.details || {};
    var isToolSpan = Boolean(span.tool_name);
    var isRoot = !span.parent_span;

    // Label + chip class.
    // - Tool spans (tool_name=Bash/Read/Edit/Agent): use existing
    //   .tool-{Name} classes for consistent coloring with hook rows.
    // - Non-tool spans (interaction, llm_request, tool_execution,
    //   tool_blocked_on_user): use a neutral .tool-otel class.
    var label, chipClass, chipStyle = '';
    if (isToolSpan) {
      label = span.tool_name;
      chipClass = 'tool-chip tool-' + span.tool_name;
      // Subagent delegations (Task/Agent tool) take on the agent's
      // color family — researcher=cyan, haiku=green, etc.
      if ((span.tool_name === 'Task' || span.tool_name === 'Agent') && d.subagent_type) {
        chipStyle = ' style="background-color: ' + this.agentBadgeColor(d.subagent_type) + '; color: #ffffff"';
      }
    } else {
      label = this._spanCanonicalLabel(span);
      chipClass = 'tool-chip tool-otel';
    }

    // Summary text: tool-specific detail beats the native span name.
    var summary = this._spanSummary(span);

    var dur = span.duration_ms
      ? (span.duration_ms >= 1000 ? (span.duration_ms / 1000).toFixed(2) + 's' : span.duration_ms + 'ms')
      : '';

    var errBorder = (span.success === false) ? 'border-error' : '';
    var padLeft = (depth + 1) * 1.25;
    var bgAlpha = 0.04 + depth * 0.04;

    var traceChip = '';
    if (isRoot) {
      if (span._pending) {
        traceChip = '<span class="tool-chip tool-otel-trace" title="Root interaction span not yet received — children are grouped here provisionally">pending</span>';
      } else {
        traceChip = '<span class="tool-chip tool-otel-trace" title="OTel trace root: ' + esc(span.native_name) + '">trace</span>';
      }
    }

    var subagentBadge = '';
    if (isToolSpan && (span.tool_name === 'Task' || span.tool_name === 'Agent') && d.subagent_type) {
      var col = this.agentBadgeColor(d.subagent_type);
      subagentBadge = '<span class="badge badge-subagent" style="background-color: ' + col + '">' + esc(d.subagent_type) + '</span>';
    }

    // For tool spans, also surface the preceding api_request (the LLM
    // turn that chose this tool) — its model / cost / duration attribute
    // to "deciding this tool call" and so belong on the tool row.
    var api = (isToolSpan && span._precedingApi) ? span._precedingApi : null;
    var apiModel = (api && api.model) || span.model;
    var apiCost = api ? api.cost_usd : span.cost_usd;

    var modelBdg = apiModel
      ? '<span class="badge badge-otel" title="' + esc(api ? 'Model for the api_request that decided this tool call' : 'Model') + '">' + esc(apiModel) + '</span>'
      : '';
    var costBdg = (apiCost > 0)
      ? '<span class="badge badge-otel">$' + apiCost.toFixed(4) + '</span>'
      : '';
    var retryBdg = (d.attempt && d.attempt > 1)
      ? '<span class="badge badge-otel" title="Attempt number">attempt ' + d.attempt + '</span>'
      : '';
    var durBdg = dur ? '<span class="turn-stats">' + dur + '</span>' : '';

    // For tool spans, roll up the child permission + exec spans into
    // compact badges on this row (instead of forcing the user to expand
    // the tool just to see the outcome). The children still render
    // in full when the tool span is expanded.
    var rollup = this._toolChildRollup(span);
    var permissionBdg = rollup.permissionBadge;
    var execErrorBdg = rollup.execErrorBadge;
    var rangeBdg = this._rangeBadge(span);

    // Hover tooltip: for Bash show the full command (since summary is
    // description); for other tools, show the native span name for
    // provenance. esc() prevents quote-breakout.
    var rowTitle = '';
    if (span.tool_name === 'Bash' && d.full_command) {
      rowTitle = d.full_command;
    } else if (span.native_name) {
      rowTitle = span.native_name;
    }

    var html = '<div class="event-row event-row-otel-span depth-' + depth + ' ' + errBorder + '"'
      + ' data-span-id="' + esc(span.span_id) + '"'
      + ' data-trace-id="' + esc(span.trace_id) + '"'
      + (span.parent_span ? ' data-parent-span="' + esc(span.parent_span) + '"' : '')
      + (rowTitle ? ' title="' + esc(rowTitle) + '"' : '')
      + ' style="padding-left: ' + padLeft + 'rem; background: rgba(56,139,253,' + bgAlpha + ')">'
      + expandIcon
      + traceChip
      + '<span class="' + chipClass + '"' + chipStyle + '>' + esc(label) + '</span>'
      + subagentBadge
      + modelBdg
      + costBdg
      + permissionBdg
      + execErrorBdg
      + retryBdg
      + rangeBdg
      + '<span class="event-summary">' + esc(summary) + '</span>'
      + durBdg
      + '</div>';

    if (isExp && hasChildren) {
      // When expanded, first render a detail block for the tool row
      // itself (full command, timeout, prompt, URL, etc.), then the
      // child spans below it. When collapsed, no detail block — the
      // badges already summarize.
      var detailBlock = this._spanDetailBlock(span, depth + 1);
      if (detailBlock) html += detailBlock;
      html += span.children.map(c => this.renderSpan(c, depth + 1)).join('');
    }
    return html;
  }

  // _spanDetailBlock renders a single fixed-width panel below a tool
  // row when it's expanded, showing the full input context the summary
  // line couldn't fit. Returns '' when there's nothing worth showing.
  _spanDetailBlock(span, depth) {
    var d = span.details || {};
    if (!span.tool_name) return '';
    var rows = [];
    if (span.tool_name === 'Bash') {
      if (d.full_command) rows.push(['command', d.full_command]);
      if (d.description)  rows.push(['description', d.description]);
      if (d.timeout)      rows.push(['timeout', d.timeout + 'ms']);
    } else if (span.tool_name === 'Read') {
      if (d.file_path)    rows.push(['file', d.file_path]);
      if (d.offset || d.limit) {
        var start = d.offset || 1;
        var end = d.limit ? (start + d.limit - 1) : '';
        rows.push(['range', end ? ('lines ' + start + '–' + end) : ('offset ' + start)]);
      }
    } else if (span.tool_name === 'Edit' || span.tool_name === 'Write' || span.tool_name === 'NotebookEdit') {
      if (d.file_path)    rows.push(['file', d.file_path]);
    } else if (span.tool_name === 'Grep' || span.tool_name === 'Glob') {
      if (d.pattern)      rows.push(['pattern', d.pattern]);
    } else if (span.tool_name === 'Task' || span.tool_name === 'Agent') {
      if (d.subagent_type) rows.push(['subagent', d.subagent_type]);
      if (d.description)   rows.push(['prompt', d.description]);
    } else if (span.tool_name === 'WebFetch' || span.tool_name === 'WebSearch') {
      if (d.url)          rows.push(['url', d.url]);
    } else if (span.tool_name === 'Skill') {
      if (d.skill_name)   rows.push(['skill', d.skill_name]);
    }
    // Preceding api_request: if absorbed, show the details we hid from
    // the top-level tree so expanding the tool reveals the full context.
    if (span._precedingApi) {
      var api = span._precedingApi;
      var ad = api.details || {};
      if (api.model)            rows.push(['model', api.model]);
      if (api.tokens_in)        rows.push(['input tokens', api.tokens_in.toLocaleString()]);
      if (api.tokens_out)       rows.push(['output tokens', api.tokens_out.toLocaleString()]);
      if (api.cost_usd > 0)     rows.push(['cost', '$' + api.cost_usd.toFixed(6)]);
      if (api.duration_ms)      rows.push(['api duration', api.duration_ms + 'ms']);
      if (ad.request_id)        rows.push(['request id', ad.request_id]);
      if (ad.speed)             rows.push(['mode', ad.speed]);
    }
    if (rows.length === 0) return '';

    var padLeft = (depth + 1) * 1.25;
    var bgAlpha = 0.03 + depth * 0.03;
    var kvHtml = rows.map(function(r) {
      return '<div class="otel-detail-row"><span class="otel-detail-key">' + esc(r[0]) + '</span>'
        + '<span class="otel-detail-val">' + esc(String(r[1])) + '</span></div>';
    }).join('');
    return '<div class="event-row event-row-otel-detail depth-' + depth + '"'
      + ' style="padding-left: ' + padLeft + 'rem; background: rgba(56,139,253,' + bgAlpha + ')">'
      + kvHtml
      + '</div>';
  }

  // _toolChildRollup scans a tool span's immediate children for
  // infrastructure spans (permission + exec) and returns compact badges
  // that summarize the outcomes. Only applies to tool_result canonical
  // spans with tool_name set — leaves non-tool spans alone.
  //
  // Badges:
  //   permissionBadge — green "✓ auto" / "✓ user" chip for approvals,
  //                     red "✗ blocked" chip for user rejections. Empty
  //                     when no permission child or source=unknown.
  //   execErrorBadge  — red "failed" chip when exec child reports
  //                     success=false. Empty on success or no exec child.
  _toolChildRollup(span) {
    var empty = { permissionBadge: '', execErrorBadge: '' };
    if (span.canonical !== 'tool_result' || !span.tool_name || !span.children) {
      return empty;
    }
    var perm = span.children.find(function(c) { return c.canonical === 'tool_blocked_on_user'; });
    var exec = span.children.find(function(c) { return c.canonical === 'tool_execution'; });

    var permBadge = '';
    if (perm) {
      var d = perm.details || {};
      switch (d.decision_source) {
        case 'config':
        case 'hook':
          permBadge = '<span class="badge badge-approve" title="Auto-approved (' + esc(d.decision_source) + ')">\u2713 auto</span>';
          break;
        case 'user_permanent':
        case 'user_temporary':
          permBadge = '<span class="badge badge-approve" title="User approved (' + esc(d.decision_source) + ')">\u2713 user</span>';
          break;
        case 'user_reject':
          permBadge = '<span class="badge badge-reject" title="User rejected the tool call">\u2717 blocked</span>';
          break;
        case 'user_abort':
          permBadge = '<span class="badge badge-reject" title="User aborted the turn">\u2717 aborted</span>';
          break;
        default: // unknown / empty — omit
          permBadge = '';
      }
    }

    var execBadge = '';
    if (exec && exec.success === false) {
      execBadge = '<span class="badge badge-reject" title="Tool execution reported failure">failed</span>';
    }

    return { permissionBadge: permBadge, execErrorBadge: execBadge };
  }

  // _rangeBadge renders a compact line-range badge for Read/Edit tool
  // spans when offset/limit or old_string/new_string context is present
  // in the span's details. Returns an empty string when no range data
  // is available (most Read calls don't pass offset/limit).
  _rangeBadge(span) {
    if (!span.tool_name) return '';
    var d = span.details || {};
    if (span.tool_name === 'Read') {
      if (d.offset || d.limit) {
        var start = d.offset || 1;
        var end = d.limit ? (start + d.limit - 1) : '?';
        return '<span class="badge badge-otel" title="Line range">L' + start + '\u2013' + end + '</span>';
      }
    }
    // Edit/Write/NotebookEdit don't currently expose a range in the
    // span's attrs; that data lives on the tool_result log's tool_input.
    // A follow-up can surface it once we join spans to logs.
    return '';
  }

  // _spanCanonicalLabel returns a short human label for non-tool spans.
  // Maps claude_code.interaction → "interaction", claude_code.llm_request
  // → "api_request", tool.execution → "exec", tool.blocked_on_user →
  // "permission", etc.
  _spanCanonicalLabel(span) {
    switch (span.canonical) {
      case 'interaction':           return 'interaction';
      case 'api_request':           return 'api_request';
      case 'tool_execution':        return 'exec';
      case 'tool_blocked_on_user':  return 'permission';
      default:
        // Strip the harness prefix if present (claude_code.*).
        var n = span.native_name || '';
        var i = n.indexOf('.');
        return i >= 0 ? n.slice(i + 1) : (n || 'span');
    }
  }

  // _spanSummary returns the descriptive text for a span row.
  // Priority: tool-specific detail (command, file path, etc.) > a
  // derived label for infrastructure spans > the native name as a
  // fallback so rows are never empty.
  //
  // For Bash: prefer the human "description" over the raw command —
  // "Push rollup + enrichment commit" scans faster than a 60-char shell
  // string. The full command is exposed via a hover title + detail row
  // on expand.
  _spanSummary(span) {
    var d = span.details || {};
    if (span.tool_name === 'Bash') {
      return d.description || d.full_command || d.bash_command || '';
    }
    if (span.tool_name === 'Read' || span.tool_name === 'Edit' || span.tool_name === 'Write' || span.tool_name === 'NotebookEdit') {
      return d.file_path || '';
    }
    if (span.tool_name === 'Grep' || span.tool_name === 'Glob') {
      return d.pattern || '';
    }
    if (span.tool_name === 'WebFetch' || span.tool_name === 'WebSearch') {
      return d.url || '';
    }
    if (span.tool_name === 'Skill') {
      return d.skill_name || '';
    }
    if (span.tool_name === 'Task' || span.tool_name === 'Agent') {
      // For subagent delegations, surface the description/prompt-summary
      // ("Multi-tool subagent for span nesting verification") since the
      // agent name is already shown as a distinct chip.
      return d.description || '';
    }
    // tool_blocked_on_user is a misleading name — it's the PERMISSION
    // GATE span covering the time between tool emit and execution. It
    // does NOT mean the tool was blocked. The decision_source attribute
    // tells us how the gate resolved:
    //   config          → auto-approved via settings / --allowedTools
    //   hook            → auto-approved via a PreToolUse hook
    //   user_permanent  → user approved, remembered for the session
    //   user_temporary  → user approved, single-use
    //   user_reject     → user blocked the call
    //   user_abort      → user cancelled the turn
    //   unknown         → no permission decision recorded
    if (span.canonical === 'tool_blocked_on_user') {
      switch (d.decision_source) {
        case 'config':         return 'auto-approved (config)';
        case 'hook':           return 'auto-approved (hook)';
        case 'user_permanent': return 'user approved (remember)';
        case 'user_temporary': return 'user approved';
        case 'user_reject':    return 'blocked by user';
        case 'user_abort':     return 'aborted by user';
        case '':
        case 'unknown':
        case undefined:
        case null:             return 'permission check';
        default:               return 'decision: ' + d.decision_source;
      }
    }
    if (span.canonical === 'tool_execution') {
      return 'executed';
    }
    if (span.canonical === 'interaction') {
      return '';
    }
    if (span.canonical === 'api_request') {
      return d.speed === 'fast' ? 'fast mode' : '';
    }
    return '';
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
