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

  // _spansForTurn returns the root spans from the trace produced BY
  // this turn. A trace belongs to turn N when its earliest span timestamp
  // sits in the window [turn_N.ts, turn_N+1.ts). That's the causal
  // ordering: a tool call happens strictly AFTER the user prompt that
  // triggered it. Using absolute-time "nearest" is wrong because it can
  // pull a previous turn's trace (slightly earlier) instead of THIS
  // turn's in-flight trace (slightly later).
  //
  // Falls back to absolute-nearest only when no forward-window match
  // exists (handles pathological cases like mid-session resumes where
  // hook-timestamp alignment can drift).
  _spansForTurn(turn) {
    var uq = turn.user_query;
    if (!uq || !uq.session_id) return [];
    var idx = this.otelSpansBySession[uq.session_id];
    if (!idx || !idx.roots || idx.roots.length === 0) return [];
    if (!uq.timestamp) return idx.roots;
    var ts = Date.parse(uq.timestamp) * 1000;
    if (!ts) return idx.roots;

    // Find the next turn in THIS session (chronological next). tree.turns
    // is newest-first, so the "next" turn chronologically is the one with
    // a larger timestamp than ts within the same session.
    var sid = uq.session_id;
    var nextTs = Infinity;
    for (var i = 0; i < (this.turns||[]).length; i++) {
      var otherUQ = this.turns[i].user_query;
      if (!otherUQ || otherUQ.session_id !== sid) continue;
      if (!otherUQ.timestamp) continue;
      var otherTs = Date.parse(otherUQ.timestamp) * 1000;
      if (otherTs > ts && otherTs < nextTs) nextTs = otherTs;
    }

    // Cap the latest turn's window at ts + 15 minutes. Without a cap,
    // every span after the latest KNOWN turn gets attributed to it —
    // including spans from LATER turns we haven't loaded yet (SSE lag,
    // UserPromptSubmit hook failures, etc.). A 15-minute ceiling keeps
    // attribution conservative: live in-flight traces finish well
    // within that window, and stray later-turn spans are dropped rather
    // than misattributed.
    var latestTurnCap = 15 * 60 * 1_000_000; // 15 min in micros
    if (nextTs === Infinity) {
      nextTs = ts + latestTurnCap;
    }

    // Window match: smallest earliest_ts that falls in [ts - slop, nextTs).
    // 1-second slop on the lower bound absorbs harmless clock skew — OTel
    // exporters can start a span a few hundred ms BEFORE the hook-logged
    // user_query timestamp when they're batching aggressively. Any larger
    // slop risks stealing a previous turn's trailing trace.
    //
    // No absolute-nearest fallback. A turn without a forward-window
    // trace genuinely has no OTel data (pre-receiver session, or a turn
    // whose exporter dropped). Returning [] is the honest answer — the
    // renderer then falls through to hook-derived children.
    var slop = 1_000_000; // 1 s in micros
    var winner = null, winnerEarliest = Infinity;
    Object.keys(idx.byTrace).forEach(function(tid) {
      var earliest = Math.min.apply(null, idx.byTrace[tid].map(function(r) { return r.ts_micros; }));
      if (earliest >= (ts - slop) && earliest < nextTs && earliest < winnerEarliest) {
        winner = tid;
        winnerEarliest = earliest;
      }
    });
    if (!winner) return [];

    // Peel off the "interaction" wrapper (or synthetic pending root)
    // so the tool spans become the turn's direct children. The user
    // prompt IS the turn root — rendering "interaction" as an
    // intermediate row just repeats the same information at two depths.
    // Real tool calls (Bash, Edit, MCP, etc.) should attach straight
    // to the user query. Walk any interaction/pending roots one level
    // down; leave non-wrapper roots untouched.
    var roots = idx.byTrace[winner];
    var flat = [];
    roots.forEach(function(r) {
      var isWrapper = r._pending || r.canonical === 'interaction';
      if (isWrapper && r.children && r.children.length) {
        flat = flat.concat(r.children);
      } else {
        flat.push(r);
      }
    });
    return flat;
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
      // Record explicit collapse so auto-expand-newest-turn doesn't
      // immediately re-expand on the next render. Limited to the most
      // recent 10 collapses — older entries age out automatically.
      var collapsed = JSON.parse(localStorage.getItem('hg-collapsed') || '[]');
      collapsed = collapsed.filter(function(id) { return id !== eventId; });
      collapsed.unshift(eventId);
      if (collapsed.length > 10) collapsed = collapsed.slice(0, 10);
      localStorage.setItem('hg-collapsed', JSON.stringify(collapsed));
    } else {
      this.expanded.add(eventId);
      // Clear from collapsed list on re-expansion so auto-expand can
      // reclaim it later if it becomes the top turn again.
      var collapsed2 = JSON.parse(localStorage.getItem('hg-collapsed') || '[]');
      collapsed2 = collapsed2.filter(function(id) { return id !== eventId; });
      localStorage.setItem('hg-collapsed', JSON.stringify(collapsed2));
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

    // Auto-expand the newest turn so live activity is visible without
    // a manual click. The user's explicit collapse wins — if they've
    // toggled off the top turn, we respect that via hg-collapsed. Older
    // turns stay user-controlled via the existing localStorage-backed
    // expand set.
    var topTurn = this.turns[0];
    if (topTurn && topTurn.user_query && topTurn.user_query.event_id) {
      var collapsed = JSON.parse(localStorage.getItem('hg-collapsed') || '[]');
      if (collapsed.indexOf(topTurn.user_query.event_id) === -1) {
        this.expanded.add(topTurn.user_query.event_id);
      }
    }

    var filters = this.getFilterValues();
    var filtered = this.turns.filter((t) => this._turnMatchesFilters(t, filters));
    this._updateFilterCount(filtered.length, this.turns.length);
    this.innerHTML = filtered.map(t => this.renderTurn(t)).join('');

    // Syntax-highlight any newly-injected <code class="language-xxx">
    // blocks. Prism.highlightAllUnder walks the subtree and tokenizes
    // each element that hasn't been highlighted yet. Silent no-op when
    // Prism isn't loaded (e.g. offline, CDN unreachable) — code still
    // renders as plain monospace.
    if (typeof Prism !== 'undefined' && Prism.highlightAllUnder) {
      try { Prism.highlightAllUnder(this); } catch (_) {}
    }
  }

  _updateFilterCount(shown, total) {
    var countEl = document.getElementById('filter-count');
    if (!countEl) return;
    countEl.textContent = (shown < total) ? shown + ' of ' + total : '';
  }

  renderTurn(turn) {
    var uq = turn.user_query;
    var isExp = this.expanded.has(uq.event_id);
    // A turn has children when EITHER hook events OR an OTel trace
    // exists. Previously we only checked turn.children (hook-derived),
    // so turns with only OTel data rendered chevron-less and couldn't
    // be collapsed — making the tree hard to navigate.
    var hasHookChildren = turn.children && turn.children.length > 0;
    var hasSpans = this._spansForTurn(turn).length > 0;
    var hasChildren = hasHookChildren || hasSpans;
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

    if (isExp) {
      // Prefer OTel spans when present — they're the canonical source
      // of hierarchy (subagent tool calls nest natively, no custom
      // attribution logic needed). When a turn has no OTel data — e.g.
      // a pre-OTel session or a session without the receiver enabled —
      // fall back to hook-derived children so older content still
      // renders.
      var rootSpans = this._spansForTurn(turn);
      if (rootSpans && rootSpans.length > 0) {
        html += rootSpans.map(s => this.renderSpan(s, 1)).join('');
      } else if (turn.children) {
        html += turn.children.map(c => this.renderEvent(c, 1)).join('');
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

    // Label + chip class + optional MCP-server pill.
    // - Built-in tool spans (Bash/Read/Edit/Agent/...): use existing
    //   .tool-{Name} classes for color consistency with hook rows.
    // - MCP tools (name pattern mcp__server__tool): strip the prefix,
    //   render a small color-coded server pill + the tool's own name.
    // - Non-tool spans (interaction, llm_request, tool_execution,
    //   tool_blocked_on_user): neutral .tool-otel class.
    var label, chipClass, chipStyle = '', mcpServerPill = '';
    if (isToolSpan) {
      var mcp = this._parseMCPToolName(span.tool_name);
      if (mcp) {
        label = mcp.toolName;
        chipClass = 'tool-chip tool-mcp';
        mcpServerPill = '<span class="tool-chip tool-mcp-server" '
          + 'style="background-color: ' + this._mcpServerColor(mcp.serverName) + '; color: #ffffff"'
          + ' title="MCP server">' + esc(mcp.serverName) + '</span>';
      } else {
        label = span.tool_name;
        chipClass = 'tool-chip tool-' + span.tool_name;
      }
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
    // Match hook-row visual treatment: same padding ladder, same
    // bg-alpha ladder, same base row class. Span rows no longer look
    // visually distinct from hook rows — the span tree IS the tree now.
    var padLeft = (depth + 1) * 1.25;
    var bgAlpha = 0.05 + depth * 0.08;

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

    // Work-item attribution: span rows mirror hook rows by rendering the
    // feature badge when feature_id is populated. Attribution comes from
    // active_work_items at ingest time (see writer.go); this lights up
    // only for sessions whose root agent claimed a work item before the
    // signal arrived. Pre-attribution rows stay silent.
    var featureBdg = (isToolSpan && span.feature_id)
      ? this.featureBadge(span.feature_id, span.feature_title)
      : '';

    // For tool spans, also surface the preceding api_request (the LLM
    // turn that chose this tool) — its model / cost / duration attribute
    // to "deciding this tool call" and so belong on the tool row.
    var api = (isToolSpan && span._precedingApi) ? span._precedingApi : null;
    var apiModel = (api && api.model) || span.model;
    var apiCost = api ? api.cost_usd : span.cost_usd;

    // Compact, color-coded model badge. Inline-styled so each family
    // (Opus/Sonnet/Haiku, plus generic for OpenAI/Google) gets its
    // own color without inflating CSS. The short label ("Opus 4.7",
    // "Sonnet 4.6", "Haiku 4.5") scans far faster than the full
    // "claude-opus-4-7-20251014" id at row scale.
    var modelBdg = apiModel ? this._modelBadge(apiModel, api ? 'Model for the api_request that decided this tool call' : 'Model') : '';
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

    var html = '<div class="event-row depth-' + depth + ' ' + errBorder + '"'
      + ' data-span-id="' + esc(span.span_id) + '"'
      + ' data-trace-id="' + esc(span.trace_id) + '"'
      + (span.parent_span ? ' data-parent-span="' + esc(span.parent_span) + '"' : '')
      + (rowTitle ? ' title="' + esc(rowTitle) + '"' : '')
      + ' style="padding-left: ' + padLeft + 'rem; background: rgba(0,0,0,' + bgAlpha + ')">'
      + expandIcon
      + traceChip
      + mcpServerPill
      + '<span class="' + chipClass + '"' + chipStyle + '>' + esc(label) + '</span>'
      + subagentBadge
      + featureBdg
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
      // command becomes a <pre><code> code block below — it's typically
      // multiline and long enough to deserve code rendering. Keep
      // description/timeout/git-commit as simple kv rows.
      if (d.description)    rows.push(['description', d.description]);
      if (d.timeout)        rows.push(['timeout', d.timeout + 'ms']);
      if (d.git_commit_id)  rows.push(['git commit', d.git_commit_id]);
    } else if (span.tool_name === 'Read') {
      if (d.file_path)      rows.push(['file', d.file_path]);
      if (d.offset || d.limit) {
        var start = d.offset || 1;
        var end = d.limit ? (start + d.limit - 1) : '';
        rows.push(['range', end ? ('lines ' + start + '–' + end) : ('offset ' + start)]);
      }
    } else if (span.tool_name === 'Edit') {
      if (d.file_path)      rows.push(['file', d.file_path]);
      if (d.old_string_len || d.new_string_len) {
        rows.push(['change', '\u2212' + (d.old_string_len || 0) + ' \u2192 +' + (d.new_string_len || 0) + ' chars']);
      }
      if (d.replace_all)    rows.push(['replace_all', 'true']);
    } else if (span.tool_name === 'Write') {
      if (d.file_path)      rows.push(['file', d.file_path]);
      if (d.content_len)    rows.push(['content', d.content_len + ' chars']);
    } else if (span.tool_name === 'NotebookEdit') {
      if (d.file_path)      rows.push(['notebook', d.file_path]);
    } else if (span.tool_name === 'Grep') {
      if (d.pattern)        rows.push(['pattern', d.pattern]);
      if (d.path)           rows.push(['path', d.path]);
      if (d.output_mode)    rows.push(['output', d.output_mode]);
    } else if (span.tool_name === 'Glob') {
      if (d.pattern)        rows.push(['pattern', d.pattern]);
      if (d.path)           rows.push(['path', d.path]);
    } else if (span.tool_name === 'Task' || span.tool_name === 'Agent') {
      if (d.subagent_type)  rows.push(['subagent', d.subagent_type]);
      if (d.description)    rows.push(['description', d.description]);
      if (d.prompt)         rows.push(['prompt', d.prompt]);
    } else if (span.tool_name === 'WebFetch') {
      if (d.url)            rows.push(['url', d.url]);
    } else if (span.tool_name === 'WebSearch') {
      if (d.query)          rows.push(['query', d.query]);
    } else if (span.tool_name === 'Skill') {
      if (d.skill_name)     rows.push(['skill', d.skill_name]);
    } else if (span.tool_name === 'TodoWrite') {
      if (d.todo_count)     rows.push(['todos', d.todo_count]);
    } else if (span.tool_name === 'TaskCreate' || span.tool_name === 'TaskUpdate' ||
               span.tool_name === 'TaskList'   || span.tool_name === 'TaskGet' ||
               span.tool_name === 'TaskStop'   || span.tool_name === 'TaskOutput') {
      // Task-management family — show whatever identifying args the
      // tool_input carried (description, prompt summary, task id).
      if (d.description)    rows.push(['description', d.description]);
      if (d.prompt)         rows.push(['prompt', d.prompt]);
      if (d.subagent_type)  rows.push(['subagent', d.subagent_type]);
    } else if (span.tool_name && span.tool_name.indexOf('mcp__') === 0) {
      // MCP tool: show server + tool split + any common args.
      var mcp = this._parseMCPToolName(span.tool_name);
      if (mcp) {
        rows.push(['server', mcp.serverName]);
        rows.push(['tool', mcp.toolName]);
      }
      if (d.url)            rows.push(['url', d.url]);
      if (d.query)          rows.push(['query', d.query]);
      if (d.pattern)        rows.push(['pattern', d.pattern]);
      if (d.file_path)      rows.push(['file', d.file_path]);
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
    // Long-content code panels: render Bash command / Edit old_string /
    // Edit new_string / Write content as <pre><code class="language-xxx">
    // blocks. The language class is set from the file extension (for
    // file-backed tools) or "bash" for Bash. A future syntax-highlighting
    // library (feat-292f87fe) will pick up the class and colorize.
    var codeBlocks = '';
    if (span.tool_name === 'Bash') {
      if (d.full_command) {
        codeBlocks += this._codeBlock('command', d.full_command, d.full_command.length, false, 'bash');
      }
    } else if (span.tool_name === 'Edit') {
      var editLang = this._detectLanguage(d.file_path);
      if (d.old_string) {
        codeBlocks += this._codeBlock('old_string', d.old_string, d.old_string_len, d.content_truncated, editLang);
      }
      if (d.new_string) {
        codeBlocks += this._codeBlock('new_string', d.new_string, d.new_string_len, d.content_truncated, editLang);
      }
    } else if (span.tool_name === 'Write') {
      if (d.content) {
        codeBlocks += this._codeBlock('content', d.content, d.content_len, d.content_truncated, this._detectLanguage(d.file_path));
      }
    }

    if (rows.length === 0 && !codeBlocks) return '';

    var padLeft = (depth + 1) * 1.25;
    var bgAlpha = 0.05 + depth * 0.08;
    var kvHtml = rows.map(function(r) {
      return '<div class="otel-detail-row"><span class="otel-detail-key">' + esc(r[0]) + '</span>'
        + '<span class="otel-detail-val">' + esc(String(r[1])) + '</span></div>';
    }).join('');
    return '<div class="event-row event-row-otel-detail depth-' + depth + '"'
      + ' style="padding-left: ' + padLeft + 'rem; background: rgba(0,0,0,' + bgAlpha + ')">'
      + kvHtml
      + codeBlocks
      + '</div>';
  }

  // _codeBlock emits a labeled <pre><code> block for a string attribute
  // (bash command, edit old_string/new_string, write content). The full
  // length is shown in the header so users know whether truncation lost
  // content. The language-* class on <code> lets a future syntax
  // highlighter (Prism / highlight.js per feat-292f87fe) colorize the
  // block without further JS changes.
  _codeBlock(label, content, fullLen, wasTruncated, language) {
    var header = esc(label);
    if (fullLen) header += ' (' + fullLen + ' chars' + (wasTruncated ? ', truncated' : '') + ')';
    if (language) header += ' · ' + esc(language);
    var codeClass = language ? ' class="language-' + esc(language) + '"' : '';
    return '<div class="otel-detail-code">'
      + '<div class="otel-detail-code-header">' + header + '</div>'
      + '<pre class="otel-detail-code-body"><code' + codeClass + '>' + esc(content) + '</code></pre>'
      + '</div>';
  }

  // _detectLanguage maps a file path extension to a Prism/highlight.js
  // language identifier. Returns empty string when the extension isn't
  // recognized (code block still renders, just without language class).
  // Extend this as we add languages — the full list tracked in
  // feat-292f87fe.
  _detectLanguage(filePath) {
    if (!filePath) return '';
    var dot = filePath.lastIndexOf('.');
    if (dot === -1) return '';
    var ext = filePath.slice(dot + 1).toLowerCase();
    switch (ext) {
      case 'go':                 return 'go';
      case 'js': case 'mjs':     return 'javascript';
      case 'ts': case 'tsx':     return 'typescript';
      case 'jsx':                return 'jsx';
      case 'py':                 return 'python';
      case 'rb':                 return 'ruby';
      case 'rs':                 return 'rust';
      case 'java':               return 'java';
      case 'c': case 'h':        return 'c';
      case 'cpp': case 'cc': case 'hpp': return 'cpp';
      case 'cs':                 return 'csharp';
      case 'sh': case 'bash':    return 'bash';
      case 'zsh':                return 'bash';
      case 'fish':               return 'bash';
      case 'html': case 'htm':   return 'html';
      case 'css':                return 'css';
      case 'scss': case 'sass':  return 'scss';
      case 'json':               return 'json';
      case 'yaml': case 'yml':   return 'yaml';
      case 'toml':               return 'toml';
      case 'xml':                return 'xml';
      case 'md': case 'markdown': return 'markdown';
      case 'sql':                return 'sql';
      case 'proto':              return 'protobuf';
      case 'dockerfile':         return 'docker';
      default:                   return '';
    }
  }

  // _toolChildRollup scans a tool span's immediate children for
  // infrastructure spans (permission + exec) and returns badges that
  // surface only the meaningful outcomes.
  //
  // Auto-approval (config / hook) is the silent happy path — don't
  // render a chip for it (would clutter every row with "✓ auto"). A
  // tiny dot on the tool row's left indicates "yes this ran" if
  // visual presence is needed; see .event-row-otel-span::before in
  // components.css.
  //
  // User-approved, blocked, rejected, aborted, and exec-failed states
  // DO get loud badges — they're the exceptional cases worth seeing.
  _toolChildRollup(span) {
    var empty = { permissionBadge: '', execErrorBadge: '' };
    if ((span.canonical !== 'tool_result' && span.canonical !== 'subagent_invocation') || !span.tool_name || !span.children) {
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
          // Auto-approved — omit. Provenance survives on the tool row's
          // title tooltip and the detail panel.
          permBadge = '';
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

  // _modelBadge returns a compact color-coded model chip. Maps Claude
  // model families to the existing agent palette (opus=purple,
  // sonnet=blue, haiku=green) so trace rows share color semantics with
  // subagent badges. Unknown/third-party models fall back to the
  // generic .badge-otel style.
  _modelBadge(model, title) {
    var short = this._shortModelName(model);
    var color = this._modelColor(model);
    if (!color) {
      return '<span class="badge badge-otel" title="' + esc(title || 'Model: ' + model) + '">' + esc(short) + '</span>';
    }
    return '<span class="badge badge-model" style="background-color: ' + color + '; color: #ffffff"'
      + ' title="' + esc((title ? title + '\n' : '') + model) + '">' + esc(short) + '</span>';
  }

  // _shortModelName trims Claude's verbose ids to a human label.
  // claude-opus-4-7              → Opus 4.7
  // claude-opus-4-7-20251014     → Opus 4.7
  // claude-sonnet-4-6-20251005   → Sonnet 4.6
  // claude-haiku-4-5-20251001    → Haiku 4.5
  // gpt-5, gpt-4.1-mini          → passed through
  _shortModelName(model) {
    if (!model) return '';
    var m = model.toLowerCase();
    var match = m.match(/^claude-(opus|sonnet|haiku)-(\d+)-(\d+)/);
    if (match) {
      var family = match[1].charAt(0).toUpperCase() + match[1].slice(1);
      return family + ' ' + match[2] + '.' + match[3];
    }
    // Strip trailing -YYYYMMDD date stamps for other providers too.
    return model.replace(/-\d{8}$/, '');
  }

  _modelColor(model) {
    if (!model) return '';
    var m = model.toLowerCase();
    if (m.indexOf('opus') !== -1)   return '#a855f7'; // purple — matches badge-agent-opus
    if (m.indexOf('sonnet') !== -1) return '#3b82f6'; // blue   — matches badge-agent-sonnet
    if (m.indexOf('haiku') !== -1)  return '#22c55e'; // green  — matches badge-agent-haiku
    return ''; // unknown → fall back to neutral
  }

  // _parseMCPToolName splits "mcp__<server>__<tool>" into its parts so
  // the renderer can show the server as a separate pill and the tool
  // name unqualified. Returns null when the name doesn't match the
  // MCP convention, so callers fall through to standard rendering.
  _parseMCPToolName(name) {
    if (!name || name.indexOf('mcp__') !== 0) return null;
    var rest = name.slice(5); // drop "mcp__"
    var sep = rest.indexOf('__');
    if (sep <= 0) return { serverName: rest, toolName: '' };
    return {
      serverName: rest.slice(0, sep),
      toolName: rest.slice(sep + 2),
    };
  }

  // _mcpServerColor returns a deterministic color for a given MCP
  // server name so multiple tools from the same server share a color.
  // Hash the name to an HSL hue, keep saturation+lightness constant so
  // every server gets a distinct but consistent tint. No static map —
  // new servers get a color automatically.
  _mcpServerColor(serverName) {
    var hash = 0;
    for (var i = 0; i < serverName.length; i++) {
      hash = ((hash << 5) - hash) + serverName.charCodeAt(i);
      hash |= 0; // force int32
    }
    var hue = Math.abs(hash) % 360;
    return 'hsl(' + hue + ', 55%, 45%)';
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
    if (span.tool_name === 'TodoWrite') {
      return d.todo_count ? d.todo_count + ' todos' : '';
    }
    if (span.tool_name === 'TaskCreate' || span.tool_name === 'TaskUpdate' ||
        span.tool_name === 'TaskList'   || span.tool_name === 'TaskGet'    ||
        span.tool_name === 'TaskStop'   || span.tool_name === 'TaskOutput') {
      return d.description || d.subagent_type || '';
    }
    // MCP tools: when tool_input carries something obviously summarizable
    // use it; otherwise leave empty (expand to see full args).
    if (span.tool_name && span.tool_name.indexOf('mcp__') === 0) {
      return d.url || d.query || d.pattern || '';
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
