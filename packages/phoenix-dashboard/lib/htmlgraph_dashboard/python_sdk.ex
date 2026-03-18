defmodule HtmlgraphDashboard.PythonSDK do
  @moduledoc """
  GenServer wrapping Pythonx calls to HtmlGraph Python SDK.
  Provides cached access to SDK operations for the Phoenix dashboard.
  """
  use GenServer

  # Client API

  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc "Get hierarchical activity feed"
  def list_activity_feed(opts \\ %{}) do
    GenServer.call(__MODULE__, {:list_activity_feed, opts}, 30_000)
  end

  @doc "Get work item detail by ID"
  def get_work_item(feature_id) do
    GenServer.call(__MODULE__, {:get_work_item, feature_id}, 10_000)
  end

  @doc "Get work item titles for a list of IDs"
  def get_work_item_titles(feature_ids) do
    GenServer.call(__MODULE__, {:get_work_item_titles, feature_ids}, 10_000)
  end

  @doc "Get graph stats: nodes, edges, cycles, critical path"
  def get_graph_stats do
    GenServer.call(__MODULE__, :get_graph_stats, 30_000)
  end

  @doc "Run arbitrary SDK Python code"
  def eval(code, globals \\ %{}) do
    GenServer.call(__MODULE__, {:eval, code, globals}, 30_000)
  end

  # Server callbacks

  @impl true
  def init(_opts) do
    # Get DB path from application config
    db_path = Application.get_env(:htmlgraph_dashboard, :db_path, "../../.htmlgraph/htmlgraph.db")

    # Resolve to absolute path
    abs_path =
      if String.starts_with?(db_path, "/") do
        db_path
      else
        Path.join(File.cwd!(), db_path)
      end

    graph_dir = abs_path |> Path.dirname() |> Path.dirname()

    {:ok, %{db_path: abs_path, graph_dir: graph_dir}}
  end

  @impl true
  def handle_call({:list_activity_feed, opts}, _from, state) do
    limit = Map.get(opts, :limit, 15)

    code = """
import json
from htmlgraph.db.schema import HtmlGraphDB

db = HtmlGraphDB(db_path.decode() if isinstance(db_path, bytes) else db_path)
db.connect()
db.create_tables()

cursor = db.conn.cursor()
cursor.execute(\"\"\"
    SELECT DISTINCT session_id,
           COUNT(CASE WHEN tool_name = 'UserQuery' THEN 1 END) as turn_count,
           COUNT(*) as event_count,
           MIN(timestamp) as first_event,
           MAX(timestamp) as last_event
    FROM agent_events
    GROUP BY session_id
    ORDER BY MAX(timestamp) DESC
    LIMIT ?
\"\"\", (limit,))

sessions = []
for row in cursor.fetchall():
    sessions.append({
        'session_id': row[0],
        'turn_count': row[1],
        'event_count': row[2],
        'first_event': row[3],
        'last_event': row[4],
    })

result = json.dumps(sessions)
result
"""

    {result, _} = Pythonx.eval(code, %{"db_path" => state.db_path, "limit" => limit})
    decoded = result |> Pythonx.decode() |> Jason.decode!()

    {:reply, {:ok, decoded}, state}
  rescue
    e -> {:reply, {:error, Exception.message(e)}, state}
  end

  @impl true
  def handle_call({:get_work_item, feature_id}, _from, state) do
    code = """
import json
import os
from htmlgraph import SDK
os.chdir(graph_dir)
sdk = SDK(agent='phoenix-dashboard')
feature_id = feature_id.decode() if isinstance(feature_id, bytes) else feature_id
try:
    f = sdk.features.get(feature_id)
    if f:
        result = json.dumps({
            'id': f.id,
            'title': f.title,
            'status': f.status,
            'priority': getattr(f, 'priority', 'medium'),
            'type': getattr(f, 'type', 'feature'),
            'steps': [{'description': s.description, 'completed': s.completed, 'step_id': getattr(s, 'step_id', None)} for s in (f.steps or [])],
        })
    else:
        result = 'null'
except Exception:
    result = 'null'
result
"""

    {result, _} = Pythonx.eval(code, %{"feature_id" => feature_id, "graph_dir" => state.graph_dir})
    decoded = result |> Pythonx.decode() |> Jason.decode!()

    {:reply, {:ok, decoded}, state}
  rescue
    e -> {:reply, {:error, Exception.message(e)}, state}
  end

  @impl true
  def handle_call(:get_graph_stats, _from, state) do
    code = """
import json
import os
from htmlgraph import SDK
os.chdir(graph_dir)
sdk = SDK(agent='phoenix-dashboard')
gm = sdk.graph
G = gm.G

stats = {
    'nodes': G.number_of_nodes(),
    'edges': G.number_of_edges(),
    'cycles': len(gm.find_cycles()),
    'critical_path': gm.critical_path(),
    'bottlenecks': gm.bottlenecks(top_n=5),
    'components': len(gm.connected_components()),
}
result = json.dumps(stats, default=str)
result
"""

    {result, _} = Pythonx.eval(code, %{"graph_dir" => state.graph_dir})
    decoded = result |> Pythonx.decode() |> Jason.decode!()

    {:reply, {:ok, decoded}, state}
  rescue
    e -> {:reply, {:error, Exception.message(e)}, state}
  end

  @impl true
  def handle_call({:get_work_item_titles, feature_ids}, _from, state) do
    code = """
import json
import os
from htmlgraph import SDK
os.chdir(graph_dir)
sdk = SDK(agent='phoenix-dashboard')
feature_ids = [f.decode() if isinstance(f, bytes) else f for f in feature_ids]
titles = {}
for fid in feature_ids:
    try:
        f = sdk.features.get(fid)
        if f:
            titles[fid] = {'title': f.title, 'type': getattr(f, 'type', 'feature')}
    except Exception:
        pass
result = json.dumps(titles)
result
"""

    {result, _} = Pythonx.eval(code, %{"feature_ids" => feature_ids, "graph_dir" => state.graph_dir})
    decoded = result |> Pythonx.decode() |> Jason.decode!()

    {:reply, {:ok, decoded}, state}
  rescue
    e -> {:reply, {:error, Exception.message(e)}, state}
  end

  @impl true
  def handle_call({:eval, code, globals}, _from, state) do
    {result, _} = Pythonx.eval(code, globals)
    {:reply, {:ok, Pythonx.decode(result)}, state}
  rescue
    e -> {:reply, {:error, Exception.message(e)}, state}
  end
end
