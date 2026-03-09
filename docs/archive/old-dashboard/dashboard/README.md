# HtmlGraph Dashboard - Spike Visualization

A modern dashboard for visualizing transition spikes and session boundaries in HtmlGraph.

## Features

### ✅ Completed

1. **Spike Display** - Shows all transition spikes from `.htmlgraph/spikes/`
2. **Timeline View** - Visualizes session → spike → session flow with boundaries
3. **Auto-Spike Filter** - Toggle to show/hide auto-generated spikes
4. **Auto-Spike Styling** - Muted appearance for auto-generated spikes

## Usage

### Opening the Dashboard

```bash
# From project root
cd dashboard
# Open index.html in a browser
```

Or use Python's built-in server:

```bash
cd dashboard
python -m http.server 8000
# Visit http://localhost:8000
```

### Controls

**View Modes:**
- **List View** - Card-based grid of all spikes
- **Timeline View** - Chronological flow showing session transitions

**Filters:**
- **Hide Auto-Generated Spikes** - Exclude spikes with `data-auto-generated="true"`
- **Show Only Active Sessions** - Filter to only `in-progress` spikes

### Stats Panel

Displays real-time statistics:
- **Total Spikes** - All spikes in the system
- **Auto-Generated** - Spikes created automatically during transitions
- **Manual Spikes** - User-created spikes
- **Active Transitions** - Spikes currently in-progress

## Architecture

### Files

- **index.html** - Main dashboard structure and layout
- **app.js** - Spike loading, filtering, and rendering logic
- **styles.css** - Visual styling with auto-spike differentiation

### Data Loading

The dashboard loads spike data from:
1. `.htmlgraph/spikes/*.html` - Individual spike files
2. `.htmlgraph/sessions/*.html` - Session files for timeline

**Auto-Generated Spike Detection:**
Spikes with `data-auto-generated="true"` attribute are:
- Styled with muted colors (grayed out)
- Marked with "Auto-Generated" badge
- Filterable via toggle

### Timeline Construction

Timeline items show:
```
Previous Session → Spike → Next Session
```

Built from spike metadata:
- `data-session-id` - Previous session
- `data-to-feature-id` - Next feature/session
- `data-spike-subtype` - Type of transition (e.g., "session-init")

## Styling

### Auto-Spike Differentiation

**List View:**
- Lighter background (`--auto-spike-bg`)
- Muted border (`--auto-spike-border`)
- Reduced opacity (0.85)
- Gray badge (`--auto-spike-badge`)

**Timeline View:**
- Muted connector lines
- Gray spike dot (vs. accent color for manual spikes)
- Reduced visual weight

### Theme Support

- **Light Theme** (default)
- **Dark Theme** (manual toggle or system preference)

## Sample Data

The dashboard includes sample spike data for demonstration. In production:

1. Create a manifest file: `.htmlgraph/spikes/manifest.json`
2. List all spike HTML files
3. The dashboard will auto-load from manifest

## Browser Compatibility

- Modern browsers (Chrome, Firefox, Safari, Edge)
- ES6+ JavaScript features
- CSS Grid and Flexbox layout
- No build step required

## Future Enhancements

- [ ] Real-time updates via WebSocket
- [ ] Export timeline as SVG/PNG
- [ ] Advanced filtering (by date, agent, priority)
- [ ] Spike detail modal
- [ ] Drag-and-drop timeline reordering
- [ ] Integration with feature dashboard

---

**Created:** 2025-12-26
**Feature:** feat-fc0193e4 (Dashboard Visualization)
**Agent:** Claude
