defmodule HtmlgraphDashboardWeb.KanbanLive do
  @moduledoc """
  Kanban board view showing work items organized by status columns.

  Columns: Todo | In Progress | Blocked | Done
  Cards display title, priority badge, type badge, and step progress bar.
  """
  use HtmlgraphDashboardWeb, :live_view

  alias HtmlgraphDashboard.ProjectRegistry
  alias HtmlgraphDashboard.PythonSDK
  alias HtmlgraphDashboard.Repo

  @columns [
    %{key: "todo", label: "Todo", color: "#94a3b8"},
    %{key: "in-progress", label: "In Progress", color: "#60a5fa"},
    %{key: "blocked", label: "Blocked", color: "#f87171"},
    %{key: "done", label: "Done", color: "#34d399"}
  ]

  @page_size 25

  @impl true
  def mount(params, _session, socket) do
    projects = ProjectRegistry.list_projects()
    selected_project_id = params["project"] || (List.first(projects, %{}) |> Map.get(:id))
    selected_project = Enum.find(projects, List.first(projects), &(&1.id == selected_project_id))

    items = load_kanban_data(selected_project && selected_project.id)

    socket =
      socket
      |> assign(:active_tab, :kanban)
      |> assign(:items, items)
      |> assign(:columns, @columns)
      |> assign(:selected_card, nil)
      |> assign(:projects, projects)
      |> assign(:selected_project, selected_project)
      |> assign(:items_shown, default_items_shown())

    {:ok, socket}
  end

  @impl true
  def handle_params(params, _uri, socket) do
    case params["project"] do
      nil ->
        {:noreply, socket}

      project_id ->
        project = Enum.find(socket.assigns.projects, socket.assigns.selected_project, &(&1.id == project_id))
        items = load_kanban_data(project && project.id)

        {:noreply,
         socket
         |> assign(:selected_project, project)
         |> assign(:items, items)
         |> assign(:selected_card, nil)
         |> assign(:items_shown, default_items_shown())}
    end
  end

  @impl true
  def handle_event("select_card", %{"id" => card_id}, socket) do
    card = Enum.find(socket.assigns.items, fn i -> i["id"] == card_id end)
    project = socket.assigns[:selected_project]

    detail =
      if card do
        load_work_item_detail(card_id, project) || card
      else
        nil
      end

    {:noreply, assign(socket, :selected_card, detail)}
  end

  def handle_event("close_detail", _params, socket) do
    {:noreply, assign(socket, :selected_card, nil)}
  end

  def handle_event("refresh_kanban", _params, socket) do
    project_id = socket.assigns[:selected_project] && socket.assigns.selected_project.id
    items = load_kanban_data(project_id)

    socket =
      socket
      |> assign(:items, items)
      |> assign(:selected_card, nil)
      |> assign(:items_shown, default_items_shown())

    {:noreply, socket}
  end

  def handle_event("select_project", %{"project_id" => project_id}, socket) do
    project = Enum.find(socket.assigns.projects, &(&1.id == project_id))
    items = load_kanban_data(project && project.id)

    socket =
      socket
      |> assign(:selected_project, project)
      |> assign(:items, items)
      |> assign(:selected_card, nil)
      |> assign(:items_shown, default_items_shown())

    {:noreply, push_patch(socket, to: "/kanban?project=#{project_id}")}
  end

  def handle_event("show_more", %{"column" => column_key}, socket) do
    current = Map.get(socket.assigns.items_shown, column_key, @page_size)
    updated = Map.put(socket.assigns.items_shown, column_key, current + @page_size)
    {:noreply, assign(socket, :items_shown, updated)}
  end

  defp load_kanban_data(project_id) do
    sql = """
    SELECT
      id,
      title,
      status,
      type,
      priority,
      steps_total,
      steps_completed
    FROM features
    WHERE type IN ('feature', 'bug', 'spike')
    ORDER BY
      CASE status
        WHEN 'in-progress' THEN 0
        WHEN 'todo' THEN 1
        WHEN 'blocked' THEN 2
        WHEN 'done' THEN 3
        ELSE 4
      END,
      title ASC
    """

    case Repo.query_maps(sql, [], project_id) do
      {:ok, rows} ->
        Enum.map(rows, fn row ->
          %{
            "id" => row["id"],
            "title" => row["title"],
            "status" => row["status"] || "todo",
            "type" => row["type"] || "feature",
            "priority" => row["priority"] || "medium",
            "steps_total" => row["steps_total"] || 0,
            "steps_completed" => row["steps_completed"] || 0
          }
        end)

      {:error, reason} ->
        require Logger
        Logger.error("KanbanLive: load_kanban_data failed: #{inspect(reason)}")
        []
    end
  end

  defp project_graph_opts(nil), do: %{}

  defp project_graph_opts(project) do
    case ProjectRegistry.get_project(project.id) do
      %{db_path: db_path} ->
        graph_dir = db_path |> Path.dirname() |> Path.dirname()
        %{db_path: db_path, graph_dir: graph_dir}

      nil ->
        %{}
    end
  end

  defp load_work_item_detail(id, project) do
    opts = project_graph_opts(project)

    try do
      case PythonSDK.get_work_item(id, opts) do
        {:ok, item} when is_map(item) -> item
        _ -> nil
      end
    rescue
      _ -> nil
    catch
      :exit, _ -> nil
    end
  end

  defp default_items_shown do
    Enum.into(@columns, %{}, fn col -> {col.key, @page_size} end)
  end

  defp items_for_column(items, status) do
    Enum.filter(items, fn i -> i["status"] == status end)
  end

  defp visible_items_for_column(items, status, items_shown) do
    all = items_for_column(items, status)
    limit = Map.get(items_shown, status, @page_size)
    Enum.take(all, limit)
  end

  defp column_count(items, status) do
    length(items_for_column(items, status))
  end

  defp total_count(items), do: length(items)

  defp priority_class(priority) do
    case priority do
      "critical" -> "kanban-card priority-critical"
      "high" -> "kanban-card priority-high"
      "medium" -> "kanban-card priority-medium"
      "low" -> "kanban-card priority-low"
      _ -> "kanban-card priority-medium"
    end
  end

  defp priority_badge_class(priority) do
    case priority do
      "critical" -> "badge kanban-badge-critical"
      "high" -> "badge kanban-badge-high"
      "medium" -> "badge kanban-badge-medium"
      "low" -> "badge kanban-badge-low"
      _ -> "badge kanban-badge-medium"
    end
  end

  defp type_badge_class(type) do
    case type do
      "bug" -> "badge badge-error"
      "spike" -> "badge badge-step"
      _ -> "badge badge-feature"
    end
  end

  defp progress_percent(item) do
    total = item["steps_total"] || 0
    completed = item["steps_completed"] || 0

    if total > 0 do
      round(completed / total * 100)
    else
      0
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

  defp status_badge_class(status) do
    case status do
      "done" -> "badge badge-status-completed"
      "in-progress" -> "badge badge-status-active"
      "blocked" -> "badge badge-error"
      _ -> "badge badge-count"
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
          <%= total_count(@items) %> work items
        </div>
      </div>
    </div>

    <nav class="dashboard-nav">
      <a href="/" class="nav-tab">Activity Feed</a>
      <a href="/graph" class="nav-tab">Graph</a>
      <a href="/kanban" class="nav-tab active">Kanban</a>
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

    <div class="kanban-toolbar">
      <button phx-click="refresh_kanban" class="graph-refresh-btn">
        Refresh
      </button>
    </div>

    <div class="kanban-board">
      <%= for col <- @columns do %>
        <div class="kanban-column">
          <div class="kanban-column-header">
            <span>
              <span class="kanban-column-dot" style={"background: #{col.color};"}></span>
              <%= col.label %>
            </span>
            <span class="kanban-column-count">
              <%= column_count(@items, col.key) %>
            </span>
          </div>

          <div class="kanban-column-body">
            <%= if items_for_column(@items, col.key) == [] do %>
              <div class="kanban-empty">No items</div>
            <% else %>
              <%= for item <- visible_items_for_column(@items, col.key, @items_shown) do %>
                <div
                  class={priority_class(item["priority"])}
                  phx-click="select_card"
                  phx-value-id={item["id"]}
                >
                  <div class="kanban-card-title">
                    <%= truncate(item["title"], 60) %>
                  </div>
                  <div class="kanban-card-meta">
                    <span class={type_badge_class(item["type"])}>
                      <%= item["type"] || "feature" %>
                    </span>
                    <span class={priority_badge_class(item["priority"])}>
                      <%= item["priority"] || "medium" %>
                    </span>
                  </div>
                  <%= if (item["steps_total"] || 0) > 0 do %>
                    <div class="kanban-card-progress">
                      <div class="kanban-progress">
                        <div
                          class="kanban-progress-bar"
                          style={"width: #{progress_percent(item)}%;"}
                        >
                        </div>
                      </div>
                      <span class="kanban-progress-label">
                        <%= item["steps_completed"] || 0 %>/<%= item["steps_total"] || 0 %>
                      </span>
                    </div>
                  <% end %>
                </div>
              <% end %>
              <%= if column_count(@items, col.key) > Map.get(@items_shown, col.key, 25) do %>
                <button
                  class="kanban-show-more"
                  phx-click="show_more"
                  phx-value-column={col.key}
                >
                  Show <%= column_count(@items, col.key) - Map.get(@items_shown, col.key, 25) %> more
                </button>
              <% end %>
            <% end %>
          </div>
        </div>
      <% end %>
    </div>

    <!-- Detail Panel -->
    <%= if @selected_card do %>
      <div class="kanban-detail-overlay" phx-click="close_detail">
        <div class="kanban-detail-panel" phx-click-away="close_detail">
          <div class="kanban-detail-header">
            <span class="kanban-detail-title">
              <%= @selected_card["title"] || "Untitled" %>
            </span>
            <button phx-click="close_detail" class="graph-detail-close">
              &#10005;
            </button>
          </div>

          <div class="kanban-detail-body">
            <div class="graph-detail-row">
              <span class="graph-detail-label">ID</span>
              <span class="badge badge-session" style="font-size: 10px;">
                <%= @selected_card["id"] %>
              </span>
            </div>
            <div class="graph-detail-row">
              <span class="graph-detail-label">Status</span>
              <span class={status_badge_class(@selected_card["status"])}>
                <%= @selected_card["status"] || "todo" %>
              </span>
            </div>
            <div class="graph-detail-row">
              <span class="graph-detail-label">Type</span>
              <span class={type_badge_class(@selected_card["type"])}>
                <%= @selected_card["type"] || "feature" %>
              </span>
            </div>
            <div class="graph-detail-row">
              <span class="graph-detail-label">Priority</span>
              <span class={priority_badge_class(@selected_card["priority"])}>
                <%= @selected_card["priority"] || "medium" %>
              </span>
            </div>

            <%= if is_list(@selected_card["steps"]) and length(@selected_card["steps"]) > 0 do %>
              <div class="kanban-detail-steps">
                <div class="graph-detail-label" style="margin-bottom: 8px;">Steps</div>
                <%= for step <- @selected_card["steps"] do %>
                  <div class="kanban-step-row">
                    <span class={"kanban-step-check #{if step["completed"], do: "completed"}"}>
                      <%= if step["completed"], do: raw("&#10003;"), else: raw("&#9675;") %>
                    </span>
                    <span class="kanban-step-text">
                      <%= step["description"] || "" %>
                    </span>
                  </div>
                <% end %>
              </div>
            <% end %>
          </div>
        </div>
      </div>
    <% end %>
    """
  end
end
