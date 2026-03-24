defmodule HtmlgraphDashboardWeb.ProjectsLive do
  @moduledoc """
  Multi-project Command Center overview.

  Displays all discovered HtmlGraph projects as cards with live stats,
  sparkline activity graphs, and navigation links. Uses the Command Center
  design system (sharp edges, monospace, scanline textures, project colors).

  Stats per project: total events, feature count, session count, estimated cost.
  Sparkline: 7-day activity (events per day).
  """
  use HtmlgraphDashboardWeb, :live_view

  import HtmlgraphDashboardWeb.ProjectComponents

  alias HtmlgraphDashboard.ProjectRegistry
  alias HtmlgraphDashboard.Repo

  @impl true
  def mount(_params, _session, socket) do
    projects = ProjectRegistry.list_projects()
    project_data = load_all_project_data(projects)
    aggregates = compute_aggregates(project_data)

    socket =
      socket
      |> assign(:active_tab, :projects)
      |> assign(:projects, projects)
      |> assign(:project_data, project_data)
      |> assign(:aggregates, aggregates)
      |> assign(:picker_open, false)

    {:ok, socket}
  end

  @impl true
  def handle_event("refresh_projects", _params, socket) do
    ProjectRegistry.refresh()
    send(self(), :reload_projects)
    {:noreply, socket}
  end

  def handle_event("toggle_project_picker", _params, socket) do
    {:noreply, assign(socket, :picker_open, !socket.assigns.picker_open)}
  end

  def handle_event("close_project_picker", _params, socket) do
    {:noreply, assign(socket, :picker_open, false)}
  end

  def handle_event("select_project", %{"project_id" => project_id}, socket) do
    {:noreply,
     socket
     |> assign(:picker_open, false)
     |> redirect(to: "/?project=#{project_id}")}
  end

  @impl true
  def handle_info(:reload_projects, socket) do
    projects = ProjectRegistry.list_projects()
    project_data = load_all_project_data(projects)
    aggregates = compute_aggregates(project_data)

    socket =
      socket
      |> assign(:projects, projects)
      |> assign(:project_data, project_data)
      |> assign(:aggregates, aggregates)

    {:noreply, socket}
  end

  # ------------------------------------------------------------------
  # Render
  # ------------------------------------------------------------------

  @impl true
  def render(assigns) do
    ~H"""
    <.identity_stripe project={nil} />

    <div class="header" style="margin-top: 3px;">
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
          <%= length(@projects) %> project(s)
        </div>
      </div>
    </div>

    <nav class="dashboard-nav">
      <a href="/" class="nav-tab">Activity Feed</a>
      <a href="/graph" class="nav-tab">Graph</a>
      <a href="/kanban" class="nav-tab">Kanban</a>
      <a href="/costs" class="nav-tab">Costs</a>
      <a href="/projects" class="nav-tab active">Projects</a>
    </nav>

    <div class="projects-header">
      <div class="projects-header-left">
        <div class="projects-title">Projects</div>
        <div class="projects-subtitle">
          <%= length(@projects) %> discovered project(s) &middot;
          <%= @aggregates.total_events %> total events
        </div>
      </div>
      <div class="projects-header-right">
        <div class="projects-aggregate-stats">
          <.stat_badge label="Events" value={format_number(@aggregates.total_events)} />
          <.stat_badge label="Features" value={"#{@aggregates.total_features}"} />
          <.stat_badge label="Sessions" value={"#{@aggregates.total_sessions}"} />
          <.stat_badge label="Est. Cost" value={@aggregates.total_cost} />
        </div>
        <button phx-click="refresh_projects" class="projects-refresh-btn">
          Refresh
        </button>
      </div>
    </div>

    <%= if @projects == [] do %>
      <div class="projects-empty">
        <h2>No projects discovered</h2>
        <p>Set <code>HTMLGRAPH_WORKSPACE</code> to your workspace root directory,</p>
        <p>or projects will be auto-discovered from the parent directory.</p>
        <p>Each project must have a <code>.htmlgraph/htmlgraph.db</code> file.</p>
      </div>
    <% else %>
      <div class="projects-grid">
        <%= for project <- @projects do %>
          <% data = Map.get(@project_data, project.id, default_project_data()) %>
          <.project_card
            project={project}
            stats={data.stats}
            active_sessions={data.active_sessions}
            sparkline_data={data.sparkline}
          />
        <% end %>
      </div>
    <% end %>
    """
  end

  # ------------------------------------------------------------------
  # Data Loading
  # ------------------------------------------------------------------

  defp load_all_project_data(projects) do
    Map.new(projects, fn project ->
      {project.id, load_project_data(project.id)}
    end)
  end

  defp load_project_data(project_id) do
    stats = load_project_stats(project_id)
    active = load_active_sessions(project_id)
    sparkline = load_sparkline(project_id)

    %{
      stats: stats,
      active_sessions: active,
      sparkline: sparkline
    }
  end

  defp load_project_stats(project_id) do
    events = query_scalar("SELECT COUNT(*) FROM agent_events", project_id)
    features = query_scalar("SELECT COUNT(*) FROM features WHERE type = 'feature'", project_id)
    sessions = query_scalar("SELECT COUNT(DISTINCT session_id) FROM agent_events", project_id)
    cost = estimate_cost(events)

    %{events: events, features: features, sessions: sessions, cost: cost}
  end

  defp load_active_sessions(project_id) do
    query_scalar(
      "SELECT COUNT(*) FROM sessions WHERE status = 'active'",
      project_id
    )
  end

  defp load_sparkline(project_id) do
    sql = """
    SELECT date(timestamp) as day, count(*) as events
    FROM agent_events
    WHERE timestamp >= datetime('now', '-7 days')
    GROUP BY date(timestamp)
    ORDER BY day
    """

    case Repo.query_maps(sql, [], project_id) do
      {:ok, rows} ->
        fill_sparkline_gaps(rows)

      _ ->
        []
    end
  end

  # ------------------------------------------------------------------
  # Aggregates
  # ------------------------------------------------------------------

  defp compute_aggregates(project_data) do
    values = Map.values(project_data)

    total_events = values |> Enum.map(& &1.stats.events) |> Enum.sum()
    total_features = values |> Enum.map(& &1.stats.features) |> Enum.sum()
    total_sessions = values |> Enum.map(& &1.stats.sessions) |> Enum.sum()

    %{
      total_events: total_events,
      total_features: total_features,
      total_sessions: total_sessions,
      total_cost: estimate_cost(total_events)
    }
  end

  # ------------------------------------------------------------------
  # Helpers
  # ------------------------------------------------------------------

  defp query_scalar(sql, project_id) do
    case Repo.query(sql, [], project_id) do
      {:ok, [[val] | _]} when is_integer(val) -> val
      _ -> 0
    end
  end

  defp estimate_cost(event_count) do
    # Rough estimate: avg ~3000 tokens per event, $3.00 per 1M tokens
    dollars = event_count * 3000 * 3.0 / 1_000_000
    format_cost(dollars)
  end

  defp format_cost(dollars) when dollars < 0.01, do: "$0.00"
  defp format_cost(dollars) when dollars < 1.0, do: "$#{Float.round(dollars, 2)}"
  defp format_cost(dollars) when dollars < 100.0, do: "$#{Float.round(dollars, 1)}"
  defp format_cost(dollars), do: "$#{round(dollars)}"

  defp format_number(n) when is_integer(n) and n >= 1_000_000 do
    "#{Float.round(n / 1_000_000, 1)}M"
  end

  defp format_number(n) when is_integer(n) and n >= 1000 do
    "#{Float.round(n / 1000, 1)}k"
  end

  defp format_number(n), do: "#{n}"

  defp default_project_data do
    %{
      stats: %{events: 0, features: 0, sessions: 0, cost: "$0.00"},
      active_sessions: 0,
      sparkline: []
    }
  end

  defp fill_sparkline_gaps(rows) do
    today = Date.utc_today()
    seven_days_ago = Date.add(today, -6)

    # Build a map of day_string => event_count from query results
    day_map =
      Map.new(rows, fn row ->
        {row["day"], row["events"] || 0}
      end)

    # Generate all 7 days, filling gaps with 0
    Date.range(seven_days_ago, today)
    |> Enum.map(fn date ->
      key = Date.to_iso8601(date)
      (day_map[key] || 0) * 1.0
    end)
  end
end
