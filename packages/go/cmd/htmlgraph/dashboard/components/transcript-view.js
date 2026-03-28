/* ── Transcript detail view ─────────────────────────────────── */

function openTranscript(sessionId, scrollToToolUseId) {
  document.getElementById('sessions-list-view').style.display = 'none';
  var detail = document.getElementById('transcript-detail');
  detail.className = 'transcript-detail active';
  document.getElementById('transcript-messages').textContent = 'Loading...';
  document.getElementById('transcript-stats').textContent = '';

  fetch('/api/transcript?session=' + encodeURIComponent(sessionId) + '&limit=500')
    .then(function(r) {
      if (!r.ok) throw new Error('HTTP ' + r.status);
      return r.json();
    })
    .then(function(data) { renderTranscript(data, scrollToToolUseId); })
    .catch(function(err) {
      document.getElementById('transcript-messages').textContent = 'Failed to load transcript: ' + err.message;
    });
}

function closeTranscript() {
  document.getElementById('transcript-detail').className = 'transcript-detail';
  document.getElementById('sessions-list-view').style.display = '';
}

document.getElementById('transcript-back').addEventListener('click', closeTranscript);

function renderTranscript(data, scrollToToolUseId) {
  renderTranscriptStats(data);
  renderTranscriptMessages(data.messages || [], scrollToToolUseId);
}

function renderTranscriptStats(data) {
  var container = document.getElementById('transcript-stats');
  container.textContent = '';
  container.className = 'transcript-stats';

  var msgs = data.messages || [];
  var totalInput = 0, totalOutput = 0, totalCache = 0;
  var model = '';
  msgs.forEach(function(m) {
    totalInput += m.input_tokens || 0;
    totalOutput += m.output_tokens || 0;
    totalCache += m.cache_read_tokens || 0;
    if (m.model && !model) model = m.model;
  });

  var firstTs = msgs.length > 0 ? msgs[0].timestamp : '';
  var lastTs = msgs.length > 0 ? msgs[msgs.length - 1].timestamp : '';
  var duration = '';
  if (firstTs && lastTs) {
    var diffMs = Math.abs(new Date(lastTs).getTime() - new Date(firstTs).getTime());
    if (diffMs > 0) {
      var mins = Math.floor(diffMs / 60000);
      var secs = Math.floor((diffMs % 60000) / 1000);
      duration = mins > 0 ? mins + 'm ' + secs + 's' : secs + 's';
    }
  }

  var items = [
    ['Session', truncId(data.session_id)],
    ['Messages', String(data.message_count || 0)],
    ['Tool Calls', String(data.tool_count || 0)],
    ['Model', model || '--'],
    ['Duration', duration || '--'],
    ['Tokens', fmtTokens(totalInput) + ' in / ' + fmtTokens(totalOutput) + ' out'],
    ['Cache Read', fmtTokens(totalCache)]
  ];

  var frag = document.createDocumentFragment();
  items.forEach(function(pair) {
    var stat = document.createElement('div');
    stat.className = 'transcript-stat';
    var lbl = document.createElement('span');
    lbl.className = 'label';
    lbl.textContent = pair[0];
    var val = document.createElement('span');
    val.className = 'value';
    val.textContent = pair[1];
    stat.appendChild(lbl);
    stat.appendChild(val);
    frag.appendChild(stat);
  });
  container.appendChild(frag);
}

function renderTranscriptMessages(messages, scrollToToolUseId) {
  var container = document.getElementById('transcript-messages');
  container.textContent = '';

  if (messages.length === 0) {
    container.textContent = 'No messages in this session.';
    return;
  }

  var scrollTarget = null;
  var frag = document.createDocumentFragment();
  messages.forEach(function(m) {
    var bubble = document.createElement('div');
    bubble.className = 'msg-bubble ' + (m.role === 'user' ? 'msg-user' : 'msg-assistant');

    // Meta row
    var meta = document.createElement('div');
    meta.className = 'msg-meta';
    var role = document.createElement('span');
    role.className = 'msg-role ' + (m.role === 'user' ? 'msg-role-user' : 'msg-role-assistant');
    role.textContent = m.role;
    meta.appendChild(role);

    if (m.model) {
      var modelBdg = document.createElement('span');
      modelBdg.className = 'model-badge';
      modelBdg.textContent = m.model;
      meta.appendChild(modelBdg);
    }

    if (m.input_tokens || m.output_tokens) {
      var tokInfo = document.createElement('span');
      tokInfo.className = 'token-info';
      var parts = [];
      if (m.input_tokens) parts.push(fmtTokens(m.input_tokens) + ' in');
      if (m.output_tokens) parts.push(fmtTokens(m.output_tokens) + ' out');
      if (m.cache_read_tokens) parts.push(fmtTokens(m.cache_read_tokens) + ' cache');
      tokInfo.textContent = parts.join(' / ');
      meta.appendChild(tokInfo);
    }

    if (m.timestamp) {
      var ts = document.createElement('span');
      ts.className = 'token-info';
      ts.textContent = fmtTime(m.timestamp);
      meta.appendChild(ts);
    }

    bubble.appendChild(meta);

    // Content
    if (m.content) {
      var content = document.createElement('div');
      content.className = 'msg-content';
      var text = m.content;
      if (text.length > 2000) text = text.substring(0, 2000) + '\n... (truncated)';
      content.textContent = text;
      bubble.appendChild(content);
    }

    // Tool calls
    if (m.tool_calls && m.tool_calls.length > 0) {
      var toolsDiv = document.createElement('div');
      toolsDiv.className = 'msg-tools';

      m.tool_calls.forEach(function(tc) {
        var chipWrapper = document.createElement('div');
        chipWrapper.style.display = 'inline-flex';
        chipWrapper.style.flexDirection = 'column';

        var chip = document.createElement('span');
        chip.className = 'tool-call-chip';
        if (tc.tool_use_id) chip.dataset.toolUseId = tc.tool_use_id;
        var name = document.createElement('span');
        name.className = 'tool-name';
        name.textContent = tc.tool_name;
        chip.appendChild(name);

        // Track scroll target
        if (scrollToToolUseId && tc.tool_use_id === scrollToToolUseId) {
          scrollTarget = bubble;
        }

        if (tc.category && tc.category !== tc.tool_name) {
          var cat = document.createElement('span');
          cat.className = 'tool-cat';
          cat.textContent = tc.category;
          chip.appendChild(cat);
        }

        var preview = document.createElement('div');
        preview.className = 'tool-input-preview';
        if (tc.input_json) {
          try {
            var parsed = JSON.parse(tc.input_json);
            preview.textContent = JSON.stringify(parsed, null, 2);
          } catch(e) {
            preview.textContent = tc.input_json;
          }
        } else {
          preview.textContent = '(no input)';
        }

        chip.addEventListener('click', function(e) {
          e.stopPropagation();
          preview.classList.toggle('open');
        });

        chipWrapper.appendChild(chip);
        chipWrapper.appendChild(preview);
        toolsDiv.appendChild(chipWrapper);
      });

      bubble.appendChild(toolsDiv);
    }

    frag.appendChild(bubble);
  });
  container.appendChild(frag);

  // Scroll to the targeted message and highlight it
  if (scrollTarget) {
    setTimeout(function() {
      scrollTarget.scrollIntoView({ behavior: 'smooth', block: 'center' });
      scrollTarget.style.outline = '2px solid var(--accent)';
      scrollTarget.style.outlineOffset = '2px';
      setTimeout(function() {
        scrollTarget.style.outline = '';
        scrollTarget.style.outlineOffset = '';
      }, 3000);
    }, 100);
  }
}
