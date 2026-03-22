defmodule HtmlgraphDashboardWeb.CostsLive do
  @moduledoc """
  Feature-level cost attribution view.

  Shows event counts, session counts, and estimated token costs per feature.
  Costs are heuristic (event_count * 3000 tokens) since Claude Code does not
  provide real token counts in hook payloads.
  """
  use HtmlgraphDashboardWeb, :live_view

  alias HtmlgraphDashboard.Repo

  @cost_query """
  SELECT
    ae.feature_id,
    COALESCE(f.title, ae.feature_id) as title,
    COALESCE(f.status, 'unknown') as status,
    COUNT(*) as event_count,
    COUNT(DISTINCT ae.session_id) as session_count,
    MIN(ae.timestamp) as first_event,
    MAX(ae.timestamp) as last_event
  FROM agent_events ae
  LEFT JOIN features f ON ae.feature_id = f.id
  WHERE ae.feature_id IS NOT NULL
  GROUP BY ae.feature_id
  ORDER BY event_count DESC
  LIMIT 50
  """

  # Heuristic: average ~3000 tokens per tool event
  @tokens_per_event 3000
  # Sonnet 4.5 input pricing: $3 per million tokens
  @cost_per_million 3.0

  @impl true
  def mount(_params, _session, socket) do
    features = load_cost_data()
    totals = compute_totals(features)

    socket =
      socket
      |> assign(:active_tab, :costs)
      |> assign(:features, features)
      |> assign(:totals, totals)

    {:ok, socket}
  end

  @impl true
  def handle_event("refresh_costs", _params, socket) do
    features = load_cost_data()
    totals = compute_totals(features)

    socket =
      socket
      |> assign(:features, features)
      |> assign(:totals, totals)

    {:noreply, socket}
  end

  defp load_cost_data do
    case Repo.query_maps(@cost_query) do
      {:ok, rows} ->
        Enum.map(rows, fn row ->
          event_count = row["event_count"] || 0
          estimated_tokens = event_count * @tokens_per_event
          estimated_cost = estimated_tokens * @cost_per_million / 1_000_000

          row
          |> Map.put("estimated_tokens", estimated_tokens)
          |> Map.put("estimated_cost", estimated_cost)
        end)

      {:error, _reason} ->
        []
    end
  end

  defp compute_totals(features) do
    %{
      event_count: Enum.reduce(features, 0, fn f, acc -> acc + (f["event_count"] || 0) end),
      session_count:
        features
        |> Enum.flat_map(fn f ->
          # session_count per feature may overlap; sum is an upper bound
          [f["session_count"] || 0]
        end)
        |> Enum.sum(),
      estimated_tokens:
        Enum.reduce(features, 0, fn f, acc -> acc + (f["estimated_tokens"] || 0) end),
      estimated_cost:
        Enum.reduce(features, 0.0, fn f, acc -> acc + (f["estimated_cost"] || 0.0) end),
      feature_count: length(features)
    }
  end

  defp status_badge_class(status) do
    case status do
      "done" -> "badge badge-status-completed"
      "in-progress" -> "badge badge-status-active"
      "blocked" -> "badge badge-error"
      "todo" -> "badge badge-count"
      _ -> "badge badge-count"
    end
  end

  defp format_cost(cost) when is_float(cost) do
    "$#{:erlang.float_to_binary(cost, decimals: 2)}"
  end

  defp format_cost(_), do: "$0.00"

  defp format_tokens(tokens) when is_integer(tokens) and tokens >= 1_000_000 do
    "#{Float.round(tokens / 1_000_000, 1)}M"
  end

  defp format_tokens(tokens) when is_integer(tokens) and tokens >= 1_000 do
    "#{Float.round(tokens / 1_000, 1)}K"
  end

  defp format_tokens(tokens) when is_integer(tokens), do: "#{tokens}"
  defp format_tokens(_), do: "0"

  defp format_number(n) when is_integer(n), do: Integer.to_string(n)
  defp format_number(_), do: "0"

  defp format_date(nil), do: "-"

  defp format_date(ts) when is_binary(ts) do
    case Regex.run(~r/(\d{4}-\d{2}-\d{2})/, ts) do
      [_, date] -> date
      _ -> ts
    end
  end

  defp truncate(nil, _), do: ""

  defp truncate(text, max_len) when is_binary(text) do
    if String.length(text) > max_len do
      String.slice(text, 0, max_len) <> "..."
    else
      text
    end
  end

  @impl true
  def render(assigns) do
    ~H"""
    <div class="header">
      <div class="header-title">
        <span class="dot"></span>
        HtmlGraph Dashboard
      </div>
      <div style="display: flex; align-items: center; gap: 16px;">
        <div class="live-indicator">
          <span class="live-dot"></span>
          Live
        </div>
        <div class="header-meta">
          <%= @totals.feature_count %> features tracked
        </div>
      </div>
    </div>

    <nav class="dashboard-nav">
      <a href="/" class="nav-tab">Activity Feed</a>
      <a href="/graph" class="nav-tab">Graph</a>
      <a href="/kanban" class="nav-tab">Kanban</a>
      <a href="/costs" class="nav-tab active">Costs</a>
    </nav>

    <div class="graph-stats-bar">
      <div class="stat-card">
        <span class="stat-label">Features</span>
        <span class="stat-value"><%= format_number(@totals.feature_count) %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Total Events</span>
        <span class="stat-value"><%= format_number(@totals.event_count) %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Sessions</span>
        <span class="stat-value"><%= format_number(@totals.session_count) %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Est. Tokens</span>
        <span class="stat-value"><%= format_tokens(@totals.estimated_tokens) %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Est. Cost</span>
        <span class="stat-value"><%= format_cost(@totals.estimated_cost) %></span>
      </div>
      <div class="stat-card" style="margin-left: auto;">
        <button phx-click="refresh_costs" class="graph-refresh-btn">
          Refresh
        </button>
      </div>
    </div>

    <div style="padding: 4px 24px 8px; font-size: 11px; color: var(--text-secondary, #94a3b8);">
      Costs are estimated based on event counts (~3K tokens/event at Sonnet pricing).
      Actual token usage is not yet captured by Claude Code hooks.
    </div>

    <div class="feed-container">
      <%= if @features == [] do %>
        <div class="empty-state">
          <h2>No cost data yet</h2>
          <p>Feature-level cost attribution will appear here once events are tracked with feature IDs.</p>
        </div>
      <% else %>
        <table class="costs-table">
          <thead>
            <tr>
              <th class="costs-th" style="text-align: left;">Feature</th>
              <th class="costs-th">Status</th>
              <th class="costs-th">Events</th>
              <th class="costs-th">Sessions</th>
              <th class="costs-th">Est. Tokens</th>
              <th class="costs-th">Est. Cost</th>
              <th class="costs-th">Time Span</th>
            </tr>
          </thead>
          <tbody>
            <%= for feature <- @features do %>
              <tr class="costs-row">
                <td class="costs-td" style="text-align: left;">
                  <div class="costs-feature-title">
                    <span class="badge badge-session" style="font-size: 9px; margin-right: 6px;">
                      <%= truncate(feature["feature_id"], 14) %>
                    </span>
                    <span><%= truncate(feature["title"], 50) %></span>
                  </div>
                </td>
                <td class="costs-td">
                  <span class={status_badge_class(feature["status"])}>
                    <%= feature["status"] || "unknown" %>
                  </span>
                </td>
                <td class="costs-td costs-number">
                  <%= format_number(feature["event_count"]) %>
                </td>
                <td class="costs-td costs-number">
                  <%= format_number(feature["session_count"]) %>
                </td>
                <td class="costs-td costs-number">
                  <%= format_tokens(feature["estimated_tokens"]) %>
                </td>
                <td class="costs-td costs-number costs-cost">
                  <%= format_cost(feature["estimated_cost"]) %>
                </td>
                <td class="costs-td" style="font-size: 11px; color: var(--text-secondary, #94a3b8);">
                  <%= format_date(feature["first_event"]) %> &rarr; <%= format_date(feature["last_event"]) %>
                </td>
              </tr>
            <% end %>
          </tbody>
          <tfoot>
            <tr class="costs-total-row">
              <td class="costs-td" style="text-align: left; font-weight: 600;">
                Total (<%= @totals.feature_count %> features)
              </td>
              <td class="costs-td"></td>
              <td class="costs-td costs-number" style="font-weight: 600;">
                <%= format_number(@totals.event_count) %>
              </td>
              <td class="costs-td costs-number" style="font-weight: 600;">
                <%= format_number(@totals.session_count) %>
              </td>
              <td class="costs-td costs-number" style="font-weight: 600;">
                <%= format_tokens(@totals.estimated_tokens) %>
              </td>
              <td class="costs-td costs-number costs-cost" style="font-weight: 600;">
                <%= format_cost(@totals.estimated_cost) %>
              </td>
              <td class="costs-td"></td>
            </tr>
          </tfoot>
        </table>
      <% end %>
    </div>
    """
  end
end
