defmodule HtmlgraphDashboard.Repo do
  @moduledoc """
  Direct SQLite3 reader for the HtmlGraph database.

  Read-only access to the existing .htmlgraph/htmlgraph.db file.
  Uses exqlite for lightweight SQLite3 connectivity.
  """

  # Compile-time anchor: lib/htmlgraph_dashboard/ → ../../ → app root (packages/phoenix-dashboard/)
  # Then the default config path ../../.htmlgraph/htmlgraph.db goes up two more levels to the repo root.
  @app_root Path.expand("../../", __DIR__)

  @doc """
  Returns the configured database path, resolved relative to the Phoenix app root.

  Uses a compile-time anchor (__DIR__) instead of File.cwd!() so the path is
  stable regardless of what directory the beam process was started from.
  """
  def db_path do
    path = Application.get_env(:htmlgraph_dashboard, :db_path, "../../.htmlgraph/htmlgraph.db")

    if Path.type(path) == :relative do
      Path.join(@app_root, path)
      |> Path.expand()
    else
      path
    end
  end

  @doc """
  Execute a read-only query against the HtmlGraph database.
  Accepts an optional `project_id` to query a specific project's database.
  Returns {:ok, rows} or {:error, reason}.
  """
  def query(sql, params \\ [], project_id \\ nil) do
    path = resolve_db_path(project_id)

    case Exqlite.Sqlite3.open(path, [:readonly]) do
      {:ok, conn} ->
        try do
          execute_query(conn, sql, params)
        after
          Exqlite.Sqlite3.close(conn)
        end

      {:error, reason} ->
        {:error, {:open_failed, reason, path}}
    end
  end

  @doc """
  Execute a query and return rows as maps with column name keys.
  Accepts an optional `project_id` to query a specific project's database.
  """
  def query_maps(sql, params \\ [], project_id \\ nil) do
    path = resolve_db_path(project_id)

    case Exqlite.Sqlite3.open(path, [:readonly]) do
      {:ok, conn} ->
        try do
          case Exqlite.Sqlite3.prepare(conn, sql) do
            {:ok, stmt} ->
              bind_params(conn, stmt, params)
              rows = collect_rows(conn, stmt)
              columns = get_columns(conn, stmt)
              Exqlite.Sqlite3.release(conn, stmt)

              maps =
                Enum.map(rows, fn row ->
                  columns
                  |> Enum.zip(row)
                  |> Map.new()
                end)

              {:ok, maps}

            {:error, reason} ->
              {:error, reason}
          end
        after
          Exqlite.Sqlite3.close(conn)
        end

      {:error, reason} ->
        {:error, {:open_failed, reason, path}}
    end
  end

  defp resolve_db_path(nil), do: db_path()

  defp resolve_db_path(project_id) do
    case HtmlgraphDashboard.ProjectRegistry.get_project(project_id) do
      %{db_path: path} -> path
      nil -> db_path()
    end
  end

  defp execute_query(conn, sql, params) do
    case Exqlite.Sqlite3.prepare(conn, sql) do
      {:ok, stmt} ->
        bind_params(conn, stmt, params)
        rows = collect_rows(conn, stmt)
        Exqlite.Sqlite3.release(conn, stmt)
        {:ok, rows}

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp bind_params(_conn, _stmt, []), do: :ok

  defp bind_params(_conn, stmt, params) do
    Exqlite.Sqlite3.bind(stmt, params)
  end

  defp collect_rows(conn, stmt) do
    collect_rows(conn, stmt, [])
  end

  defp collect_rows(conn, stmt, acc) do
    case Exqlite.Sqlite3.step(conn, stmt) do
      {:row, row} -> collect_rows(conn, stmt, [row | acc])
      :done -> Enum.reverse(acc)
      {:error, _reason} -> Enum.reverse(acc)
    end
  end

  defp get_columns(conn, stmt) do
    case Exqlite.Sqlite3.columns(conn, stmt) do
      {:ok, column_names} -> column_names
      {:error, _reason} -> []
    end
  end
end
