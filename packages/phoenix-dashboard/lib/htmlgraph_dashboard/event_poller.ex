defmodule HtmlgraphDashboard.EventPoller do
  @moduledoc """
  Polls the HtmlGraph SQLite database for new events and broadcasts
  them via Phoenix PubSub for live updates.

  Polls ALL registered projects on each tick. Tracks last_timestamp
  per project using a map keyed by project_id.

  Checks every 1 second for new events since last poll.
  """
  use GenServer

  alias HtmlgraphDashboard.ProjectRegistry
  alias HtmlgraphDashboard.Repo

  @poll_interval_ms 1_000
  @topic "activity_feed"

  def start_link(opts) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc "Subscribe to live event updates."
  def subscribe do
    Phoenix.PubSub.subscribe(HtmlgraphDashboard.PubSub, @topic)
  end

  @impl true
  def init(_opts) do
    # last_timestamps: %{project_id => timestamp_string | nil}
    state = %{last_timestamps: %{}}
    schedule_poll()
    {:ok, state}
  end

  @impl true
  def handle_info(:poll, state) do
    new_state = poll_all_projects(state)
    schedule_poll()
    {:noreply, new_state}
  end

  defp schedule_poll do
    Process.send_after(self(), :poll, @poll_interval_ms)
  end

  defp poll_all_projects(state) do
    projects = ProjectRegistry.list_projects()

    updated_timestamps =
      Enum.reduce(projects, state.last_timestamps, fn project, acc ->
        last_ts = Map.get(acc, project.id)
        new_ts = poll_project(project, last_ts)
        Map.put(acc, project.id, new_ts)
      end)

    %{state | last_timestamps: updated_timestamps}
  end

  defp poll_project(project, nil) do
    # First poll for this project — record latest timestamp, don't broadcast history
    sql = "SELECT timestamp FROM agent_events ORDER BY timestamp DESC LIMIT 1"

    case Repo.query(sql, [], project.id) do
      {:ok, [[ts]]} -> ts
      _ -> nil
    end
  end

  defp poll_project(project, last_ts) do
    sql = """
    SELECT event_id, tool_name, event_type, timestamp, input_summary,
           output_summary, session_id, agent_id, parent_event_id,
           subagent_type, model, status, cost_tokens,
           execution_duration_seconds, feature_id, context
    FROM agent_events
    WHERE timestamp > ?
    ORDER BY timestamp ASC
    LIMIT 100
    """

    case Repo.query_maps(sql, [last_ts], project.id) do
      {:ok, []} ->
        last_ts

      {:ok, events} ->
        Enum.each(events, fn event ->
          Phoenix.PubSub.broadcast(
            HtmlgraphDashboard.PubSub,
            @topic,
            {:new_event, Map.put(event, "project_id", project.id)}
          )
        end)

        List.last(events)["timestamp"]

      {:error, _reason} ->
        last_ts
    end
  end
end
