// GraphControls — zoom/pan hook for the dependency graph SVG.
//
// Attach with phx-hook="GraphControls" on the SVG element.
// Wraps SVG children in a <g id="graph-transform-group"> and applies
// CSS transform for zoom/pan state. Vanilla JS only, no libraries.

const MIN_ZOOM = 0.3;
const MAX_ZOOM = 3.0;
const ZOOM_STEP = 0.15;
const WHEEL_SENSITIVITY = 0.001;

const GraphControls = {
  mounted() {
    this.scale = 1.0;
    this.panX = 0;
    this.panY = 0;
    this.isPanning = false;
    this.lastMouseX = 0;
    this.lastMouseY = 0;

    this._wrapChildren();
    this._buildControls();
    this._bindEvents();
  },

  // LiveView patches the SVG DOM on every re-render (new data, refresh, etc.).
  // After the patch, the new <path> and <g> elements are direct SVG children
  // again — outside graph-transform-group. Re-wrap so they get the transform.
  updated() {
    this._rewrapAfterPatch();
    this._applyTransform();
  },

  destroyed() {
    this._unbindEvents();
    const controls = document.getElementById("graph-controls-panel");
    if (controls) controls.remove();
  },

  // Wrap existing SVG children in a transform group so we can pan/zoom
  // without altering the viewBox (which LiveView may update on re-render).
  // <defs> must remain a direct SVG child (SVG spec + marker/filter refs).
  _wrapChildren() {
    const svg = this.el;
    const existing = svg.getElementById("graph-transform-group");
    if (existing) {
      this.group = existing;
      return;
    }

    this.group = document.createElementNS("http://www.w3.org/2000/svg", "g");
    this.group.setAttribute("id", "graph-transform-group");

    // Move non-defs children into the group; keep <defs> as direct SVG child.
    const toMove = Array.from(svg.childNodes).filter(
      (n) => n.nodeName !== "defs"
    );
    toMove.forEach((n) => this.group.appendChild(n));
    svg.appendChild(this.group);
    this._applyTransform();
  },

  // Called from updated(): move any new SVG children (injected by LiveView's
  // DOM patch) that landed outside the transform group back into it.
  _rewrapAfterPatch() {
    const svg = this.el;
    let group = svg.getElementById("graph-transform-group");

    if (!group) {
      // Group was wiped by a full patch — recreate it.
      this.group = document.createElementNS("http://www.w3.org/2000/svg", "g");
      this.group.setAttribute("id", "graph-transform-group");
      group = this.group;
      svg.appendChild(group);
    } else {
      this.group = group;
    }

    // Move any stray children (not <defs>, not the group itself) into group.
    const stray = Array.from(svg.childNodes).filter(
      (n) => n !== group && n.nodeName !== "defs"
    );
    stray.forEach((n) => group.appendChild(n));
  },

  _applyTransform() {
    this.group.setAttribute(
      "transform",
      `translate(${this.panX}, ${this.panY}) scale(${this.scale})`
    );
  },

  _buildControls() {
    // Remove existing panel if LiveView re-mounts the hook
    const old = document.getElementById("graph-controls-panel");
    if (old) old.remove();

    const panel = document.createElement("div");
    panel.id = "graph-controls-panel";
    panel.className = "graph-controls";
    panel.innerHTML = `
      <button class="graph-ctrl-btn" data-action="zoom-in" title="Zoom in">+</button>
      <button class="graph-ctrl-btn" data-action="zoom-out" title="Zoom out">&minus;</button>
      <button class="graph-ctrl-btn" data-action="fit" title="Fit to view">&#9635;</button>
    `;

    // Insert as sibling of the SVG's parent container
    const container = this.el.closest(".graph-svg-wrapper") || this.el.parentElement;
    if (container) {
      container.style.position = "relative";
      container.appendChild(panel);
    }

    panel.addEventListener("click", (e) => {
      const btn = e.target.closest("[data-action]");
      if (!btn) return;
      const action = btn.dataset.action;
      if (action === "zoom-in") this._zoom(ZOOM_STEP, null, null);
      else if (action === "zoom-out") this._zoom(-ZOOM_STEP, null, null);
      else if (action === "fit") this._fitToView();
    });
  },

  _zoom(delta, clientX, clientY) {
    const svg = this.el;
    const rect = svg.getBoundingClientRect();

    // Zoom toward cursor position; default to SVG center
    const cx = clientX != null ? clientX - rect.left : rect.width / 2;
    const cy = clientY != null ? clientY - rect.top : rect.height / 2;

    const newScale = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, this.scale + delta));
    if (newScale === this.scale) return;

    // Adjust pan so the point under the cursor stays fixed
    const ratio = newScale / this.scale;
    this.panX = cx - ratio * (cx - this.panX);
    this.panY = cy - ratio * (cy - this.panY);
    this.scale = newScale;

    this._applyTransform();
  },

  _fitToView() {
    this.scale = 1.0;
    this.panX = 0;
    this.panY = 0;
    this._applyTransform();
  },

  _onWheel(e) {
    e.preventDefault();
    // Normalize deltaY across browsers (deltaMode: 0=pixels, 1=lines, 2=pages)
    let dy = e.deltaY;
    if (e.deltaMode === 1) dy *= 30;
    if (e.deltaMode === 2) dy *= 300;
    const delta = Math.max(-0.15, Math.min(0.15, -dy * 0.002));
    this._zoom(delta, e.clientX, e.clientY);
  },

  _onMouseDown(e) {
    // Only pan on primary button; ignore clicks on graph nodes (phx-click)
    if (e.button !== 0) return;
    if (e.target.closest(".graph-node")) return;
    this.isPanning = true;
    this.lastMouseX = e.clientX;
    this.lastMouseY = e.clientY;
    this.el.style.cursor = "grabbing";
    e.preventDefault();
  },

  _onMouseMove(e) {
    if (!this.isPanning) return;
    const dx = e.clientX - this.lastMouseX;
    const dy = e.clientY - this.lastMouseY;
    this.lastMouseX = e.clientX;
    this.lastMouseY = e.clientY;
    this.panX += dx;
    this.panY += dy;
    this._applyTransform();
  },

  _onMouseUp() {
    if (!this.isPanning) return;
    this.isPanning = false;
    this.el.style.cursor = "grab";
  },

  _bindEvents() {
    this._wheelHandler = this._onWheel.bind(this);
    this._mouseDownHandler = this._onMouseDown.bind(this);
    this._mouseMoveHandler = this._onMouseMove.bind(this);
    this._mouseUpHandler = this._onMouseUp.bind(this);

    this.el.addEventListener("wheel", this._wheelHandler, { passive: false });
    this.el.addEventListener("mousedown", this._mouseDownHandler);
    window.addEventListener("mousemove", this._mouseMoveHandler);
    window.addEventListener("mouseup", this._mouseUpHandler);

    this.el.style.cursor = "grab";
  },

  _unbindEvents() {
    this.el.removeEventListener("wheel", this._wheelHandler);
    this.el.removeEventListener("mousedown", this._mouseDownHandler);
    window.removeEventListener("mousemove", this._mouseMoveHandler);
    window.removeEventListener("mouseup", this._mouseUpHandler);
  },
};

export default GraphControls;
