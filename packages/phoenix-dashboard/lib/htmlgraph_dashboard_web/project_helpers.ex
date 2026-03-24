defmodule HtmlgraphDashboardWeb.ProjectHelpers do
  @moduledoc "Shared helpers for project-scoped operations across LiveViews."

  alias HtmlgraphDashboard.ProjectRegistry

  @doc "Build PythonSDK opts map from a project struct. Returns %{} for nil."
  def project_graph_opts(nil), do: %{}

  def project_graph_opts(project) do
    case ProjectRegistry.get_project(project.id) do
      %{db_path: db_path} ->
        graph_dir = db_path |> Path.dirname() |> Path.dirname()
        %{db_path: db_path, graph_dir: graph_dir}

      nil ->
        %{}
    end
  end
end
