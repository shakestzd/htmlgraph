defmodule HtmlgraphDashboardWeb.ProjectComponents do
  @moduledoc """
  Reusable Phoenix function components for multi-project UI.

  Provides: identity_stripe, project_switcher, project_card,
  stat_badge, sparkline, nav_project_context.

  All components use CSS classes from components/multi-project.css.
  """
  use Phoenix.Component

  @project_colors ~w(#58a6ff #3fb950 #d29922 #bc8cff #f778ba #39d2c0 #f85149 #db6d28)

  @doc "Returns a deterministic color for a project based on its id."
  def project_color(project_id) when is_binary(project_id) do
    index = :erlang.phash2(project_id, length(@project_colors))
    Enum.at(@project_colors, index)
  end

  def project_color(_), do: hd(@project_colors)

  # ------------------------------------------------------------------
  # Identity Stripe
  # ------------------------------------------------------------------

  @doc "3px colored bar fixed at the top of the viewport."
  attr :project, :any, default: nil

  def identity_stripe(assigns) do
    color = if assigns.project, do: project_color(assigns.project.id), else: hd(@project_colors)
    assigns = assign(assigns, :color, color)

    ~H"""
    <div class="identity-stripe" style={"--project-active: #{@color}"}></div>
    """
  end

  # ------------------------------------------------------------------
  # Project Switcher
  # ------------------------------------------------------------------

  @doc "Compact project switcher in the nav bar with picker overlay."
  attr :current_project, :map, required: true
  attr :projects, :list, required: true
  attr :picker_open, :boolean, default: false

  def project_switcher(assigns) do
    color = project_color(assigns.current_project.id)
    assigns = assign(assigns, :color, color)

    ~H"""
    <div class="project-switcher">
      <button
        class="project-switcher-btn"
        phx-click="toggle_project_picker"
        style={"--project-active: #{@color}"}
      >
        <span class="project-color-dot" style={"background: #{@color}"}></span>
        <span><%= @current_project.name %></span>
        <span class={["project-switcher-chevron", @picker_open && "open"]}>&#9660;</span>
      </button>

      <%= if @picker_open do %>
        <.project_picker
          projects={@projects}
          current_project={@current_project}
        />
      <% end %>
    </div>
    """
  end

  # ------------------------------------------------------------------
  # Project Picker (overlay)
  # ------------------------------------------------------------------

  attr :projects, :list, required: true
  attr :current_project, :map, required: true

  defp project_picker(assigns) do
    ~H"""
    <div class="project-picker-backdrop" phx-click="close_project_picker"></div>
    <div class="project-picker" phx-click-away="close_project_picker">
      <div class="project-picker-header">Select Project</div>
      <%= for project <- @projects do %>
        <% color = project_color(project.id) %>
        <div
          class={["project-picker-row", project.id == @current_project.id && "active"]}
          style={"--project-active: #{color}"}
          phx-click="select_project"
          phx-value-project_id={project.id}
        >
          <span class="project-color-dot" style={"background: #{color}"}></span>
          <div class="project-picker-info">
            <div class="project-picker-name"><%= project.name %></div>
            <div class="project-picker-path"><%= project.path %></div>
          </div>
        </div>
      <% end %>
    </div>
    """
  end

  # ------------------------------------------------------------------
  # Project Card
  # ------------------------------------------------------------------

  @doc "Project overview card for the /projects page."
  attr :project, :map, required: true
  attr :stats, :map, default: %{events: 0, features: 0, sessions: 0, cost: "$0.00"}
  attr :active_sessions, :integer, default: 0
  attr :sparkline_data, :list, default: []

  def project_card(assigns) do
    color = project_color(assigns.project.id)
    assigns = assign(assigns, :color, color)

    ~H"""
    <div class="project-card" style={"--project-active: #{@color}; border-top-color: #{@color}"}>
      <div class="card-body">
        <div class="card-header">
          <span class="project-color-dot" style={"background: #{@color}"}></span>
          <span class="card-project-name"><%= @project.name %></span>
          <%= if @active_sessions > 0 do %>
            <span class="card-pulse" title={"#{@active_sessions} active session(s)"}></span>
          <% end %>
        </div>

        <div class="card-project-path"><%= @project.path %></div>

        <div class="card-stats">
          <div class="stat-box">
            <span class="stat-box-label">Events</span>
            <span class="stat-box-value"><%= format_number(@stats.events) %></span>
          </div>
          <div class="stat-box">
            <span class="stat-box-label">Features</span>
            <span class="stat-box-value"><%= @stats.features %></span>
          </div>
          <div class="stat-box">
            <span class="stat-box-label">Sessions</span>
            <span class="stat-box-value"><%= @stats.sessions %></span>
          </div>
          <div class="stat-box">
            <span class="stat-box-label">Est. Cost</span>
            <span class="stat-box-value"><%= @stats.cost %></span>
          </div>
        </div>

        <%= if @sparkline_data != [] do %>
          <div class="card-sparkline">
            <.sparkline data={@sparkline_data} color={@color} height={48} width={280} />
          </div>
        <% end %>

        <div class="card-actions">
          <a href={"/?project=#{@project.id}"} class="card-action-link">Activity</a>
          <a href={"/graph?project=#{@project.id}"} class="card-action-link">Graph</a>
          <a href={"/kanban?project=#{@project.id}"} class="card-action-link">Kanban</a>
          <a href={"/costs?project=#{@project.id}"} class="card-action-link">Costs</a>
        </div>
      </div>
    </div>
    """
  end

  # ------------------------------------------------------------------
  # Stat Badge
  # ------------------------------------------------------------------

  @doc "Compact inline stat with optional icon."
  attr :label, :string, required: true
  attr :value, :string, required: true
  attr :icon, :string, default: nil

  def stat_badge(assigns) do
    ~H"""
    <span class="stat-badge">
      <%= if @icon do %>
        <span class="stat-badge-icon"><%= @icon %></span>
      <% end %>
      <span><%= @label %>:</span>
      <strong><%= @value %></strong>
    </span>
    """
  end

  # ------------------------------------------------------------------
  # Sparkline (SVG)
  # ------------------------------------------------------------------

  @doc "SVG sparkline from a list of numeric data points."
  attr :data, :list, required: true
  attr :color, :string, default: "#58a6ff"
  attr :height, :integer, default: 48
  attr :width, :integer, default: 200

  def sparkline(assigns) do
    points = build_sparkline_points(assigns.data, assigns.width, assigns.height)
    fill_points = build_sparkline_fill(points, assigns.width, assigns.height)
    gradient_id = "spark-grad-#{System.unique_integer([:positive])}"

    assigns =
      assigns
      |> assign(:points, points)
      |> assign(:fill_points, fill_points)
      |> assign(:gradient_id, gradient_id)

    ~H"""
    <svg
      viewBox={"0 0 #{@width} #{@height}"}
      preserveAspectRatio="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <defs>
        <linearGradient id={@gradient_id} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color={@color} stop-opacity="0.3" />
          <stop offset="100%" stop-color={@color} stop-opacity="0.0" />
        </linearGradient>
      </defs>
      <!-- Gradient fill below the line -->
      <polygon
        points={@fill_points}
        fill={"url(##{@gradient_id})"}
      />
      <!-- The sparkline itself -->
      <polyline
        points={@points}
        fill="none"
        stroke={@color}
        stroke-width="1.5"
        stroke-linejoin="round"
        stroke-linecap="round"
      />
    </svg>
    """
  end

  # ------------------------------------------------------------------
  # Nav Project Context
  # ------------------------------------------------------------------

  @doc "Shows project color dot and name in the nav, always visible."
  attr :project, :map, required: true

  def nav_project_context(assigns) do
    color = project_color(assigns.project.id)
    assigns = assign(assigns, :color, color)

    ~H"""
    <div class="nav-project-context">
      <span class="project-color-dot" style={"background: #{@color}"}></span>
      <span><%= @project.name %></span>
    </div>
    """
  end

  # ------------------------------------------------------------------
  # Private Helpers
  # ------------------------------------------------------------------

  defp build_sparkline_points([], _width, _height), do: "0,0"

  defp build_sparkline_points(data, width, height) do
    max_val = Enum.max(data) |> max(1)
    count = length(data)
    step = if count > 1, do: width / (count - 1), else: 0.0
    padding = 2

    data
    |> Enum.with_index()
    |> Enum.map(fn {val, i} ->
      x = Float.round(i * step, 1)
      y = Float.round(height - padding - val / max_val * (height - padding * 2), 1)
      "#{x},#{y}"
    end)
    |> Enum.join(" ")
  end

  defp build_sparkline_fill(points_str, width, height) do
    "0,#{height} #{points_str} #{width},#{height}"
  end

  defp format_number(n) when is_integer(n) and n >= 1000 do
    "#{Float.round(n / 1000, 1)}k"
  end

  defp format_number(n), do: "#{n}"
end
