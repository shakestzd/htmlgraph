"""Dagre-D3 dependency graph widget for CRISPI plan notebooks."""

import anywidget
import traitlets


class DependencyGraphWidget(anywidget.AnyWidget):
    """Interactive dependency graph rendered with dagre-d3.

    Nodes colored by status: done (green), approved (lime), todo (themed).
    Adapts to marimo's light/dark theme automatically via CSS variables.
    """

    nodes = traitlets.List([]).tag(sync=True)
    approved_ids = traitlets.List([]).tag(sync=True)

    _esm = """
    async function render({ model, el }) {
      const d3 = await import("https://cdn.jsdelivr.net/npm/d3@7/+esm");
      await new Promise((resolve, reject) => {
        if (window.dagreD3) return resolve();
        const s = document.createElement("script");
        s.src = "https://cdn.jsdelivr.net/npm/dagre-d3@0.6.4/dist/dagre-d3.min.js";
        s.onload = resolve; s.onerror = reject;
        document.head.appendChild(s);
      });

      // Detect marimo's theme reliably.
      function getTheme() {
        // Check marimo's data-color-mode attribute (most reliable).
        const root = document.documentElement;
        const mode = root.getAttribute("data-color-mode")
                  || root.getAttribute("data-theme")
                  || root.className;
        if (mode && (mode.includes("light") || mode === "light")) return "light";
        if (mode && (mode.includes("dark") || mode === "dark")) return "dark";
        // Check body background luminance as fallback.
        const bg = getComputedStyle(document.body).backgroundColor;
        const m = bg.match(/\\d+/g);
        if (m && m.length >= 3) {
          const lum = (parseInt(m[0]) * 299 + parseInt(m[1]) * 587 + parseInt(m[2]) * 114) / 1000;
          return lum > 128 ? "light" : "dark";
        }
        return window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark";
      }

      function themeColors(theme) {
        const isLight = theme === "light";
        return {
          containerBg: "transparent",
          containerBorder: "transparent",
          headerColor: isLight ? "#1a1a1a" : "#e0ded8",
          edgeColor: isLight ? "#b0b0a8" : "#555",
          done:     isLight ? { fill: "#dcfce7", stroke: "#16a34a", text: "#15803d" }
                            : { fill: "#16a34a", stroke: "#0d6e2a", text: "#ffffff" },
          approved: isLight ? { fill: "#f0fdf4", stroke: "#65a30d", text: "#3f6212" }
                            : { fill: "#1a2e00", stroke: "#cdff00", text: "#cdff00" },
          todo:     isLight ? { fill: "#f5f5f0", stroke: "#d0d0c8", text: "#555555" }
                            : { fill: "#2a2a2f", stroke: "#6b7280", text: "#e0ded8" },
        };
      }

      // Container.
      el.style.borderRadius = "6px";
      el.style.padding = "16px";
      el.style.overflowX = "auto";

      const svgEl = document.createElementNS("http://www.w3.org/2000/svg", "svg");
      svgEl.setAttribute("width", "100%");
      el.appendChild(svgEl);

      // Word-wrap: split a label into lines of maxLen chars at word boundaries.
      function wrapLabel(text, maxLen) {
        if (text.length <= maxLen) return text;
        const words = text.split(/\\s+/);
        const lines = [];
        let line = "";
        for (const w of words) {
          if (line && (line.length + 1 + w.length) > maxLen) {
            lines.push(line);
            line = w;
          } else {
            line = line ? line + " " + w : w;
          }
        }
        if (line) lines.push(line);
        return lines.join("\\n");
      }

      function draw() {
        const nodes = model.get("nodes") || [];
        const approvedIds = new Set(model.get("approved_ids") || []);
        const theme = getTheme();
        const tc = themeColors(theme);

        // Apply theme to container.
        el.style.background = tc.containerBg;
        el.style.border = "1px solid " + tc.containerBorder;

        if (nodes.length === 0) { svgEl.innerHTML = ""; return; }

        const g = new dagreD3.graphlib.Graph()
          .setGraph({ rankdir: "TB", marginx: 24, marginy: 24, ranksep: 52, nodesep: 28 })
          .setDefaultEdgeLabel(() => ({}));

        nodes.forEach((n) => {
          let status = n.status || "todo";
          if (status !== "done" && approvedIds.has(n.id)) status = "approved";
          const label = wrapLabel(n.name, 22);
          g.setNode(String(n.num), {
            label: label, labelStyle: "font-size:11px;font-family:system-ui,sans-serif",
            rx: 6, ry: 6, paddingX: 14, paddingY: 10, status: status,
          });
        });

        nodes.forEach((n) => {
          if (!n.deps) return;
          n.deps.split(",").map(s => s.trim()).filter(Boolean)
            .forEach((dep) => g.setEdge(dep, String(n.num)));
        });

        svgEl.innerHTML = "";
        const svg = d3.select(svgEl);
        const inner = svg.append("g");
        new dagreD3.render()(inner, g);

        // Style nodes.
        inner.selectAll("g.node").each(function (id) {
          const nd = g.node(id);
          const c = tc[nd.status] || tc.todo;
          d3.select(this).select("rect")
            .style("fill", c.fill).style("stroke", c.stroke).style("stroke-width", "2px");
          d3.select(this).selectAll("tspan,text")
            .style("fill", c.text);
        });

        // Style edges.
        inner.selectAll("g.edgePath path")
          .style("stroke", tc.edgeColor).style("stroke-dasharray", "4 3").style("fill", "none");
        inner.selectAll("g.edgePath marker path").style("fill", tc.edgeColor);

        // Size SVG.
        const pad = 24;
        inner.attr("transform", `translate(${pad},${pad})`);
        svgEl.setAttribute("width", (g.graph().width || 200) + pad * 2);
        svgEl.setAttribute("height", (g.graph().height || 100) + pad * 2);
      }

      draw();
      model.on("change:nodes", draw);
      model.on("change:approved_ids", draw);

      // Re-draw on theme change.
      // Re-draw on any theme change (marimo uses data-color-mode, data-theme, or class).
      new MutationObserver(draw).observe(document.documentElement,
        { attributes: true, attributeFilter: ["class", "data-theme", "data-color-mode", "style"] });
      new MutationObserver(draw).observe(document.body,
        { attributes: true, attributeFilter: ["class", "style"] });
    }
    export default { render };
    """

    _css = ":host { display: block; width: 100%; }"
