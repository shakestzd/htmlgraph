defmodule HtmlgraphDashboard.PythonSDK do
  @moduledoc """
  GenServer wrapping Pythonx calls to HtmlGraph Python SDK.
  Provides cached access to SDK operations for the Phoenix dashboard.
  """
  use GenServer

  # Compile-time anchor: lib/htmlgraph_dashboard/ → ../../ → app root (packages/phoenix-dashboard/)
  # Stable regardless of what directory beam was launched from.
  @app_root Path.expand("../../", __DIR__)

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

  @doc "Get full dependency graph with nodes and edges for visualization"
  def get_dependency_graph do
    GenServer.call(__MODULE__, :get_dependency_graph, 30_000)
  end

  @doc "Get kanban board data: features grouped by status"
  def get_kanban_data do
    GenServer.call(__MODULE__, :get_kanban_data, 30_000)
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

    # Resolve to absolute path using compile-time anchor, not File.cwd!().
    # File.cwd!() is unreliable — the beam process cwd depends on how the server
    # was launched (e.g. from repo root vs packages/phoenix-dashboard/).
    abs_path =
      if Path.type(db_path) == :absolute do
        db_path
      else
        Path.join(@app_root, db_path) |> Path.expand()
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
import sqlite3
import os
import re

db_path_str = db_path.decode() if isinstance(db_path, bytes) else db_path
graph_dir_str = graph_dir.decode() if isinstance(graph_dir, bytes) else graph_dir
feature_id = feature_id.decode() if isinstance(feature_id, bytes) else feature_id

conn = sqlite3.connect(db_path_str)
conn.row_factory = sqlite3.Row
cursor = conn.cursor()
cursor.execute(
    "SELECT id, title, type, status, priority, track_id, steps_total, steps_completed FROM features WHERE id = ?",
    (feature_id,)
)
row = cursor.fetchone()
if row:
    result = json.dumps({
        'id': row['id'],
        'title': row['title'],
        'type': row['type'] or 'feature',
        'status': row['status'] or 'todo',
        'priority': row['priority'] or 'medium',
        'track_id': row['track_id'],
        'steps_total': row['steps_total'] or 0,
        'steps_completed': row['steps_completed'] or 0,
        'steps': [],
    })
else:
    # Fallback: read HTML file (covers spikes and bugs not in SQLite features table)
    result = 'null'
    for subdir in ['features', 'bugs', 'spikes']:
        html_path = os.path.join(graph_dir_str, '.htmlgraph', subdir, feature_id + '.html')
        if os.path.exists(html_path):
            try:
                content = open(html_path).read()
                title_match = re.search(r'<title>(.*?)</title>', content)
                status_match = re.search(r'data-status="([^"]*)"', content)
                priority_match = re.search(r'data-priority="([^"]*)"', content)
                item_type = 'spike' if feature_id.startswith('spk-') else 'bug' if feature_id.startswith('bug-') else 'feature'
                # Extract steps from HTML list items
                steps = []
                for li_match in re.finditer(r'<li[^>]*data-completed="([^"]*)"[^>]*>(.*?)</li>', content, re.DOTALL):
                    completed = li_match.group(1) == 'true'
                    desc = re.sub(r'<[^>]+>', '', li_match.group(2)).strip()
                    if desc:
                        steps.append({'description': desc, 'completed': completed})
                result = json.dumps({
                    'id': feature_id,
                    'title': title_match.group(1).strip() if title_match else feature_id,
                    'type': item_type,
                    'status': status_match.group(1) if status_match else 'todo',
                    'priority': priority_match.group(1) if priority_match else 'medium',
                    'steps': steps,
                    'steps_total': len(steps),
                    'steps_completed': sum(1 for s in steps if s['completed']),
                })
            except Exception:
                pass
            break
conn.close()
result
"""

    {result, _} =
      Pythonx.eval(code, %{
        "feature_id" => feature_id,
        "db_path" => state.db_path,
        "graph_dir" => state.graph_dir
      })

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
import sqlite3
import os
import re

db_path_str = db_path.decode() if isinstance(db_path, bytes) else db_path
graph_dir_str = graph_dir.decode() if isinstance(graph_dir, bytes) else graph_dir
feature_ids = [f.decode() if isinstance(f, bytes) else f for f in feature_ids]

conn = sqlite3.connect(db_path_str)
conn.row_factory = sqlite3.Row
cursor = conn.cursor()
titles = {}

for fid in feature_ids:
    cursor.execute("SELECT id, title, type FROM features WHERE id = ?", (fid,))
    row = cursor.fetchone()
    if row:
        titles[fid] = {'title': row['title'], 'type': row['type'] or 'feature'}
    else:
        # Fallback: try reading from HTML file
        for subdir in ['features', 'bugs', 'spikes']:
            html_path = os.path.join(graph_dir_str, '.htmlgraph', subdir, fid + '.html')
            if os.path.exists(html_path):
                try:
                    content = open(html_path).read()
                    title_match = re.search(r'<title>(.*?)</title>', content)
                    if title_match:
                        item_type = 'spike' if fid.startswith('spk-') else 'bug' if fid.startswith('bug-') else 'feature'
                        titles[fid] = {'title': title_match.group(1).strip(), 'type': item_type}
                except Exception:
                    pass
                break

conn.close()
result = json.dumps(titles)
result
"""

    {result, _} = Pythonx.eval(code, %{"feature_ids" => feature_ids, "db_path" => state.db_path, "graph_dir" => state.graph_dir})
    decoded = result |> Pythonx.decode() |> Jason.decode!()

    {:reply, {:ok, decoded}, state}
  rescue
    e -> {:reply, {:error, Exception.message(e)}, state}
  end

  @impl true
  def handle_call(:get_dependency_graph, _from, state) do
    code = """
import json
import math
import os
os.chdir(graph_dir.decode() if isinstance(graph_dir, bytes) else graph_dir)
from htmlgraph import SDK
from htmlgraph.graph.networkx_manager import GraphManager

sdk = SDK(agent='phoenix-dashboard')
gm = sdk.graph

# Get graph data
G = gm.G

# If graph is empty, skip all processing
if len(G.nodes()) > 0:
    # Separate connected nodes (have edges) from isolated nodes
    connected = set()
    for u, v in G.edges():
        connected.add(u)
        connected.add(v)

    isolated = [n for n in G.nodes() if n not in connected]

    # Always include ALL connected nodes (they form the actual dependency graph)
    # For isolated nodes, include only non-done ones (up to 30) — done isolates are clutter
    isolated_active = [n for n in isolated if G.nodes[n].get('status') != 'done'][:30]

    keep = list(connected) + isolated_active
    if keep:
        G = G.subgraph(keep).copy()
    else:
        # All nodes are isolated and done — show up to 30 non-done anyway
        non_done = [n for n, d in G.nodes(data=True) if d.get('status') != 'done'][:30]
        G = G.subgraph(non_done).copy() if non_done else G.subgraph([]).copy()

    # Cap at 100 nodes total, prioritising connected > in-progress > todo > blocked
    if len(G.nodes()) > 100:
        status_priority = {'in-progress': 0, 'todo': 1, 'blocked': 2}
        sorted_nodes = sorted(
            G.nodes(data=True),
            key=lambda x: (
                0 if x[0] in connected else 1,
                status_priority.get(x[1].get('status', 'todo'), 3),
            ),
        )
        keep = [n for n, _ in sorted_nodes[:100]]
        G = G.subgraph(keep).copy()

critical_path_ids = set(gm.critical_path())
bottleneck_ids = set(b['id'] for b in gm.bottlenecks(top_n=10))

# Grid layout: arrange nodes in a readable grid instead of a single column
nodes_list = list(G.nodes(data=True))
n = len(nodes_list)

# Sort nodes so in-progress first, then todo, then blocked, then done
status_order = {'in-progress': 0, 'todo': 1, 'blocked': 2, 'done': 3}
nodes_list.sort(key=lambda x: status_order.get(x[1].get('status', 'todo'), 1))

margin_x = 80
margin_y = 80
col_width = 280
row_height = 100

if n > 0:
    cols = max(3, int(math.ceil(math.sqrt(n))))
    rows = math.ceil(n / cols)
else:
    cols = 3
    rows = 1

color_map = {
    'in-progress': '#22c55e',
    'todo': '#3b82f6',
    'done': '#6b7280',
    'blocked': '#ef4444',
}

nodes = []
node_positions = {}
for idx, (node_id, data) in enumerate(nodes_list):
    row = idx // cols
    col = idx % cols
    x = margin_x + col * col_width
    y = margin_y + row * row_height
    node_positions[node_id] = {'x': x, 'y': y}
    status = data.get('status', 'todo')
    title = data.get('title', node_id)
    nodes.append({
        'id': node_id,
        'title': title,
        'status': status,
        'type': data.get('type', 'feature'),
        'priority': data.get('priority', 'medium'),
        'x': x,
        'y': y,
        'color': color_map.get(status, '#8b5cf6'),
        'is_critical': node_id in critical_path_ids,
        'is_bottleneck': node_id in bottleneck_ids,
        'depth': 0,
    })

edges = []
for u, v, edge_data in G.edges(data=True):
    if u in node_positions and v in node_positions:
        edges.append({
            'from': u,
            'to': v,
            'relationship': edge_data.get('relationship_type', 'relates_to'),
            'x1': node_positions[u]['x'],
            'y1': node_positions[u]['y'],
            'x2': node_positions[v]['x'],
            'y2': node_positions[v]['y'],
        })

# Compute SVG dimensions from grid
svg_width = margin_x * 2 + cols * col_width
svg_height = margin_y * 2 + rows * row_height

result = json.dumps({
    'nodes': nodes,
    'edges': edges,
    'critical_path': list(critical_path_ids),
    'viewbox_width': svg_width,
    'viewbox_height': svg_height,
}, default=str)
result
"""

    {result, _} = Pythonx.eval(code, %{"graph_dir" => state.graph_dir})
    decoded = result |> Pythonx.decode() |> Jason.decode!()

    {:reply, {:ok, decoded}, state}
  rescue
    e -> {:reply, {:error, Exception.message(e)}, state}
  end

  @impl true
  def handle_call(:get_kanban_data, _from, state) do
    code = """
import json
import sqlite3

db_path_str = db_path.decode() if isinstance(db_path, bytes) else db_path
conn = sqlite3.connect(db_path_str)
conn.row_factory = sqlite3.Row
cursor = conn.cursor()
features = []
for status in ['todo', 'in-progress', 'blocked', 'done']:
    cursor.execute(\"\"\"
        SELECT id, title, status, priority, type, steps_total, steps_completed
        FROM features
        WHERE status = ? AND type IN ('feature', 'bug')
        ORDER BY priority DESC, title ASC
        LIMIT 50
    \"\"\", (status,))
    for row in cursor.fetchall():
        features.append({
            'id': row['id'],
            'title': row['title'],
            'status': row['status'],
            'priority': row['priority'] or 'medium',
            'type': row['type'] or 'feature',
            'steps_total': row['steps_total'] or 0,
            'steps_completed': row['steps_completed'] or 0,
        })
conn.close()
json.dumps(features)
"""

    {result, _} = Pythonx.eval(code, %{"db_path" => state.db_path})
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
