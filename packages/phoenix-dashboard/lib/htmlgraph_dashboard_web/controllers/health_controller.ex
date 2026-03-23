defmodule HtmlgraphDashboardWeb.HealthController do
  use Phoenix.Controller, formats: [:html]

  def index(conn, _params) do
    conn
    |> put_status(200)
    |> text("ok")
  end
end
