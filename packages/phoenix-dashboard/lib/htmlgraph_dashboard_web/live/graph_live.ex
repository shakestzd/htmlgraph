defmodule HtmlgraphDashboardWeb.GraphLive do
  @moduledoc """
  Dependency graph visualization showing work items as nodes and their
  relationships as directed edges.

  Layout uses topological sorting to position nodes left-to-right by
  dependency depth, rendered as SVG within the LiveView.
  """
  use HtmlgraphDashboardWeb, :live_view

  alias HtmlgraphDashboard.PythonSDK

  @default_graph %{
    "nodes" => [],
    "edges" => [],
    "critical_path" => [],
    "viewbox_width" => 920,
    "viewbox_height" => 460
  }

  @impl true
  def mount(_params, _session, socket) do
    graph_data = load_dependency_graph()

    if connected?(socket) do
      :timer.send_interval(30_000, self(), :refresh_graph)
    end

    socket =
      socket
      |> assign(:active_tab, :graph)
      |> assign(:graph_data, graph_data)
      |> assign(:selected_node, nil)

    {:ok, socket}
  end

  @impl true
  def handle_info(:refresh_graph, socket) do
    graph_data = load_dependency_graph()
    {:noreply, assign(socket, :graph_data, graph_data)}
  end

  @impl true
  def handle_event("select_node", %{"id" => node_id}, socket) do
    node =
      Enum.find(socket.assigns.graph_data["nodes"] || [], fn n ->
        n["id"] == node_id
      end)

    {:noreply, assign(socket, :selected_node, node)}
  end

  def handle_event("close_detail", _params, socket) do
    {:noreply, assign(socket, :selected_node, nil)}
  end

  def handle_event("refresh_graph", _params, socket) do
    graph_data = load_dependency_graph()

    socket =
      socket
      |> assign(:graph_data, graph_data)
      |> assign(:selected_node, nil)

    {:noreply, socket}
  end

  defp load_dependency_graph do
    try do
      case PythonSDK.get_dependency_graph() do
        {:ok, data} when is_map(data) -> Map.merge(@default_graph, data)
        _ -> @default_graph
      end
    rescue
      _ -> @default_graph
    catch
      :exit, _ -> @default_graph
    end
  end

  defp node_count(graph_data), do: length(graph_data["nodes"] || [])
  defp edge_count(graph_data), do: length(graph_data["edges"] || [])

  defp critical_count(graph_data) do
    length(graph_data["critical_path"] || [])
  end

  defp bottleneck_count(graph_data) do
    (graph_data["nodes"] || [])
    |> Enum.count(fn n -> n["is_bottleneck"] == true end)
  end

  defp node_radius(node) do
    cond do
      node["is_bottleneck"] -> 20
      node["is_critical"] -> 16
      true -> 12
    end
  end

  defp node_status_class(node) do
    case node["status"] do
      "done" -> "node-done"
      "in-progress" -> "node-in-progress"
      "blocked" -> "node-blocked"
      _ -> "node-todo"
    end
  end

  defp node_status_color(node) do
    case node["status"] do
      "in-progress" -> "#22c55e"
      "todo" -> "#3b82f6"
      "done" -> "#6b7280"
      "blocked" -> "#ef4444"
      _ -> "#8b5cf6"
    end
  end

  defp truncate_label(nil), do: ""

  defp truncate_label(text) when is_binary(text) do
    if String.length(text) > 22 do
      String.slice(text, 0, 22) <> "..."
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

  defp type_label(nil), do: "feature"
  defp type_label(type), do: type

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
        <button
          phx-click="refresh_graph"
          class="graph-refresh-btn"
        >
          Refresh
        </button>
      </div>
    </div>

    <div class="graph-container">
      <div class="graph-viewport">
        <%= if node_count(@graph_data) == 0 do %>
          <div class="empty-state">
            <h2>No dependency graph</h2>
            <p>
              Work items with relationships will appear here.
              Create features with dependency edges to see the graph.
            </p>
          </div>
        <% else %>
          <div style="overflow: auto; max-height: 600px;">
            <svg
              viewBox={"0 0 #{@graph_data["viewbox_width"] || 920} #{@graph_data["viewbox_height"] || 460}"}
              width="100%"
              style={"min-height: 200px; height: #{@graph_data["viewbox_height"] || 460}px; max-height: 600px; background: transparent;"}
              xmlns="http://www.w3.org/2000/svg"
            >
              <defs>
                <marker id="arrowhead" markerWidth="8" markerHeight="6"
                        refX="8" refY="3" orient="auto" fill="#475569">
                  <polygon points="0 0, 8 3, 0 6" />
                </marker>
                <marker id="arrowhead-blocks" markerWidth="8" markerHeight="6"
                        refX="8" refY="3" orient="auto" fill="#f87171">
                  <polygon points="0 0, 8 3, 0 6" />
                </marker>
                <marker id="arrowhead-relates" markerWidth="8" markerHeight="6"
                        refX="8" refY="3" orient="auto" fill="#60a5fa">
                  <polygon points="0 0, 8 3, 0 6" />
                </marker>
                <marker id="arrowhead-spawned" markerWidth="8" markerHeight="6"
                        refX="8" refY="3" orient="auto" fill="#a78bfa">
                  <polygon points="0 0, 8 3, 0 6" />
                </marker>
                <filter id="glow-critical">
                  <feGaussianBlur stdDeviation="3" result="blur" />
                  <feMerge>
                    <feMergeNode in="blur" />
                    <feMergeNode in="SourceGraphic" />
                  </feMerge>
                </filter>
              </defs>

              <!-- Edges (drawn first so nodes render on top) -->
              <%= for edge <- @graph_data["edges"] || [] do %>
                <line
                  x1={edge["x1"]}
                  y1={edge["y1"]}
                  x2={edge["x2"]}
                  y2={edge["y2"]}
                  stroke="#4b5563"
                  stroke-width="1.5"
                  opacity="0.5"
                  marker-end={edge_marker(edge)}
                />
              <% end %>

              <!-- Nodes -->
              <%= for node <- @graph_data["nodes"] || [] do %>
                <g
                  class={"graph-node #{node_status_class(node)} #{if node["is_critical"], do: "node-critical"} #{if node["is_bottleneck"], do: "node-bottleneck"}"}
                  phx-click="select_node"
                  phx-value-id={node["id"]}
                  style="cursor: pointer;"
                >
                  <%= if node["is_critical"] do %>
                    <circle
                      cx={node["x"]}
                      cy={node["y"]}
                      r={node_radius(node) + 5}
                      fill="none"
                      stroke="#fbbf24"
                      stroke-width="2"
                      opacity="0.6"
                      filter="url(#glow-critical)"
                    />
                  <% end %>
                  <circle
                    cx={node["x"]}
                    cy={node["y"]}
                    r={node_radius(node)}
                    fill={node["color"] || node_status_color(node)}
                    stroke={if node["is_bottleneck"], do: "#f87171", else: "rgba(255,255,255,0.2)"}
                    stroke-width={if node["is_bottleneck"], do: "2.5", else: "1"}
                  />
                  <!-- Type initial inside circle -->
                  <text
                    x={node["x"]}
                    y={node["y"] + 4}
                    text-anchor="middle"
                    fill="white"
                    font-size="9"
                    font-weight="bold"
                    style="pointer-events: none;"
                  >
                    <%= String.upcase(String.first(type_label(node["type"]) || "f")) %>
                  </text>
                  <!-- Label to the right of the node -->
                  <text
                    x={node["x"] + node_radius(node) + 6}
                    y={node["y"] + 4}
                    fill="#d1d5db"
                    font-size="11"
                    style="pointer-events: none;"
                  >
                    <%= truncate_label(node["title"]) %>
                  </text>
                </g>
              <% end %>
            </svg>
          </div>
        <% end %>
      </div>

      <!-- Detail Panel -->
      <%= if @selected_node do %>
        <div class="graph-detail-panel">
          <div class="graph-detail-header">
            <span class="graph-detail-title">
              <%= @selected_node["title"] || "Untitled" %>
            </span>
            <button phx-click="close_detail" class="graph-detail-close">
              &#10005;
            </button>
          </div>

          <div class="graph-detail-body">
            <div class="graph-detail-row">
              <span class="graph-detail-label">ID</span>
              <span class="badge badge-session" style="font-size: 10px;">
                <%= @selected_node["id"] %>
              </span>
            </div>
            <div class="graph-detail-row">
              <span class="graph-detail-label">Status</span>
              <span class={status_badge_class(@selected_node["status"])}>
                <%= @selected_node["status"] || "todo" %>
              </span>
            </div>
            <div class="graph-detail-row">
              <span class="graph-detail-label">Type</span>
              <span class="badge badge-count">
                <%= type_label(@selected_node["type"]) %>
              </span>
            </div>
            <div class="graph-detail-row">
              <span class="graph-detail-label">Priority</span>
              <span class={"badge priority-badge-#{@selected_node["priority"] || "medium"}"}>
                <%= @selected_node["priority"] || "medium" %>
              </span>
            </div>
            <div class="graph-detail-row">
              <span class="graph-detail-label">Depth</span>
              <span class="badge badge-count"><%= @selected_node["depth"] || 0 %></span>
            </div>

            <%= if @selected_node["is_critical"] do %>
              <div class="graph-detail-flag">
                <span class="badge badge-critical-path">On Critical Path</span>
              </div>
            <% end %>
            <%= if @selected_node["is_bottleneck"] do %>
              <div class="graph-detail-flag">
                <span class="bottleneck-warning">Bottleneck Node</span>
              </div>
            <% end %>
          </div>
        </div>
      <% end %>
    </div>
    """
  end

  defp edge_marker(edge) do
    case edge["relationship"] do
      "blocks" -> "url(#arrowhead-blocks)"
      "relates_to" -> "url(#arrowhead-relates)"
      "spawned_from" -> "url(#arrowhead-spawned)"
      _ -> "url(#arrowhead)"
    end
  end
end
