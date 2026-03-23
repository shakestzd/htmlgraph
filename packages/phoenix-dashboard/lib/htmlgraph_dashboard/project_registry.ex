defmodule HtmlgraphDashboard.ProjectRegistry do
  @moduledoc """
  Discovers and tracks HtmlGraph projects in the workspace.
  Scans for .htmlgraph/ directories and maintains a registry of available projects.
  """
  use GenServer

  defmodule Project do
    @moduledoc "Represents a discovered HtmlGraph project."
    defstruct [:id, :name, :path, :db_path]
  end

  # Client API

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc "Returns all discovered projects, sorted by name."
  def list_projects do
    GenServer.call(__MODULE__, :list_projects)
  end

  @doc "Returns the project with the given id, or nil if not found."
  def get_project(project_id) do
    GenServer.call(__MODULE__, {:get_project, project_id})
  end

  @doc "Re-scans the workspace and updates the project list."
  def refresh do
    GenServer.cast(__MODULE__, :refresh)
  end

  # Server callbacks

  @impl true
  def init(opts) do
    workspace_root = opts[:workspace_root] || discover_workspace_root()
    projects = scan_projects(workspace_root)
    {:ok, %{workspace_root: workspace_root, projects: projects}}
  end

  @impl true
  def handle_call(:list_projects, _from, state) do
    {:reply, state.projects, state}
  end

  @impl true
  def handle_call({:get_project, project_id}, _from, state) do
    project = Enum.find(state.projects, &(&1.id == project_id))
    {:reply, project, state}
  end

  @impl true
  def handle_cast(:refresh, state) do
    projects = scan_projects(state.workspace_root)
    {:noreply, %{state | projects: projects}}
  end

  # Private helpers

  # Compile-time anchor so path resolution is stable regardless of beam cwd.
  @app_root Path.expand("../../", __DIR__)

  defp discover_workspace_root do
    case System.get_env("HTMLGRAPH_WORKSPACE") do
      nil ->
        # Default: parent of the Phoenix app root (i.e. the repo workspace)
        Path.expand("../../..", @app_root)

      path ->
        path
    end
  end

  defp scan_projects(workspace_root) do
    workspace_root
    |> Path.join("*/.htmlgraph/htmlgraph.db")
    |> Path.wildcard()
    |> Enum.map(fn db_path ->
      project_dir = db_path |> Path.dirname() |> Path.dirname()
      project_name = Path.basename(project_dir)

      project_id =
        project_name
        |> String.downcase()
        |> String.replace(~r/[^a-z0-9-]/, "-")

      %Project{
        id: project_id,
        name: project_name,
        path: project_dir,
        db_path: db_path
      }
    end)
    |> Enum.sort_by(& &1.name)
  end
end
