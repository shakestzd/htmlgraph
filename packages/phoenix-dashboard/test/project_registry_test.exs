defmodule HtmlgraphDashboard.ProjectRegistryTest do
  use ExUnit.Case

  alias HtmlgraphDashboard.ProjectRegistry
  alias HtmlgraphDashboard.ProjectRegistry.Project

  test "scan_projects finds .htmlgraph directories" do
    tmp =
      System.tmp_dir!()
      |> Path.join("test_workspace_#{System.unique_integer([:positive])}")

    File.mkdir_p!(Path.join([tmp, "project-a", ".htmlgraph"]))
    File.write!(Path.join([tmp, "project-a", ".htmlgraph", "htmlgraph.db"]), "")
    File.mkdir_p!(Path.join([tmp, "project-b", ".htmlgraph"]))
    File.write!(Path.join([tmp, "project-b", ".htmlgraph", "htmlgraph.db"]), "")

    {:ok, pid} = ProjectRegistry.start_link(workspace_root: tmp)
    projects = ProjectRegistry.list_projects()

    assert length(projects) == 2
    assert Enum.any?(projects, &(&1.name == "project-a"))
    assert Enum.any?(projects, &(&1.name == "project-b"))

    # Projects are sorted by name
    names = Enum.map(projects, & &1.name)
    assert names == Enum.sort(names)

    GenServer.stop(pid)
    File.rm_rf!(tmp)
  end

  test "get_project returns the correct project struct" do
    tmp =
      System.tmp_dir!()
      |> Path.join("test_workspace_#{System.unique_integer([:positive])}")

    File.mkdir_p!(Path.join([tmp, "my-project", ".htmlgraph"]))
    File.write!(Path.join([tmp, "my-project", ".htmlgraph", "htmlgraph.db"]), "")

    {:ok, pid} = ProjectRegistry.start_link(workspace_root: tmp)
    project = ProjectRegistry.get_project("my-project")

    assert %Project{} = project
    assert project.id == "my-project"
    assert project.name == "my-project"
    assert String.ends_with?(project.db_path, ".htmlgraph/htmlgraph.db")

    GenServer.stop(pid)
    File.rm_rf!(tmp)
  end

  test "get_project returns nil for unknown project" do
    tmp =
      System.tmp_dir!()
      |> Path.join("test_empty_#{System.unique_integer([:positive])}")

    File.mkdir_p!(tmp)

    {:ok, pid} = ProjectRegistry.start_link(workspace_root: tmp)
    assert ProjectRegistry.get_project("nonexistent") == nil

    GenServer.stop(pid)
    File.rm_rf!(tmp)
  end

  test "list_projects returns empty list when no projects exist" do
    tmp =
      System.tmp_dir!()
      |> Path.join("test_empty2_#{System.unique_integer([:positive])}")

    File.mkdir_p!(tmp)

    {:ok, pid} = ProjectRegistry.start_link(workspace_root: tmp)
    assert ProjectRegistry.list_projects() == []

    GenServer.stop(pid)
    File.rm_rf!(tmp)
  end

  test "refresh re-scans the workspace" do
    tmp =
      System.tmp_dir!()
      |> Path.join("test_refresh_#{System.unique_integer([:positive])}")

    File.mkdir_p!(tmp)

    {:ok, pid} = ProjectRegistry.start_link(workspace_root: tmp)
    assert ProjectRegistry.list_projects() == []

    # Add a project after registry started
    File.mkdir_p!(Path.join([tmp, "new-project", ".htmlgraph"]))
    File.write!(Path.join([tmp, "new-project", ".htmlgraph", "htmlgraph.db"]), "")

    ProjectRegistry.refresh()
    # Allow the cast to be processed
    Process.sleep(50)

    projects = ProjectRegistry.list_projects()
    assert length(projects) == 1
    assert hd(projects).name == "new-project"

    GenServer.stop(pid)
    File.rm_rf!(tmp)
  end

  test "project id sanitises non-alphanumeric characters" do
    tmp =
      System.tmp_dir!()
      |> Path.join("test_sanitise_#{System.unique_integer([:positive])}")

    File.mkdir_p!(Path.join([tmp, "My.Project_Name", ".htmlgraph"]))
    File.write!(Path.join([tmp, "My.Project_Name", ".htmlgraph", "htmlgraph.db"]), "")

    {:ok, pid} = ProjectRegistry.start_link(workspace_root: tmp)
    [project] = ProjectRegistry.list_projects()

    # Uppercase letters, dots, and underscores are replaced with dashes
    assert project.id =~ ~r/^[a-z0-9-]+$/

    GenServer.stop(pid)
    File.rm_rf!(tmp)
  end
end
