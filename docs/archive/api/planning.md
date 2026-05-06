# Planning

Spec and Plan models for track planning.

## Spec

The Spec model defines requirements and success criteria for a track.

```python
from wipnote.planning import Spec, Requirement, AcceptanceCriterion

spec = Spec(
    overview="Add OAuth 2.0 support for user authentication",
    context="Current system has no authentication. Need secure login.",
    requirements=[
        Requirement(
            description="Implement OAuth 2.0 flow",
            priority="must-have"
        ),
        Requirement(
            description="Add JWT token management",
            priority="must-have"
        )
    ],
    acceptance_criteria=[
        AcceptanceCriterion(
            description="Users can log in with Google/GitHub",
            test_case="OAuth login test passes"
        )
    ]
)
```

### Fields

- `overview: str` - High-level summary
- `context: str` - Background and constraints
- `requirements: list[Requirement]` - List of requirements
- `acceptance_criteria: list[AcceptanceCriterion]` - Success criteria
- `constraints: Optional[list[str]]` - Technical/business constraints

## Plan

The Plan model defines phased implementation tasks.

```python
from wipnote.planning import Plan, Phase, Task

plan = Plan(
    phases=[
        Phase(
            name="Phase 1: Setup",
            tasks=[
                Task(description="Configure OAuth providers", estimate_hours=2.0),
                Task(description="Set up database schema", estimate_hours=1.0)
            ]
        ),
        Phase(
            name="Phase 2: Implementation",
            tasks=[
                Task(description="Implement login flow", estimate_hours=4.0),
                Task(description="Add JWT middleware", estimate_hours=3.0)
            ]
        )
    ]
)
```

### Fields

- `phases: list[Phase]` - List of implementation phases
- `total_estimate_hours: float` - Auto-calculated total time

## Requirement

```python
from wipnote.planning import Requirement, RequirementPriority

req = Requirement(
    description="Implement OAuth 2.0 flow",
    priority=RequirementPriority.MUST_HAVE
)
```

### Fields

- `description: str` - Requirement description
- `priority: RequirementPriority` - must-have, should-have, nice-to-have

## AcceptanceCriterion

```python
from wipnote.planning import AcceptanceCriterion

criterion = AcceptanceCriterion(
    description="Users can log in with Google",
    test_case="OAuth login test passes"
)
```

### Fields

- `description: str` - What success looks like
- `test_case: Optional[str]` - How to verify

## Phase

```python
from wipnote.planning import Phase, Task

phase = Phase(
    name="Phase 1: Setup",
    tasks=[
        Task(description="Configure OAuth (2h)", estimate_hours=2.0),
        Task(description="Setup DB (1h)", estimate_hours=1.0)
    ]
)
```

### Fields

- `name: str` - Phase name
- `tasks: list[Task]` - List of tasks in this phase

## Task

```python
from wipnote.planning import Task

task = Task(
    description="Implement OAuth flow",
    estimate_hours=4.0,
    completed=False
)
```

### Fields

- `description: str` - Task description
- `estimate_hours: Optional[float]` - Time estimate
- `completed: bool` - Completion status

## Complete API Reference

For detailed API documentation with type signatures and validation rules, see the Python source code in `src/python/wipnote/planning.py`.
