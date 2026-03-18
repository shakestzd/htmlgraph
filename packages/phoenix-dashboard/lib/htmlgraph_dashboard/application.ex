defmodule HtmlgraphDashboard.Application do
  @moduledoc false
  use Application

  @impl true
  def start(_type, _args) do
    # Initialize embedded Python with htmlgraph SDK
    Pythonx.uv_init("""
    [project]
    name = "htmlgraph-dashboard"
    version = "0.0.0"
    requires-python = ">=3.10"
    dependencies = ["htmlgraph>=0.33.80"]
    """)

    children = [
      HtmlgraphDashboard.PythonSDK,
      {Phoenix.PubSub, name: HtmlgraphDashboard.PubSub},
      HtmlgraphDashboardWeb.Endpoint,
      {HtmlgraphDashboard.EventPoller, []}
    ]

    opts = [strategy: :one_for_one, name: HtmlgraphDashboard.Supervisor]
    Supervisor.start_link(children, opts)
  end

  @impl true
  def config_change(changed, _new, removed) do
    HtmlgraphDashboardWeb.Endpoint.config_change(changed, removed)
    :ok
  end
end
