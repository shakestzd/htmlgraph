defmodule HtmlgraphDashboardWeb.GraphComponents do
  @moduledoc """
  Function components for the graph view: SVG canvas and detail panel.
  Extracted from GraphLive to keep the LiveView module under 500 lines.
  """
  use Phoenix.Component

  # ---------------------------------------------------------------------------
  # Node helpers (imported into components)
  # ---------------------------------------------------------------------------

  def node_radius(node) do
    cond do
      node["is_bottleneck"] -> 28
      node["is_critical"] -> 24
      true -> 22
    end
  end

  def node_status_class(node) do
    case node["status"] do
      "done" -> "node-done"
      "in-progress" -> "node-in-progress"
      "blocked" -> "node-blocked"
      _ -> "node-todo"
    end
  end

  def node_status_color(node) do
    case node["status"] do
      "in-progress" -> "#22c55e"
      "todo" -> "#3b82f6"
      "done" -> "#6b7280"
      "blocked" -> "#ef4444"
      _ -> "#8b5cf6"
    end
  end

  def truncate_label(nil), do: ""

  def truncate_label(text) when is_binary(text) do
    if String.length(text) > 28 do
      String.slice(text, 0, 28) <> "..."
    else
      text
    end
  end

  def type_label(nil), do: "feature"
  def type_label(type), do: type

  def status_badge_class(status) do
    case status do
      "done" -> "badge badge-status-completed"
      "in-progress" -> "badge badge-status-active"
      "blocked" -> "badge badge-error"
      _ -> "badge badge-count"
    end
  end

  def edge_marker(edge) do
    case edge["relationship"] do
      "blocks" -> "url(#arrowhead-blocks)"
      "relates_to" -> "url(#arrowhead-relates)"
      "spawned_from" -> "url(#arrowhead-spawned)"
      _ -> "url(#arrowhead)"
    end
  end

  def edge_stroke_color(edge) do
    case edge["relationship"] do
      "blocks" -> "#f87171"
      "relates_to" -> "#60a5fa"
      "spawned_from" -> "#a78bfa"
      _ -> "#94a3b8"
    end
  end

  def curved_path(edge, index) do
    x1 = edge["x1"] || 0
    y1 = edge["y1"] || 0
    x2 = edge["x2"] || 0
    y2 = edge["y2"] || 0
    mx = (x1 + x2) / 2
    my = (y1 + y2) / 2
    dx = x2 - x1
    dy = y2 - y1
    length = :math.sqrt(dx * dx + dy * dy)
    offset = if rem(index, 2) == 0, do: 30, else: -30
    {px, py} =
      if length > 0 do
        {mx + offset * (-dy / length), my + offset * (dx / length)}
      else
        {mx, my}
      end
    "M #{x1} #{y1} Q #{px} #{py} #{x2} #{y2}"
  end

  def connected_nodes(node, graph_data) do
    node_id = node["id"]
    edges = graph_data["edges"] || []
    nodes = graph_data["nodes"] || []
    node_map = Map.new(nodes, &{&1["id"], &1})

    # Python SDK produces "from"/"to" keys (not "source"/"target").
    blocks =
      edges
      |> Enum.filter(&(&1["from"] == node_id))
      |> Enum.map(fn e -> {e["relationship"] || "depends", Map.get(node_map, e["to"])} end)
      |> Enum.reject(fn {_, n} -> is_nil(n) end)

    blocked_by =
      edges
      |> Enum.filter(&(&1["to"] == node_id))
      |> Enum.map(fn e -> {e["relationship"] || "depends", Map.get(node_map, e["from"])} end)
      |> Enum.reject(fn {_, n} -> is_nil(n) end)

    %{blocks: blocks, blocked_by: blocked_by}
  end

  # ---------------------------------------------------------------------------
  # SVG graph canvas component
  # ---------------------------------------------------------------------------

  attr :graph_data, :map, required: true

  def svg_graph(assigns) do
    ~H"""
    <div class="graph-svg-wrapper">
      <svg
        phx-hook="GraphControls"
        id="graph-svg"
        viewBox={"0 0 #{@graph_data["viewbox_width"] || 920} #{@graph_data["viewbox_height"] || 460}"}
        width="100%"
        style={"min-height: 500px; height: #{@graph_data["viewbox_height"] || 460}px; background: transparent;"}
        xmlns="http://www.w3.org/2000/svg"
      >
        <defs>
          <marker id="arrowhead" markerWidth="12" markerHeight="9"
                  refX="11" refY="4.5" orient="auto" fill="#94a3b8">
            <polygon points="0 0, 12 4.5, 0 9" />
          </marker>
          <marker id="arrowhead-blocks" markerWidth="12" markerHeight="9"
                  refX="11" refY="4.5" orient="auto" fill="#f87171">
            <polygon points="0 0, 12 4.5, 0 9" />
          </marker>
          <marker id="arrowhead-relates" markerWidth="12" markerHeight="9"
                  refX="11" refY="4.5" orient="auto" fill="#60a5fa">
            <polygon points="0 0, 12 4.5, 0 9" />
          </marker>
          <marker id="arrowhead-spawned" markerWidth="12" markerHeight="9"
                  refX="11" refY="4.5" orient="auto" fill="#a78bfa">
            <polygon points="0 0, 12 4.5, 0 9" />
          </marker>
          <filter id="glow-critical">
            <feGaussianBlur stdDeviation="3" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        <%= for {edge, idx} <- Enum.with_index(@graph_data["edges"] || []) do %>
          <path
            d={curved_path(edge, idx)}
            fill="none"
            stroke={edge_stroke_color(edge)}
            stroke-width="2"
            opacity="0.65"
            marker-end={edge_marker(edge)}
          />
        <% end %>

        <%= for node <- @graph_data["nodes"] || [] do %>
          <g
            class={"graph-node #{node_status_class(node)} #{if node["is_critical"], do: "node-critical"} #{if node["is_bottleneck"], do: "node-bottleneck"}"}
            phx-click="select_node"
            phx-value-id={node["id"]}
            style="cursor: pointer;"
          >
            <title><%= node["title"] || node["id"] %></title>
            <%= if node["is_critical"] do %>
              <circle cx={node["x"]} cy={node["y"]} r={node_radius(node) + 7}
                fill="none" stroke="#fbbf24" stroke-width="2" opacity="0.55"
                filter="url(#glow-critical)" />
            <% end %>
            <circle
              cx={node["x"]} cy={node["y"]} r={node_radius(node)}
              fill={node["color"] || node_status_color(node)}
              stroke={if node["is_bottleneck"], do: "#f87171", else: "rgba(255,255,255,0.3)"}
              stroke-width={if node["is_bottleneck"], do: "3", else: "1.5"}
            />
            <text x={node["x"]} y={node["y"] + 5} text-anchor="middle"
              fill="white" font-size="13" font-weight="bold"
              style="pointer-events: none;">
              <%= String.upcase(String.first(type_label(node["type"]) || "f")) %>
            </text>
            <rect
              x={node["x"] + node_radius(node) + 6} y={node["y"] - 11}
              width={String.length(truncate_label(node["title"])) * 8 + 8}
              height="18" rx="3" fill="rgba(13,17,23,0.82)"
              style="pointer-events: none;" />
            <text x={node["x"] + node_radius(node) + 10} y={node["y"] + 4}
              fill="#e6edf3" font-size="14" font-weight="600"
              style="pointer-events: none;">
              <%= truncate_label(node["title"]) %>
            </text>
          </g>
        <% end %>
      </svg>
    </div>
    """
  end

  # ---------------------------------------------------------------------------
  # Detail panel component
  # ---------------------------------------------------------------------------

  attr :node, :map, required: true
  attr :graph_data, :map, required: true

  def detail_panel(assigns) do
    assigns = assign(assigns, :connections, connected_nodes(assigns.node, assigns.graph_data))
    ~H"""
    <div class="graph-detail-panel">
      <div class="graph-detail-header">
        <span class="graph-detail-title"><%= @node["title"] || "Untitled" %></span>
        <button phx-click="close_detail" class="graph-detail-close">&#10005;</button>
      </div>
      <div class="graph-detail-body">
        <div class="graph-detail-row">
          <span class="graph-detail-label">ID</span>
          <span class="badge badge-session" style="font-size: 10px;"><%= @node["id"] %></span>
        </div>
        <div class="graph-detail-row">
          <span class="graph-detail-label">Status</span>
          <span class={status_badge_class(@node["status"])}><%= @node["status"] || "todo" %></span>
        </div>
        <div class="graph-detail-row">
          <span class="graph-detail-label">Type</span>
          <span class="badge badge-count"><%= type_label(@node["type"]) %></span>
        </div>
        <div class="graph-detail-row">
          <span class="graph-detail-label">Priority</span>
          <span class={"badge priority-badge-#{@node["priority"] || "medium"}"}>
            <%= @node["priority"] || "medium" %>
          </span>
        </div>
        <div class="graph-detail-row">
          <span class="graph-detail-label">Depth</span>
          <span class="badge badge-count"><%= @node["depth"] || 0 %></span>
        </div>
        <%= if @node["is_critical"] do %>
          <div class="graph-detail-flag">
            <span class="badge badge-critical-path">On Critical Path</span>
          </div>
        <% end %>
        <%= if @node["is_bottleneck"] do %>
          <div class="graph-detail-flag">
            <span class="bottleneck-warning">Bottleneck Node</span>
          </div>
        <% end %>
        <%= if @connections.blocks != [] do %>
          <div class="graph-detail-connections">
            <div class="graph-detail-conn-label">Blocks</div>
            <%= for {rel, peer} <- @connections.blocks do %>
              <div class="graph-detail-conn-row" phx-click="select_node" phx-value-id={peer["id"]}>
                <span class="graph-conn-rel"><%= rel %></span>
                <span class="graph-conn-title"><%= truncate_label(peer["title"]) %></span>
              </div>
            <% end %>
          </div>
        <% end %>
        <%= if @connections.blocked_by != [] do %>
          <div class="graph-detail-connections">
            <div class="graph-detail-conn-label">Blocked by</div>
            <%= for {rel, peer} <- @connections.blocked_by do %>
              <div class="graph-detail-conn-row" phx-click="select_node" phx-value-id={peer["id"]}>
                <span class="graph-conn-rel"><%= rel %></span>
                <span class="graph-conn-title"><%= truncate_label(peer["title"]) %></span>
              </div>
            <% end %>
          </div>
        <% end %>
        <div class="graph-detail-links">
          <a href={"/kanban?highlight=#{@node["id"]}"} class="graph-detail-link">View in Kanban</a>
          <a href={"/?feature=#{@node["id"]}"} class="graph-detail-link">Activity Feed</a>
        </div>
      </div>
    </div>
    """
  end
end
