# Wipnote Dashboard Redesign - Design Document

## Aesthetic Direction: Maximalist with Technical Sophistication

A bold, information-dense dashboard that celebrates code and data visualization with playful, sophisticated interactions.

### Design Philosophy
- **Information Density** - Every pixel conveys meaning
- **Visual Hierarchy** - Clear path through data complexity
- **Technical Authenticity** - Monospace for data, display fonts for structure
- **Playful Interactions** - Smooth animations, delightful micro-interactions
- **Dark + Lime** - High contrast, energy, developer aesthetic

---

## Color Palette

### Primary Colors
- **Background**: `#0a0a0a` (Near black, slightly warm)
- **Accent/Primary**: `#CDFF00` (Lime, vibrant and energetic)
- **Secondary**: `#6C5CE7` (Purple, sophisticated depth)

### Agent Colors (by model)
- **Claude**: `#8B5CF6` (Purple)
- **Gemini**: `#3B82F6` (Blue)
- **Copilot**: `#6B7280` (Gray)
- **OpenAI**: `#10B981` (Green)

### Status Colors
- **Success**: `#10B981` (Green)
- **In Progress**: `#3B82F6` (Blue)
- **Blocked**: `#EF4444` (Red)
- **Todo**: `#6B7280` (Gray)
- **Done**: `#8B5CF6` (Purple)

### Semantic Colors
- **Text Primary**: `#FFFFFF` (Pure white for contrast)
- **Text Secondary**: `#A3A3A3` (Subtle gray)
- **Border**: `#2A2A2A` (Subtle dividers)
- **Hover**: `#1A1A1A` (Slight elevation)

---

## Typography

### Font Stack
```css
/* Display (headers, titles) */
font-family: 'JetBrains Mono', 'IBM Plex Mono', monospace;
font-weight: 700;
letter-spacing: 0.05em;

/* Body (descriptions, content) */
font-family: 'Courier New', 'Courier', monospace;
font-weight: 400;

/* Code (data, technical info) */
font-family: 'JetBrains Mono', monospace;
font-size: 0.875rem;
```

### Type Scale
- **H1** (Page Title): 2.5rem, bold, display font
- **H2** (Section Title): 1.75rem, bold, display font
- **H3** (Card Title): 1.25rem, bold, display font
- **Body**: 1rem, regular, body font
- **Small**: 0.875rem, regular, gray text
- **Code**: 0.75rem, monospace font

---

## Component Design

### 1. Header (Enhanced)
- **Height**: 80px
- **Background**: Gradient `#0a0a0a` → `#1a1a1a`
- **Border**: Thin lime bottom border with glow
- **Content**:
  - Logo with animated icon
  - Live stats (Events, Agents, Sessions) with pulse animation
  - Status indicator (WS connected/disconnected)

### 2. Navigation Tabs
- **Style**: Underline tabs with lime accent
- **Interaction**: Smooth color transition
- **Active State**: Lime underline + text color
- **Hover**: Lime text preview + background tint
- **Icons**: Geometric symbols (not emoji)

### 3. Activity Timeline (Hierarchical)
- **Layout**: Vertical timeline with time flowing top→bottom
- **Parent Events**: Bold nodes on timeline with:
  - Event type icon (geometric SVG)
  - Agent badge (color-coded by model)
  - Timestamp and duration
  - Status indicator
  - Expandable detail panel

- **Child Events**: Indented branches with:
  - SVG connector lines (diagonal, smooth curves)
  - Right-aligned indentation
  - Lighter styling to show hierarchy
  - Expandable on click

- **Visual Elements**:
  - Timeline axis: Thin lime vertical line
  - Node styling: Bordered circles with agent color fill
  - Connectors: SVG paths with agent color stroke
  - Metrics inline: Token cost, duration, model
  - On hover: Highlight entire chain (parent + children)

### 4. Orchestration Graph
- **Layout**: DAG visualization (left→right flow)
- **Nodes**:
  - Size by token cost (log scale)
  - Color by model type
  - Label with agent name + cost
  - Hover: Show task details

- **Edges**:
  - Directed arrows showing delegation
  - Label with task type
  - Color by source agent
  - Animate on hover: Highlight entire path

- **Interactions**:
  - Click node → show activity details
  - Filter by agent/model/cost range
  - Timeline overlay showing temporal sequence
  - Zoom/pan controls

### 5. Smart Kanban Board
- **Default State**: To Do + In Progress columns visible
- **Column Management**:
  - Collapsed columns show header + item count
  - Click to expand (auto-collapses another column)
  - Collapse priority: Done > Blocked > oldest opened
  - Smooth CSS transitions
  - Visual indicator: "3 more items" badge

- **Cards**:
  - Bold title, monospace ID
  - Type badge (feature/bug/task)
  - Priority badge (high/medium/low)
  - Assigned agent indicator
  - Subtle hover effects (scale + shadow)

- **Drag & Drop**:
  - Works within visible columns only
  - Visual placeholder on drag
  - Smooth drop animation
  - WIP limit indicator per column

### 6. Agents Page
- **Layout**: Grid of agent cards
- **Card Elements**:
  - Model name (large, bold)
  - Status indicator (active/idle)
  - Recent activity count
  - Total tokens used (formatted)
  - Average execution time
  - Model-specific icon/color

### 7. Metrics Page
- **Sections**:
  - **Real-time Stats**: Events/min, avg token cost, WS connection status
  - **Trends**: Line chart of activity over time
  - **Model Breakdown**: Pie chart of token usage by model
  - **Performance**: Histogram of execution durations
  - **Leaderboard**: Top agents by token usage, activity, or success rate

---

## Animations & Interactions

### Page Load
- Staggered fade-in of sections (200ms delay between)
- Loading skeletons for data
- Header glows in from top

### Real-time Updates
- New events slide in from right with pulse
- Stats badges pulse on update
- Timeline nodes appear with pop animation

### Hover Effects
- Cards scale up 1.02x with shadow elevation
- Text color changes to lime
- Borders brighten
- Cursor changes to pointer

### State Changes
- Column collapse/expand: CSS transition (300ms)
- Tab switching: Smooth fade + slide
- Panel open/close: Spring animation

### Micro-interactions
- Loading spinner: Rotating geometric shapes
- Ripple effect on click (from click point)
- Tooltip on hover: Fade in (100ms delay)
- Confirmation dialogs: Slide up from bottom

---

## Layout Specifications

### Breakpoints
- **Desktop**: 1400px+ (full width, all columns visible)
- **Laptop**: 1024px-1399px (constrained width, 4-column Kanban)
- **Tablet**: 768px-1023px (2-column layout, 2-column Kanban)
- **Mobile**: <768px (1-column, single-column Kanban)

### Spacing System
```css
--spacing-xs: 0.25rem
--spacing-sm: 0.5rem
--spacing-md: 1rem
--spacing-lg: 1.5rem
--spacing-xl: 2rem
--spacing-2xl: 3rem
```

### Shadow System
```css
--shadow-sm: 0 1px 3px rgba(205, 255, 0, 0.05)
--shadow-md: 0 4px 12px rgba(205, 255, 0, 0.08)
--shadow-lg: 0 8px 24px rgba(205, 255, 0, 0.12)
--shadow-glow: 0 0 20px rgba(205, 255, 0, 0.15)
```

---

## Visual Accents

### Geometric Borders
- Cards: `1px solid #2A2A2A` with hover `#CDFF00`
- Inputs: `2px solid #2A2A2A` focused
- Sections: Top border accent `2px solid #CDFF00`

### Background Patterns
- Subtle grid overlay (opacity 0.03)
- Diagonal lines in card backgrounds
- Gradient overlays for depth

### Icons & Symbols
- Use SVG geometric icons (not emoji)
- Agent colors for all icons/badges
- 24x24px base size, scales with context

### Glows & Lighting
- Lime accent: Slight text-shadow glow
- Active states: box-shadow glow
- Hover effects: Brighten border and text color

---

## File Structure

```
src/python/wipnote/api/templates/
├── dashboard.html                    ← New main dashboard
├── partials/
│   ├── nav-tabs.html                ← Navigation component
│   ├── activity-timeline.html        ← Hierarchical timeline (NEW)
│   ├── orchestration-graph.html      ← DAG visualization (NEW)
│   ├── features-kanban.html          ← Smart Kanban (UPDATED)
│   ├── agents.html                   ← Agent cards (UPDATED)
│   ├── metrics.html                  ← Performance metrics (UPDATED)
│   ├── activity-feed-hierarchical.html ← Keep for backward compat
│   └── event-traces.html

src/python/wipnote/api/static/
├── style.css                         ← New comprehensive stylesheet
├── animations.css                    ← Keyframe animations
├── dashboard.js                      ← Interactive features (NEW)
└── timeline.js                       ← Timeline interactions (NEW)
```

---

## Implementation Notes

### CSS Custom Properties
All colors, spacing, and sizing use CSS variables for consistency and theming.

### JavaScript Approach
- Vanilla JS (no jQuery)
- Event delegation for efficiency
- RequestAnimationFrame for smooth animations
- LocalStorage for UI state (Kanban column visibility)

### Accessibility
- Semantic HTML5 elements
- ARIA labels on interactive elements
- Keyboard navigation support (Tab, Enter, Arrow keys)
- Focus indicators (lime borders)
- Color contrast minimum WCAG AA

### Performance
- CSS-only animations (no JavaScript)
- Lazy-load heavy visualizations
- Minimal DOM manipulation
- Event debouncing on resize
- CSS containment for isolated styling

---

## Success Criteria

- [ ] All pages load without error
- [ ] Interactive features work (tabs, Kanban, timeline expand/collapse)
- [ ] Real-time updates visible in Activity Feed
- [ ] Responsive design works at all breakpoints
- [ ] Animations are smooth (60fps)
- [ ] Color contrast meets WCAG AA
- [ ] No console errors or warnings
- [ ] Dashboard feels bold and distinctive (not generic)
- [ ] All metrics and data clearly visible
- [ ] Hover/click feedback is immediate and satisfying
