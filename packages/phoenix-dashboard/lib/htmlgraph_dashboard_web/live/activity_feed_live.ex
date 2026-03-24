defmodule HtmlgraphDashboardWeb.ActivityFeedLive do
  @moduledoc """
  Live activity feed with multi-level nested events, badges, and real-time updates.

  Architecture:
  - Polls SQLite database via EventPoller GenServer
  - Receives new events via PubSub broadcast
  - Maintains expand/collapse state per conversation turn
  - Multi-level nesting: Session > UserQuery > Tool Events > Subagent Events
  """
  use HtmlgraphDashboardWeb, :live_view

  alias HtmlgraphDashboard.Activity
  alias HtmlgraphDashboard.EventPoller
  alias HtmlgraphDashboard.ProjectRegistry
  alias HtmlgraphDashboard.PythonSDK
  alias HtmlgraphDashboard.Repo

  @default_activity_stats %{
    "sessions" => 0,
    "events" => 0,
    "features" => 0,
    "bugs" => 0,
    "active" => 0,
    "tools" => 0
  }

  @impl true
  def mount(params, _session, socket) do
    if connected?(socket) do
      EventPoller.subscribe()
    end

    session_id = params["session_id"]

    projects = ProjectRegistry.list_projects()
    selected_project_id = params["project"] || (List.first(projects, %{}) |> Map.get(:id))
    selected_project = Enum.find(projects, List.first(projects), &(&1.id == selected_project_id))

    activity_stats = load_activity_stats(selected_project && selected_project.id)

    socket =
      socket
      |> assign(:session_filter, session_id)
      |> assign(:expanded, MapSet.new())
      |> assign(:reload_timer, nil)
      |> assign(:activity_stats, activity_stats)
      |> assign(:selected_work_item, nil)
      |> assign(:projects, projects)
      |> assign(:selected_project, selected_project)
      |> load_feed()

    {:ok, socket}
  end

  @impl true
  def handle_params(params, _uri, socket) do
    session_id = params["session_id"]

    # Re-resolve selected_project from URL param on every navigation
    socket =
      case params["project"] do
        nil ->
          socket

        project_id ->
          project = Enum.find(socket.assigns.projects, socket.assigns.selected_project, &(&1.id == project_id))
          pid = project && project.id

          socket
          |> assign(:selected_project, project)
          |> assign(:activity_stats, load_activity_stats(pid))
      end

    socket =
      socket
      |> assign(:session_filter, session_id)
      |> load_feed()

    {:noreply, socket}
  end

  @impl true
  def handle_event("toggle", %{"event-id" => event_id}, socket) do
    expanded = socket.assigns.expanded

    expanded =
      if MapSet.member?(expanded, event_id) do
        MapSet.delete(expanded, event_id)
      else
        MapSet.put(expanded, event_id)
      end

    {:noreply, assign(socket, :expanded, expanded)}
  end

  def handle_event("toggle_session", %{"session-id" => session_id}, socket) do
    expanded = socket.assigns.expanded
    key = "session:#{session_id}"

    expanded =
      if MapSet.member?(expanded, key) do
        MapSet.delete(expanded, key)
      else
        MapSet.put(expanded, key)
      end

    {:noreply, assign(socket, :expanded, expanded)}
  end

  def handle_event("select_work_item", %{"id" => work_item_id}, socket) do
    project = socket.assigns[:selected_project]
    opts = if project, do: project_graph_opts(project), else: %{}

    work_item =
      try do
        case PythonSDK.get_work_item(work_item_id, opts) do
          {:ok, item} -> item
          _ -> nil
        end
      rescue
        _ -> nil
      catch
        :exit, _ -> nil
      end

    {:noreply, assign(socket, :selected_work_item, work_item)}
  end

  def handle_event("close_work_item_detail", _params, socket) do
    {:noreply, assign(socket, :selected_work_item, nil)}
  end

  def handle_event("select_project", %{"project_id" => project_id}, socket) do
    project = Enum.find(socket.assigns.projects, &(&1.id == project_id))
    project_id_val = project && project.id

    socket =
      socket
      |> assign(:selected_project, project)
      |> assign(:activity_stats, load_activity_stats(project_id_val))
      |> load_feed()

    {:noreply, push_patch(socket, to: "/?project=#{project_id_val}")}
  end

  @impl true
  def handle_info({:new_event, _event}, socket) do
    # Debounce: schedule a single reload 500ms from now
    # Cancel any existing pending reload to avoid redundant work
    if socket.assigns[:reload_timer] do
      Process.cancel_timer(socket.assigns.reload_timer)
    end

    timer = Process.send_after(self(), :do_reload, 500)
    {:noreply, assign(socket, :reload_timer, timer)}
  end

  def handle_info(:do_reload, socket) do
    project_id = socket.assigns[:selected_project] && socket.assigns.selected_project.id

    socket =
      socket
      |> assign(:reload_timer, nil)
      |> assign(:activity_stats, load_activity_stats(project_id))
      |> load_feed()

    {:noreply, socket}
  end

  defp load_activity_stats(project_id) do
    queries = [
      {"sessions", "SELECT COUNT(DISTINCT session_id) as v FROM agent_events"},
      {"events", "SELECT COUNT(*) as v FROM agent_events"},
      {"features", "SELECT COUNT(*) as v FROM features WHERE type = 'feature'"},
      {"bugs", "SELECT COUNT(*) as v FROM features WHERE type = 'bug'"},
      {"active", "SELECT COUNT(*) as v FROM features WHERE status = 'in-progress'"},
      {"tools",
       "SELECT COUNT(*) as v FROM agent_events WHERE event_type NOT IN ('UserQuery', 'session_start', 'session_end')"}
    ]

    Enum.reduce(queries, @default_activity_stats, fn {key, sql}, acc ->
      case Repo.query_maps(sql, [], project_id) do
        {:ok, [%{"v" => val} | _]} -> Map.put(acc, key, val || 0)
        _ -> acc
      end
    end)
  end

  defp load_feed(socket) do
    project_id = socket.assigns[:selected_project] && socket.assigns.selected_project.id

    opts =
      case socket.assigns[:session_filter] do
        nil -> [limit: 50, project_id: project_id]
        sid -> [limit: 50, session_id: sid, project_id: project_id]
      end

    feed = Activity.list_activity_feed(opts)
    total_events = feed |> Enum.map(fn g -> length(g.turns) end) |> Enum.sum()

    # Collect all unique feature_ids across turns and their children
    feature_ids =
      feed
      |> Enum.flat_map(fn group ->
        group.turns
        |> Enum.flat_map(fn turn ->
          turn_id = turn.user_query["feature_id"]
          child_ids = collect_feature_ids(turn.children)
          [turn_id | child_ids]
        end)
      end)
      |> Enum.reject(&is_nil/1)
      |> Enum.uniq()

    project = socket.assigns[:selected_project]
    sdk_opts = if project, do: project_graph_opts(project), else: %{}

    work_item_titles =
      if feature_ids == [] do
        %{}
      else
        try do
          case PythonSDK.get_work_item_titles(feature_ids, sdk_opts) do
            {:ok, titles} -> titles
            {:error, _} -> %{}
          end
        rescue
          _ -> %{}
        catch
          :exit, _ -> %{}
        end
      end

    socket
    |> assign(:feed, feed)
    |> assign(:total_events, total_events)
    |> assign(:work_item_titles, work_item_titles)
  end

  defdelegate project_graph_opts(project), to: HtmlgraphDashboardWeb.ProjectHelpers

  defp collect_feature_ids(events) do
    Enum.flat_map(events, fn event ->
      id = event["feature_id"]
      children = event["children"] || []
      [id | collect_feature_ids(children)]
    end)
  end

  # --- Template Helpers ---

  defp tool_chip_class(tool_name) do
    case tool_name do
      "Bash" -> "tool-chip tool-chip-bash"
      "Read" -> "tool-chip tool-chip-read"
      "Edit" -> "tool-chip tool-chip-edit"
      "Write" -> "tool-chip tool-chip-write"
      "Grep" -> "tool-chip tool-chip-grep"
      "Glob" -> "tool-chip tool-chip-glob"
      "Task" -> "tool-chip tool-chip-task"
      "Agent" -> "tool-chip tool-chip-task"
      "TodoWrite" -> "tool-chip tool-chip-edit"
      "TodoRead" -> "tool-chip tool-chip-read"
      "TaskCreate" -> "tool-chip tool-chip-task"
      "TaskOutput" -> "tool-chip tool-chip-task"
      "Stop" -> "tool-chip tool-chip-stop"
      _ -> "tool-chip tool-chip-default"
    end
  end

  defp event_dot_class(event_type) do
    case event_type do
      "error" -> "error"
      "task_delegation" -> "task_delegation"
      "delegation" -> "delegation"
      "tool_result" -> "tool_result"
      _ -> "tool_call"
    end
  end

  defp format_timestamp(nil), do: ""

  defp format_timestamp(ts) when is_binary(ts) do
    case Regex.run(~r/(\d{2}:\d{2}:\d{2})/, ts) do
      [_, time] -> time
      _ -> ts
    end
  end

  defp format_duration(nil), do: ""
  defp format_duration(+0.0), do: ""

  defp format_duration(seconds) when is_number(seconds) do
    cond do
      seconds < 1 -> "#{round(seconds * 1000)}ms"
      seconds < 60 -> "#{Float.round(seconds * 1.0, 1)}s"
      true -> "#{round(seconds / 60)}m"
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

  defp format_session_id(session_id) when is_binary(session_id) do
    case Regex.run(
           ~r/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/i,
           session_id
         ) do
      [uuid] -> String.slice(uuid, 0, 8)
      nil -> String.slice(session_id, 0, 12)
    end
  end

  defp format_session_id(_), do: "unknown"

  defp has_children?(event) do
    children = event["children"] || []
    length(children) > 0
  end

  defp descendant_count(event) do
    event["descendant_count"] || child_count(event)
  end

  defp child_count(event) do
    children = event["children"] || []
    length(children)
  end

  defp is_expanded?(expanded, event_id) do
    MapSet.member?(expanded, event_id)
  end

  defp session_expanded?(expanded, session_id) do
    MapSet.member?(expanded, "session:#{session_id}")
  end

  defp infer_work_item_type(id) when is_binary(id) do
    cond do
      String.starts_with?(id, "feat-") -> "feature"
      String.starts_with?(id, "bug-") -> "bug"
      String.starts_with?(id, "spk-") -> "spike"
      true -> "feature"
    end
  end

  defp infer_work_item_type(_), do: "feature"

  defp depth_class(depth) do
    case depth do
      0 -> "depth-0"
      1 -> "depth-1"
      2 -> "depth-2"
      _ -> "depth-3"
    end
  end

  defp is_task_event?(event) do
    event["event_type"] == "task_delegation" or
      (event["tool_name"] == "Task" and event["subagent_type"] != nil)
  end

  defp row_border_class(event) do
    cond do
      is_task_event?(event) -> "border-task"
      event["event_type"] == "error" -> "border-error"
      true -> ""
    end
  end

  defp summary_text(event) do
    input = event["input_summary"] || ""
    output = event["output_summary"] || ""

    cond do
      input != "" -> input
      output != "" -> output
      true -> ""
    end
  end

  defp session_title(group) do
    # Prefer last_user_query from the session record
    lq = group.session && group.session["last_user_query"]

    if is_binary(lq) and String.trim(lq) != "" do
      truncate(String.trim(lq), 80)
    else
      # Fall back to first turn's prompt text
      case group.turns do
        [first | _] ->
          text = first.user_query["input_summary"] || ""

          if String.trim(text) != "" do
            truncate(text, 80)
          else
            format_session_id(group.session_id)
          end

        [] ->
          format_session_id(group.session_id)
      end
    end
  end

  defp agent_label(nil), do: nil
  defp agent_label("system"), do: nil
  defp agent_label(""), do: nil
  defp agent_label(name), do: name

  defp format_relative_time(nil), do: ""

  defp format_relative_time(ts) when is_binary(ts) do
    case NaiveDateTime.from_iso8601(ts) do
      {:ok, ndt} ->
        diff = NaiveDateTime.diff(NaiveDateTime.utc_now(), ndt, :second)

        cond do
          diff < 60 -> "just now"
          diff < 3600 -> "#{div(diff, 60)}m ago"
          diff < 86400 -> "#{div(diff, 3600)}h ago"
          true -> "#{div(diff, 86400)}d ago"
        end

      _ ->
        format_timestamp(ts)
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
          <%= @total_events %> conversation turns
        </div>
      </div>
    </div>

    <nav class="dashboard-nav">
      <a href="/" class="nav-tab active">Activity Feed</a>
      <a href="/graph" class="nav-tab">Graph</a>
      <a href="/kanban" class="nav-tab">Kanban</a>
      <a href="/costs" class="nav-tab">Costs</a>
      <a href="/projects" class="nav-tab">Projects</a>
      <%= if length(@projects) > 1 do %>
        <div class="project-selector" style="margin-left: auto; display: flex; align-items: center; gap: 0.5rem;">
          <span style="color: #888; font-size: 0.8rem;">Project:</span>
          <form phx-change="select_project" style="margin: 0;">
            <select name="project_id" style="background: #1C1C20; color: #e0ded8; border: 1px solid #333; padding: 0.25rem 0.5rem; font-size: 0.8rem;">
              <%= for project <- @projects do %>
                <option value={project.id} selected={@selected_project && project.id == @selected_project.id}>
                  <%= project.name %>
                </option>
              <% end %>
            </select>
          </form>
        </div>
      <% end %>
    </nav>

    <div class="graph-stats-bar">
      <div class="stat-card">
        <span class="stat-label">Sessions</span>
        <span class="stat-value"><%= @activity_stats["sessions"] || 0 %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Events</span>
        <span class="stat-value"><%= @activity_stats["events"] || 0 %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Features</span>
        <span class="stat-value"><%= @activity_stats["features"] || 0 %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Bugs</span>
        <span class="stat-value"><%= @activity_stats["bugs"] || 0 %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Active</span>
        <span class={"stat-value #{if (@activity_stats["active"] || 0) > 0, do: "stat-active"}"}>
          <%= @activity_stats["active"] || 0 %>
        </span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Tools</span>
        <span class="stat-value"><%= @activity_stats["tools"] || 0 %></span>
      </div>
    </div>

    <%= if @selected_work_item do %>
      <.work_item_detail_panel
        selected_work_item={@selected_work_item}
      />
    <% end %>

    <div class="feed-container">
      <%= if @feed == [] do %>
        <div class="empty-state">
          <h2>No activity yet</h2>
          <p>Events will appear here as agents work. The feed updates in real-time.</p>
        </div>
      <% else %>
        <%= for group <- @feed do %>
          <div class="session-group" phx-key={"session-#{group.session_id}"}>
            <!-- Session Header -->
            <div
              class="session-header"
              phx-click="toggle_session"
              phx-value-session-id={group.session_id}
            >
              <div class="session-info">
                <span class="toggle-btn">
                  <span class={["arrow", session_expanded?(@expanded, group.session_id) && "expanded"]}>
                    &#9654;
                  </span>
                </span>
                <span class="summary-text prompt">
                  <%= session_title(group) %>
                </span>
                <%= if group.session do %>
                  <span class={"badge badge-status-#{group.session["status"] || "active"}"}>
                    <%= group.session["status"] || "active" %>
                  </span>
                  <%= if agent_label(group.session["agent_assigned"]) do %>
                    <span class="badge badge-agent">
                      <%= agent_label(group.session["agent_assigned"]) %>
                    </span>
                  <% end %>
                <% end %>
              </div>
              <div class="stats-badges">
                <span class="badge badge-count">
                  <%= length(group.turns) %> turns
                </span>
                <%= if group.session do %>
                  <span class="badge badge-count">
                    <%= group.session["total_events"] || 0 %> events
                  </span>
                <% end %>
              </div>
            </div>

            <!-- Session subtitle: time + session ID -->
            <div
              class="session-subtitle"
              style={unless(session_expanded?(@expanded, group.session_id) || @session_filter, do: "display: none")}
            >
              <%= if group.session do %>
                <span class="timestamp">
                  Started: <%= format_relative_time(group.session["created_at"]) %>
                </span>
                <span class="badge badge-session" style="font-size: 10px;">
                  <%= format_session_id(group.session_id) %>
                </span>
                <%= if group.session["model"] do %>
                  <span class="badge badge-model"><%= group.session["model"] %></span>
                <% end %>
              <% end %>
            </div>

            <!-- Activity Table (shown when session expanded or no filter) -->
            <div
              class="activity-list"
              style={unless(session_expanded?(@expanded, group.session_id) || @session_filter, do: "display: none")}
            >
              <%= for turn <- group.turns do %>
                <!-- UserQuery Parent Row -->
                <div
                  class="activity-row parent-row"
                  phx-key={"turn-#{turn.user_query["event_id"]}"}
                >
                  <div class="row-toggle">
                    <%= if length(turn.children) > 0 do %>
                      <button
                        class="toggle-btn"
                        phx-click="toggle"
                        phx-value-event-id={turn.user_query["event_id"]}
                      >
                        <span class={["arrow", is_expanded?(@expanded, turn.user_query["event_id"]) && "expanded"]}>
                          &#9654;
                        </span>
                      </button>
                    <% end %>
                  </div>
                  <div class="row-content">
                    <div class="row-summary">
                      <span class="summary-text prompt">
                        <%= truncate(turn.user_query["input_summary"], 100) %>
                      </span>
                    </div>
                    <div class="row-meta">
                      <span class="badge badge-count">
                        <%= turn.stats.tool_count %> tools
                      </span>
                      <%= if turn.stats.error_count > 0 do %>
                        <span class="badge badge-error">
                          <%= turn.stats.error_count %> errors
                        </span>
                      <% end %>
                      <%= if turn.user_query["feature_id"] && @work_item_titles[turn.user_query["feature_id"]] do %>
                        <% wi = @work_item_titles[turn.user_query["feature_id"]] %>
                        <% wi_type = wi["type"] || infer_work_item_type(turn.user_query["feature_id"]) %>
                        <span
                          class={"badge badge-workitem badge-workitem-#{wi_type}"}
                          phx-click="select_work_item"
                          phx-value-id={turn.user_query["feature_id"]}
                          title={turn.user_query["feature_id"]}
                        >
                          <%= truncate(wi["title"] || "", 30) %>
                        </span>
                      <% end %>
                      <%= for model <- turn.stats.models do %>
                        <span class="badge badge-model"><%= model %></span>
                      <% end %>
                      <span class="timestamp">
                        <%= format_timestamp(turn.user_query["timestamp"]) %>
                      </span>
                      <span class="duration">
                        <%= format_duration(turn.stats.total_duration) %>
                      </span>
                    </div>
                  </div>
                </div>

                <!-- Child Events (nested, collapsible) -->
                <%= if is_expanded?(@expanded, turn.user_query["event_id"]) do %>
                  <%= for child <- turn.children do %>
                    <.event_row
                      event={child}
                      expanded={@expanded}
                    />
                  <% end %>
                <% end %>
              <% end %>
            </div>
          </div>
        <% end %>
      <% end %>
    </div>
    """
  end

  defp event_row(assigns) do
    ~H"""
    <div
      class={[
        "activity-row child-row",
        depth_class(@event["depth"] || 0),
        row_border_class(@event)
      ]}
      style={"padding-left: #{((@event["depth"] || 0) + 1) * 1.25}rem"}
      phx-key={"event-#{@event["event_id"]}"}
    >
      <div class="row-toggle">
        <%= if has_children?(@event) do %>
          <button
            class="toggle-btn"
            phx-click="toggle"
            phx-value-event-id={@event["event_id"]}
          >
            <span class={["arrow", is_expanded?(@expanded, @event["event_id"]) && "expanded"]}>
              &#9654;
            </span>
          </button>
        <% end %>
      </div>
      <div class="row-content">
        <div class="row-summary">
          <span class={"event-dot #{event_dot_class(@event["event_type"])}"}>
          </span>
          <span class={tool_chip_class(@event["tool_name"])}>
            <%= @event["tool_name"] %>
          </span>
          <span class="summary-text">
            <%= truncate(summary_text(@event), 80) %>
          </span>
          <%= if @event["step_id"] do %>
            <span class="badge badge-step" title={@event["step_id"]}>
              Step <%= @event["step_id"] |> String.split("-") |> List.last() %>
            </span>
          <% end %>
        </div>
        <div class="row-meta">
          <%= if @event["subagent_type"] do %>
            <span class="badge badge-subagent">
              <%= @event["subagent_type"] %>
            </span>
          <% end %>
          <%= if @event["model"] do %>
            <span class="badge badge-model">
              <%= @event["model"] %>
            </span>
          <% end %>
          <%= if @event["event_type"] == "error" do %>
            <span class="badge badge-error">error</span>
          <% end %>
          <%= if has_children?(@event) do %>
            <span class="badge badge-count">
              (<%= descendant_count(@event) %>)
            </span>
          <% end %>
          <span class="timestamp">
            <%= format_timestamp(@event["timestamp"]) %>
          </span>
          <span class="duration">
            <%= format_duration(@event["execution_duration_seconds"]) %>
          </span>
        </div>
      </div>
    </div>

    <!-- Recursive children -->
    <%= if is_expanded?(@expanded, @event["event_id"]) do %>
      <%= for child <- (@event["children"] || []) do %>
        <.event_row
          event={child}
          expanded={@expanded}
        />
      <% end %>
    <% end %>
    """
  end

  defp work_item_detail_panel(assigns) do
    ~H"""
    <div
      style="background: var(--bg-secondary); border: 1px solid var(--border); border-radius: var(--radius); margin: 12px 24px; padding: 16px;"
    >
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
        <span style="font-weight: 600; font-size: 15px;">
          <%= @selected_work_item["title"] || "Work Item" %>
        </span>
        <button
          phx-click="close_work_item_detail"
          style="background: none; border: none; color: var(--text-secondary); cursor: pointer; font-size: 16px;"
        >
          &#10005;
        </button>
      </div>

      <div style="display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 8px;">
        <span class={"badge badge-status-#{@selected_work_item["status"] || "active"}"}>
          <%= @selected_work_item["status"] || "unknown" %>
        </span>
        <span class="badge badge-session" style="font-size: 10px;">
          <%= @selected_work_item["id"] %>
        </span>
      </div>

      <%= if is_list(@selected_work_item["steps"]) and length(@selected_work_item["steps"]) > 0 do %>
        <div style="margin-top: 10px; font-size: 12px; color: var(--text-secondary);">
          <div style="font-weight: 600; margin-bottom: 4px;">Steps</div>
          <%= for step <- @selected_work_item["steps"] do %>
            <div style="padding: 2px 0; display: flex; align-items: center; gap: 6px;">
              <span><%= if step["completed"], do: "done", else: "pending" %></span>
              <span><%= step["description"] || "" %></span>
              <%= if step["step_id"] do %>
                <span class="badge badge-step"><%= step["step_id"] %></span>
              <% end %>
            </div>
          <% end %>
        </div>
      <% end %>
    </div>
    """
  end
end
