defmodule HtmlgraphDashboardWeb.ProjectsLive do
  @moduledoc """
  Lists all HtmlGraph projects discovered in the workspace.

  Projects are found by scanning for .htmlgraph/htmlgraph.db files under the
  configured HTMLGRAPH_WORKSPACE directory (or the parent workspace root by
  default). Each project card links to the Activity, Kanban, and Costs views
  for that project via the ?project=<id> query param.
  """
  use HtmlgraphDashboardWeb, :live_view

  alias HtmlgraphDashboard.ProjectRegistry

  @impl true
  def mount(_params, _session, socket) do
    projects = ProjectRegistry.list_projects()

    socket =
      socket
      |> assign(:active_tab, :projects)
      |> assign(:projects, projects)

    {:ok, socket}
  end

  @impl true
  def handle_event("refresh_projects", _params, socket) do
    ProjectRegistry.refresh()
    # Give the GenServer a moment to finish scanning, then re-read
    Process.sleep(100)
    projects = ProjectRegistry.list_projects()
    {:noreply, assign(socket, :projects, projects)}
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

    <div style="padding: 1.5rem 1.5rem 0.5rem; display: flex; justify-content: space-between; align-items: center;">
      <h2 style="margin: 0; font-size: 1.1rem; color: var(--text-primary, #e2e8f0);">
        Discovered Projects
      </h2>
      <button phx-click="refresh_projects" class="graph-refresh-btn">
        Refresh
      </button>
    </div>

    <div style="padding: 0.5rem 1.5rem 1.5rem;">
      <%= if @projects == [] do %>
        <div class="empty-state">
          <h2>No projects discovered</h2>
          <p>Set <code>HTMLGRAPH_WORKSPACE</code> to your workspace root directory,</p>
          <p>or projects will be auto-discovered from the parent directory.</p>
          <p style="margin-top: 1rem; font-size: 0.85rem; color: #64748b;">
            Each project must have a <code>.htmlgraph/htmlgraph.db</code> file.
          </p>
        </div>
      <% else %>
        <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 1rem;">
          <%= for project <- @projects do %>
            <div style="background: var(--surface, #1C1C20); border: 1px solid var(--border, #333); padding: 1.5rem; border-radius: 4px;">
              <h3 style="margin: 0 0 0.4rem; color: #CDFF00; font-size: 1rem;">
                <%= project.name %>
              </h3>
              <p style="color: #64748b; font-size: 0.75rem; margin: 0 0 1rem; word-break: break-all;">
                <%= project.path %>
              </p>
              <div style="display: flex; gap: 1rem; flex-wrap: wrap;">
                <a href={"/?project=#{project.id}"} class="nav-tab" style="font-size: 0.8rem; padding: 4px 10px;">
                  Activity
                </a>
                <a href={"/kanban?project=#{project.id}"} class="nav-tab" style="font-size: 0.8rem; padding: 4px 10px;">
                  Kanban
                </a>
                <a href={"/costs?project=#{project.id}"} class="nav-tab" style="font-size: 0.8rem; padding: 4px 10px;">
                  Costs
                </a>
              </div>
            </div>
          <% end %>
        </div>
      <% end %>
    </div>
    """
  end
end
