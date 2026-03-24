// Minimal LiveView client — connects WebSocket for live updates.
// No build tools needed; served as static JS.

import {Socket} from "phoenix"
import {LiveSocket} from "phoenix_live_view"
import GraphControls from "./hooks/graph_controls"

let csrfToken = document.querySelector("meta[name='csrf-token']").getAttribute("content")

let hooks = {
  GraphControls,
}

let liveSocket = new LiveSocket("/live", Socket, {
  params: {_csrf_token: csrfToken},
  hooks,
})

liveSocket.connect()

window.liveSocket = liveSocket
