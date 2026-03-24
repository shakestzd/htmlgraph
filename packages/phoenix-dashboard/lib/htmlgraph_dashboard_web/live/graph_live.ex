defmodule HtmlgraphDashboardWeb.GraphLive do
  @moduledoc """
  Dependency graph visualization showing work items as nodes and their
  relationships as directed edges.

  Layout uses topological sorting to position nodes left-to-right by
  dependency depth, rendered as SVG within the LiveView.

  Rendering helpers and function components live in GraphComponents.
  """
  use HtmlgraphDashboardWeb, :live_view

  import HtmlgraphDashboardWeb.GraphComponents

  alias HtmlgraphDashboard.ProjectRegistry
  alias HtmlgraphDashboard.PythonSDK

  @default_graph %{
    "nodes" => [],
    "edges" => [],
    "critical_path" => [],
    "viewbox_width" => 920,
    "viewbox_height" => 460
  }

  @impl true
  def mount(params, _session, socket) do
    projects = ProjectRegistry.list_projects()
    selected_project_id = params["project"] || (List.first(projects, %{}) |> Map.get(:id))
    selected_project = Enum.find(projects, List.first(projects), &(&1.id == selected_project_id))

    graph_opts = project_graph_opts(selected_project)
    graph_data = load_dependency_graph(graph_opts)

    if connected?(socket) do
      :timer.send_interval(30_000, self(), :refresh_graph)
    end

    socket =
      socket
      |> assign(:active_tab, :graph)
      |> assign(:graph_data, graph_data)
      |> assign(:selected_node, nil)
      |> assign(:projects, projects)
      |> assign(:selected_project, selected_project)

    {:ok, socket}
  end

  @impl true
  def handle_params(params, _uri, socket) do
    case params["project"] do
      nil ->
        {:noreply, socket}

      project_id ->
        project = Enum.find(socket.assigns.projects, socket.assigns.selected_project, &(&1.id == project_id))
        graph_data = load_dependency_graph(project_graph_opts(project))

        {:noreply,
         socket
         |> assign(:selected_project, project)
         |> assign(:graph_data, graph_data)
         |> assign(:selected_node, nil)}
    end
  end

  @impl true
  def handle_info(:refresh_graph, socket) do
    project = socket.assigns[:selected_project]
    graph_data = load_dependency_graph(project_graph_opts(project))
    {:noreply, assign(socket, :graph_data, graph_data)}
  end

  @impl true
  def handle_event("select_node", %{"id" => node_id}, socket) do
    node = Enum.find(socket.assigns.graph_data["nodes"] || [], &(&1["id"] == node_id))
    {:noreply, assign(socket, :selected_node, node)}
  end

  def handle_event("close_detail", _params, socket) do
    {:noreply, assign(socket, :selected_node, nil)}
  end

  def handle_event("refresh_graph", _params, socket) do
    project = socket.assigns[:selected_project]
    graph_data = load_dependency_graph(project_graph_opts(project))

    {:noreply,
     socket
     |> assign(:graph_data, graph_data)
     |> assign(:selected_node, nil)}
  end

  def handle_event("select_project", %{"project_id" => project_id}, socket) do
    project = Enum.find(socket.assigns.projects, &(&1.id == project_id))
    graph_data = load_dependency_graph(project_graph_opts(project))

    {:noreply,
     socket
     |> assign(:selected_project, project)
     |> assign(:graph_data, graph_data)
     |> assign(:selected_node, nil)
     |> push_patch(to: "/graph?project=#{project_id}")}
  end

  # ---------------------------------------------------------------------------
  # Private — data loading
  # ---------------------------------------------------------------------------

  defdelegate project_graph_opts(project), to: HtmlgraphDashboardWeb.ProjectHelpers

  defp load_dependency_graph(opts) do
    try do
      case PythonSDK.get_dependency_graph(opts) do
        {:ok, data} when is_map(data) -> Map.merge(@default_graph, data)
        {:error, msg} ->
          require Logger
          Logger.error("GraphLive: dependency graph failed: #{msg}")
          @default_graph
        _ -> @default_graph
      end
    rescue
      e ->
        require Logger
        Logger.error("GraphLive: dependency graph exception: #{Exception.message(e)}")
        @default_graph
    catch
      :exit, reason ->
        require Logger
        Logger.error("GraphLive: dependency graph exit: #{inspect(reason)}")
        @default_graph
    end
  end

  # ---------------------------------------------------------------------------
  # Private — statistics (used in template)
  # ---------------------------------------------------------------------------

  defp node_count(graph_data), do: length(graph_data["nodes"] || [])
  defp edge_count(graph_data), do: length(graph_data["edges"] || [])
  defp critical_count(graph_data), do: length(graph_data["critical_path"] || [])

  defp bottleneck_count(graph_data) do
    Enum.count(graph_data["nodes"] || [], & &1["is_bottleneck"] == true)
  end

  # ---------------------------------------------------------------------------
  # Render
  # ---------------------------------------------------------------------------

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
      </div>
    </div>

    <nav class="dashboard-nav">
      <a href="/" class="nav-tab">Activity Feed</a>
      <a href="/graph" class="nav-tab active">Graph</a>
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
        <span class="stat-label">Nodes</span>
        <span class="stat-value"><%= node_count(@graph_data) %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Edges</span>
        <span class="stat-value"><%= edge_count(@graph_data) %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Critical Path</span>
        <span class="stat-value"><%= critical_count(@graph_data) %></span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Bottlenecks</span>
        <span class={"stat-value #{if bottleneck_count(@graph_data) > 0, do: "stat-warning"}"}>
          <%= bottleneck_count(@graph_data) %>
        </span>
      </div>
      <div class="stat-card" style="margin-left: auto;">
        <button phx-click="refresh_graph" class="graph-refresh-btn">Refresh</button>
      </div>
    </div>

    <div class="graph-container">
      <div class="graph-viewport">
        <%= if node_count(@graph_data) == 0 do %>
          <div class="empty-state">
            <h2>No dependency graph</h2>
            <p>Work items with relationships will appear here.
              Create features with dependency edges to see the graph.</p>
          </div>
        <% else %>
          <.svg_graph graph_data={@graph_data} />
        <% end %>
      </div>

      <%= if @selected_node do %>
        <.detail_panel node={@selected_node} graph_data={@graph_data} />
      <% end %>
    </div>
    """
  end
end
