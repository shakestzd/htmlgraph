defmodule HtmlgraphDashboardWeb.KanbanLive do
  @moduledoc """
  Kanban board view showing work items organized by status columns.

  Columns: Todo | In Progress | Blocked | Done
  Cards display title, priority badge, type badge, and step progress bar.
  """
  use HtmlgraphDashboardWeb, :live_view

  alias HtmlgraphDashboard.PythonSDK

  @columns [
    %{key: "todo", label: "Todo", color: "#94a3b8"},
    %{key: "in-progress", label: "In Progress", color: "#60a5fa"},
    %{key: "blocked", label: "Blocked", color: "#f87171"},
    %{key: "done", label: "Done", color: "#34d399"}
  ]

  @impl true
  def mount(_params, _session, socket) do
    items = load_kanban_data()

    socket =
      socket
      |> assign(:active_tab, :kanban)
      |> assign(:items, items)
      |> assign(:columns, @columns)
      |> assign(:selected_card, nil)

    {:ok, socket}
  end

  @impl true
  def handle_event("select_card", %{"id" => card_id}, socket) do
    card = Enum.find(socket.assigns.items, fn i -> i["id"] == card_id end)

    detail =
      if card do
        load_work_item_detail(card_id) || card
      else
        nil
      end

    {:noreply, assign(socket, :selected_card, detail)}
  end

  def handle_event("close_detail", _params, socket) do
    {:noreply, assign(socket, :selected_card, nil)}
  end

  def handle_event("refresh_kanban", _params, socket) do
    items = load_kanban_data()

    socket =
      socket
      |> assign(:items, items)
      |> assign(:selected_card, nil)

    {:noreply, socket}
  end

  defp load_kanban_data do
    try do
      case PythonSDK.get_kanban_data() do
        {:ok, data} when is_list(data) -> data
        _ -> []
      end
    rescue
      _ -> []
    catch
      :exit, _ -> []
    end
  end

  defp load_work_item_detail(id) do
    try do
      case PythonSDK.get_work_item(id) do
        {:ok, item} when is_map(item) -> item
        _ -> nil
      end
    rescue
      _ -> nil
    catch
      :exit, _ -> nil
    end
  end

  defp items_for_column(items, status) do
    Enum.filter(items, fn i -> i["status"] == status end)
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
              <%= for item <- items_for_column(@items, col.key) do %>
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
