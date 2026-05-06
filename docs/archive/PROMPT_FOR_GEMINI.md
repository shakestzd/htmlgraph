# Prompt for Gemini

Copy this and paste it into Gemini:

---

Please analyze Wipnote's dependencies to maximize value for AI agents and human users.

**Task Details:**
- Read the comprehensive instructions: `GEMINI_TASK_DEPENDENCY_ANALYSIS.md`
- Update spike: `spk-f1bf9a98` with your findings

**Key Context:**

1. **Philosophy Evolution** (CRITICAL):
   - ❌ OLD: "HTML is All You Need" (outdated, limiting)
   - ✅ NEW: Use the BEST tools to maximize agent/user experience
   - We already use Python, Pydantic, SQLite, Rich - don't self-limit!

2. **Current Dependencies to Analyze:**
   - justhtml (HTML parsing/generation)
   - pydantic (data validation)
   - watchdog (file monitoring)
   - rich (terminal UI)
   - jinja2 (templates)
   - typing_extensions (type hints)

3. **Your Mission:**
   - Map current usage vs unused features for each dependency
   - Identify high-impact optimizations
   - **THINK BIG**: Recommend transformative new dependencies
   - Don't be dogmatic about lean dependencies - prioritize impact
   - Research: official docs, GitHub, web, real-world examples

4. **Areas to Explore for New Dependencies:**
   - Graph visualization (networkx, plotly, graphviz)
   - Schema generation (OpenAPI, JSON Schema, GraphQL)
   - Terminal UIs (textual, rich advanced features)
   - Web dashboards (FastAPI, Streamlit)
   - Performance (orjson, msgpack, caching)
   - Query DSLs (Cypher-like, graph query)

5. **Deliverable:**
   Update spike `spk-f1bf9a98` with:
   ```python
   from wipnote import SDK

   sdk = SDK(agent='gemini')

   with sdk.spikes.edit('spk-f1bf9a98') as spike:
       spike.findings = '''
       [Your complete analysis here - see template in delegation file]
       '''
       spike.decision = '''
       Priority Order:
       1. [Top priority improvement]
       2. [Second priority]
       3. [Third priority]

       New Dependencies to Add:
       - [Package name]: [Why transformative]

       Next Steps:
       [What to implement first]
       '''
   ```

6. **Focus:**
   - Primary audience: AI agents (easy to learn, discover, use)
   - Secondary audience: Human users (beautiful output, clear history)
   - High bar for impact, not arbitrary constraints
   - Be bold with recommendations!

**Start by reading:** `GEMINI_TASK_DEPENDENCY_ANALYSIS.md` for complete instructions.
