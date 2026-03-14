defmodule HtmlgraphDashboard.Activity do
  @moduledoc """
  Queries and structures the activity feed data from the HtmlGraph database.

  Builds a multi-level nested tree:
    Session -> UserQuery (conversation turn) -> Tool events -> Subagent events
  """

  alias HtmlgraphDashboard.Repo

  @max_depth 4

  @doc """
  Fetch recent conversation turns with nested children, grouped by session.
  Returns a list of session groups, each containing conversation turns.
  """
  def list_activity_feed(opts \\ []) do
    limit = Keyword.get(opts, :limit, 50)
    session_id = Keyword.get(opts, :session_id, nil)

    # Fetch UserQuery events (conversation turns) — these are the top-level entries
    user_queries = fetch_user_queries(limit, session_id)

    # For each UserQuery, recursively fetch children + adopt orphans
    turns =
      Enum.map(user_queries, fn uq ->
        children = fetch_children_with_subagents(uq["event_id"], uq["session_id"], 0)

        # Adopt orphan events that belong to this UserQuery's time window
        orphans = fetch_orphan_events(uq, user_queries)
        all_children = merge_children_by_timestamp(children, orphans)

        work_item = if uq["feature_id"], do: fetch_feature(uq["feature_id"]), else: nil

        displayed_children =
          all_children
          |> Enum.map(&sanitize_tree/1)
          |> Enum.filter(&has_meaningful_content/1)

        stats = compute_stats(displayed_children)

        %{
          user_query: sanitize_event(uq),
          children: displayed_children,
          stats: stats,
          work_item: work_item
        }
      end)

    # Group by session
    turns
    |> Enum.group_by(fn t -> t.user_query["session_id"] end)
    |> Enum.map(fn {sid, session_turns} ->
      session = fetch_session(sid)

      %{
        session_id: sid,
        session: session,
        turns: session_turns
      }
    end)
    |> Enum.sort_by(
      fn group ->
        case group.turns do
          [first | _] -> first.user_query["timestamp"]
          [] -> ""
        end
      end,
      :desc
    )
  end

  @doc """
  Fetch a single event by ID with its full subtree.
  """
  def get_event_tree(event_id) do
    sql = """
    SELECT event_id, tool_name, event_type, timestamp, input_summary,
           output_summary, session_id, agent_id, parent_event_id,
           subagent_type, model, status, cost_tokens,
           execution_duration_seconds, feature_id, context
    FROM agent_events
    WHERE event_id = ?
    """

    case Repo.query_maps(sql, [event_id]) do
      {:ok, [event]} ->
        children = fetch_children_with_subagents(event_id, event["session_id"], 0)
        {:ok, Map.put(event, "children", children)}

      {:ok, []} ->
        {:error, :not_found}

      {:error, reason} ->
        {:error, reason}
    end
  end

  # --- Summary Sanitization ---

  @doc """
  Sanitize a summary string by stripping noise:
  - XML tags (task-notification, system-reminder) and their content
  - Raw JSON objects (context/metadata dumps)
  - Truncate to 120 chars
  """
  def sanitize_summary(nil), do: ""
  def sanitize_summary(""), do: ""

  def sanitize_summary(text) when is_binary(text) do
    trimmed = String.trim(text)

    # Early exit: if string starts with {", it's a raw JSON metadata dump — discard entirely
    if String.starts_with?(trimmed, "{\"") do
      ""
    else
      trimmed
      |> strip_xml_tags()
      |> strip_json_dumps()
      |> String.trim()
      |> truncate_text(120)
    end
  end

  defp strip_xml_tags(text) do
    text
    # Strip matched pairs first (greedy within each pair)
    |> String.replace(~r/<task-notification>[\s\S]*?<\/task-notification>/i, "")
    |> String.replace(~r/<system-reminder>[\s\S]*?<\/system-reminder>/i, "")
    |> String.replace(~r/<[a-zA-Z_-]+>[\s\S]*?<\/[a-zA-Z_-]+>/i, "")
    # Strip orphaned opening/closing tags (no matching pair in string)
    |> String.replace(~r/<\/?[a-zA-Z_-]+>/i, "")
  end

  defp strip_json_dumps(text) do
    # If the entire string looks like a JSON object, replace it
    trimmed = String.trim(text)

    if String.starts_with?(trimmed, "{") and String.ends_with?(trimmed, "}") do
      case Jason.decode(trimmed) do
        {:ok, map} when is_map(map) ->
          # Extract useful fields if present, otherwise discard
          cond do
            Map.has_key?(map, "subagent_type") ->
              prompt = Map.get(map, "prompt", "")
              type = Map.get(map, "subagent_type", "")

              if prompt != "" do
                "Task (#{type}): #{prompt}"
              else
                "Task delegation: #{type}"
              end

            Map.has_key?(map, "session_id") ->
              # Pure context/metadata dump — discard
              ""

            true ->
              trimmed
          end

        _ ->
          trimmed
      end
    else
      text
    end
  end

  defp truncate_text(text, max_len) do
    if String.length(text) > max_len do
      String.slice(text, 0, max_len) <> "..."
    else
      text
    end
  end

  defp sanitize_event(event) do
    event
    |> Map.update("input_summary", "", &sanitize_summary/1)
    |> Map.update("output_summary", "", &sanitize_summary/1)
  end

  defp sanitize_tree(event) do
    event
    |> sanitize_event()
    |> Map.update("children", [], fn children ->
      children || []
      |> Enum.map(&sanitize_tree/1)
      |> Enum.filter(&has_meaningful_content/1)
    end)
  end

  # Check if an event has meaningful content (not just empty summaries).
  # Filters out PreToolUse events that have no useful summary.
  defp has_meaningful_content(event) do
    input_summary = event["input_summary"] || ""
    output_summary = event["output_summary"] || ""

    # Filter out noise tool names that never carry useful content
    noise_tool = event["tool_name"] in ["Stop", "SessionResume", "InstructionsLoaded"]

    if noise_tool do
      false
    else
      # Keep events that have at least one meaningful summary
      (String.trim(input_summary) != "" or String.trim(output_summary) != "") and
        not is_empty_pretooluse(event)
    end
  end

  defp is_empty_pretooluse(event) do
    # Filter out PreToolUse events with no real content
    event["event_type"] == "start" and
      (event["input_summary"] == nil or event["input_summary"] == "" or event["input_summary"] == "{}") and
      (event["output_summary"] == nil or event["output_summary"] == "")
  end

  # --- Private: Data fetching ---

  defp fetch_user_queries(limit, nil) do
    sql = """
    SELECT event_id, tool_name, event_type, timestamp, input_summary,
           output_summary, session_id, agent_id, parent_event_id,
           subagent_type, model, status, cost_tokens,
           execution_duration_seconds, feature_id, context
    FROM agent_events
    WHERE tool_name = 'UserQuery'
    ORDER BY timestamp DESC
    LIMIT ?
    """

    case Repo.query_maps(sql, [limit]) do
      {:ok, rows} -> rows
      {:error, _} -> []
    end
  end

  defp fetch_user_queries(limit, session_id) do
    sql = """
    SELECT event_id, tool_name, event_type, timestamp, input_summary,
           output_summary, session_id, agent_id, parent_event_id,
           subagent_type, model, status, cost_tokens,
           execution_duration_seconds, feature_id, context
    FROM agent_events
    WHERE tool_name = 'UserQuery' AND session_id = ?
    ORDER BY timestamp DESC
    LIMIT ?
    """

    case Repo.query_maps(sql, [session_id, limit]) do
      {:ok, rows} -> rows
      {:error, _} -> []
    end
  end

  defp fetch_children_with_subagents(_parent_id, _session_id, depth) when depth >= @max_depth,
    do: []

  defp fetch_children_with_subagents(parent_id, session_id, depth) do
    # Fetch direct children by parent_event_id
    sql = """
    SELECT event_id, tool_name, event_type, timestamp, input_summary,
           output_summary, session_id, agent_id, parent_event_id,
           subagent_type, model, status, cost_tokens,
           execution_duration_seconds, feature_id, context
    FROM agent_events
    WHERE parent_event_id = ?
      AND NOT (tool_name = 'Agent' AND event_type != 'task_delegation')
    ORDER BY timestamp DESC
    """

    rows =
      case Repo.query_maps(sql, [parent_id]) do
        {:ok, rows} -> rows
        {:error, _} -> []
      end

    Enum.map(rows, fn row ->
      grandchildren =
        if row["event_type"] == "task_delegation" do
          # For task delegations, also pull subagent session events
          subagent_children =
            fetch_subagent_events(row["event_id"], session_id, row["subagent_type"], depth + 1)

          direct = fetch_children_with_subagents(row["event_id"], session_id, depth + 1)
          merge_children_by_timestamp(direct, subagent_children)
        else
          fetch_children_with_subagents(row["event_id"], session_id, depth + 1)
        end

      row
      |> Map.put("children", grandchildren)
      |> Map.put("depth", depth)
      |> Map.put("descendant_count", count_descendants(grandchildren))
    end)
  end

  defp fetch_subagent_events(_task_event_id, _parent_session_id, _subagent_type, depth)
       when depth >= @max_depth,
       do: []

  defp fetch_subagent_events(_task_event_id, parent_session_id, subagent_type, depth) do
    # Subagent sessions follow the pattern: {parent_session_id}-{agent_name}
    # Try multiple patterns to find subagent events
    patterns = build_subagent_session_patterns(parent_session_id, subagent_type)

    Enum.flat_map(patterns, fn pattern ->
      sql = """
      SELECT event_id, tool_name, event_type, timestamp, input_summary,
             output_summary, session_id, agent_id, parent_event_id,
             subagent_type, model, status, cost_tokens,
             execution_duration_seconds, feature_id, context
      FROM agent_events
      WHERE session_id LIKE ?
        AND tool_name != 'UserQuery'
        AND NOT (tool_name = 'Agent' AND event_type != 'task_delegation')
      ORDER BY timestamp DESC
      """

      case Repo.query_maps(sql, [pattern]) do
        {:ok, rows} ->
          # Only include events that don't already have a parent pointing elsewhere
          # (they may already be fetched via parent_event_id)
          rows
          |> Enum.reject(fn r -> r["parent_event_id"] != nil end)
          |> Enum.map(fn r ->
            r
            |> Map.put("depth", depth)
            |> Map.put("children", [])
            |> Map.put("descendant_count", 0)
          end)

        {:error, _} ->
          []
      end
    end)
  end

  defp build_subagent_session_patterns(parent_session_id, nil) do
    ["#{parent_session_id}-%"]
  end

  defp build_subagent_session_patterns(parent_session_id, subagent_type) do
    # Try exact match first, then wildcard
    [
      "#{parent_session_id}-#{subagent_type}%"
    ]
  end

  # --- Orphan Adoption ---

  defp fetch_orphan_events(user_query, all_user_queries) do
    session_id = user_query["session_id"]
    uq_timestamp = user_query["timestamp"]
    uq_event_id = user_query["event_id"]

    # Find the next UserQuery in the same session (by timestamp)
    next_uq =
      all_user_queries
      |> Enum.filter(fn uq ->
        uq["session_id"] == session_id and
          uq["timestamp"] > uq_timestamp and
          uq["event_id"] != uq_event_id
      end)
      |> Enum.sort_by(fn uq -> uq["timestamp"] end)
      |> List.first()

    # Query for orphan events in the time window
    {sql, params} =
      if next_uq do
        {"""
         SELECT event_id, tool_name, event_type, timestamp, input_summary,
                output_summary, session_id, agent_id, parent_event_id,
                subagent_type, model, status, cost_tokens,
                execution_duration_seconds, feature_id, context
         FROM agent_events
         WHERE session_id = ?
           AND parent_event_id IS NULL
           AND tool_name != 'UserQuery'
           AND NOT (tool_name = 'Agent' AND event_type != 'task_delegation')
           AND timestamp >= ?
           AND timestamp < ?
         ORDER BY timestamp DESC
         """, [session_id, uq_timestamp, next_uq["timestamp"]]}
      else
        {"""
         SELECT event_id, tool_name, event_type, timestamp, input_summary,
                output_summary, session_id, agent_id, parent_event_id,
                subagent_type, model, status, cost_tokens,
                execution_duration_seconds, feature_id, context
         FROM agent_events
         WHERE session_id = ?
           AND parent_event_id IS NULL
           AND tool_name != 'UserQuery'
           AND NOT (tool_name = 'Agent' AND event_type != 'task_delegation')
           AND timestamp >= ?
         ORDER BY timestamp DESC
         """, [session_id, uq_timestamp]}
      end

    case Repo.query_maps(sql, params) do
      {:ok, rows} ->
        rows
        |> Enum.map(fn row ->
          row
          |> Map.put("depth", 0)
          |> Map.put("children", [])
          |> Map.put("descendant_count", 0)
        end)
        |> Enum.filter(&has_meaningful_content/1)

      {:error, _} ->
        []
    end
  end

  # --- Helpers ---

  defp merge_children_by_timestamp(list_a, list_b) do
    # Deduplicate by event_id, then sort by timestamp descending
    (list_a ++ list_b)
    |> Enum.uniq_by(fn e -> e["event_id"] end)
    |> Enum.sort_by(fn e -> e["timestamp"] end, :desc)
  end

  defp count_descendants(children) do
    Enum.reduce(children, 0, fn child, acc ->
      acc + 1 + (child["descendant_count"] || count_descendants(child["children"] || []))
    end)
  end

  defp compute_stats(children) do
    flat = flatten_children(children)

    %{
      tool_count: length(flat),
      total_duration:
        flat
        |> Enum.map(fn c -> c["execution_duration_seconds"] || 0 end)
        |> Enum.sum()
        |> to_float()
        |> Float.round(2),
      success_count:
        Enum.count(flat, fn c -> c["status"] in ["recorded", "success", "completed"] end),
      error_count: Enum.count(flat, fn c -> c["event_type"] == "error" end),
      models:
        flat |> Enum.map(fn c -> c["model"] end) |> Enum.reject(&is_nil/1) |> Enum.uniq(),
      total_tokens:
        flat
        |> Enum.map(fn c -> c["cost_tokens"] || 0 end)
        |> Enum.sum()
    }
  end

  defp to_float(value) when is_float(value), do: value
  defp to_float(value) when is_integer(value), do: value * 1.0

  defp flatten_children(children) do
    Enum.flat_map(children, fn child ->
      [child | flatten_children(child["children"] || [])]
    end)
  end

  defp fetch_session(nil), do: nil

  defp fetch_session(session_id) do
    sql = """
    SELECT session_id, agent_assigned, status, created_at, completed_at,
           total_events, total_tokens_used, is_subagent, last_user_query,
           model
    FROM sessions
    WHERE session_id = ?
    """

    case Repo.query_maps(sql, [session_id]) do
      {:ok, [session]} -> derive_session_status(session)
      _ -> nil
    end
  end

  defp derive_session_status(session) do
    cond do
      # If completed_at is set, it's completed
      session["completed_at"] != nil ->
        Map.put(session, "status", "completed")

      # If status is already explicitly set to something other than active, keep it
      session["status"] not in [nil, "active"] ->
        session

      # Check if the session's last event is older than 30 minutes
      true ->
        case last_event_timestamp(session["session_id"]) do
          nil ->
            session

          ts_string ->
            case NaiveDateTime.from_iso8601(ts_string) do
              {:ok, last_event_ts} ->
                cutoff = NaiveDateTime.add(NaiveDateTime.utc_now(), -30, :minute)

                if NaiveDateTime.compare(last_event_ts, cutoff) == :lt do
                  Map.put(session, "status", "idle")
                else
                  session
                end

              _ ->
                session
            end
        end
    end
  end

  defp last_event_timestamp(nil), do: nil

  defp last_event_timestamp(session_id) do
    sql = """
    SELECT MAX(timestamp) AS last_ts
    FROM agent_events
    WHERE session_id = ?
    """

    case Repo.query_maps(sql, [session_id]) do
      {:ok, [%{"last_ts" => ts}]} -> ts
      _ -> nil
    end
  end

  defp fetch_feature(nil), do: nil

  defp fetch_feature(feature_id) do
    sql = """
    SELECT id, type, title, status, priority
    FROM features
    WHERE id = ?
    """

    case Repo.query_maps(sql, [feature_id]) do
      {:ok, [feature]} -> feature
      _ -> nil
    end
  end
end
