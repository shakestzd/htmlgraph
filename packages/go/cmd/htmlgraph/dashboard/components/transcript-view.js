/* ── Transcript detail view ─────────────────────────────────── */

// scrollHint: { toolUseId, toolName, timestamp } or undefined
function openTranscript(sessionId, scrollHint) {
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
    .then(function(data) { renderTranscript(data, scrollHint); })
    .catch(function(err) {
      document.getElementById('transcript-messages').textContent = 'Failed to load transcript: ' + err.message;
    });
}

function closeTranscript() {
  document.getElementById('transcript-detail').className = 'transcript-detail';
  document.getElementById('sessions-list-view').style.display = '';
}

document.getElementById('transcript-back').addEventListener('click', closeTranscript);

function renderTranscript(data, scrollHint) {
  renderTranscriptStats(data);
  renderTranscriptMessages(data.messages || [], scrollHint);
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

function renderTranscriptMessages(messages, scrollHint) {
  var container = document.getElementById('transcript-messages');
  container.textContent = '';

  if (messages.length === 0) {
    container.textContent = 'No messages in this session.';
    return;
  }

  var hint = scrollHint || {};
  var scrollTarget = null;
  var bestScore = -1;
  var targetTs = hint.timestamp ? new Date(hint.timestamp).getTime() : 0;
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

        // Exact tool_use_id match — highest priority
        if (hint.toolUseId && tc.tool_use_id === hint.toolUseId) {
          scrollTarget = bubble;
          bestScore = 1000;
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

    // Scored fallback matching when no exact tool_use_id hit
    if (bestScore < 1000 && targetTs && m.timestamp) {
      var msgTs = new Date(m.timestamp).getTime();
      var timeDiff = Math.abs(msgTs - targetTs);
      // Score: tool_name match + time proximity (closer = higher)
      var score = 0;
      if (timeDiff < 30000) { // within 30 seconds
        score = 100 - (timeDiff / 300); // 0-100 based on proximity
        // Bonus for matching tool_name in this message's tool_calls
        if (hint.toolName && m.tool_calls) {
          for (var i = 0; i < m.tool_calls.length; i++) {
            if (m.tool_calls[i].tool_name === hint.toolName) {
              score += 200; // strong signal
              break;
            }
          }
        }
        // Bonus for user messages matching UserQuery events
        if (!hint.toolName && m.role === 'user') {
          score += 50;
        }
      }
      if (score > bestScore) {
        bestScore = score;
        scrollTarget = bubble;
      }
    }

    frag.appendChild(bubble);
  });
  container.appendChild(frag);

  // Scroll to the targeted message and highlight it
  if (scrollTarget) {
    setTimeout(function() {
      scrollTarget.scrollIntoView({ behavior: 'smooth', block: 'center' });
      scrollTarget.classList.add('msg-highlight');
      setTimeout(function() { scrollTarget.classList.remove('msg-highlight'); }, 4000);
    }, 150);
  }
}
