from __future__ import annotations

"""
Active Learning Persistence Module.

Bridges TranscriptAnalytics to the HtmlGraph for persistent learning.
Analyzes sessions and persists patterns, insights, and metrics to the graph.
"""


from collections import Counter
from datetime import datetime
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from htmlgraph.sdk import SDK


class LearningPersistence:
    """Persists analytics insights to the HtmlGraph.

    Example:
        >>> sdk = SDK(agent="claude")
        >>> learning = LearningPersistence(sdk)
        >>> learning.persist_session_insight("sess-123")
        >>> learning.persist_patterns()
    """

    def __init__(self, sdk: SDK):
        self.sdk = sdk

    def _calculate_health(self, session: Any) -> dict[str, Any]:
        """Calculate health metrics from session activity log."""
        health: dict[str, Any] = {
            "efficiency": 0.8,  # Default reasonable value
            "retry_rate": 0.0,
            "context_rebuilds": 0,
            "tool_diversity": 0.5,
            "error_recovery": 1.0,
            "issues": [],
            "recommendations": [],
        }

        if not hasattr(session, "activity_log") or not session.activity_log:
            return health

        activities = session.activity_log
        total = len(activities)

        if total == 0:
            return health

        # Count tool usage
        tools = [
            a.tool if not isinstance(a, dict) else a.get("tool", "") for a in activities
        ]
        tool_counts = Counter(tools)
        unique_tools = len(tool_counts)

        # Tool diversity (0-1, normalized by 10 expected tools)
        health["tool_diversity"] = min(unique_tools / 10.0, 1.0)

        # Detect retries (same tool twice in a row)
        retries = sum(1 for i in range(1, len(tools)) if tools[i] == tools[i - 1])
        health["retry_rate"] = retries / total if total > 0 else 0.0

        # Detect context rebuilds (Read same file multiple times)
        reads = [
            a
            for a in activities
            if (hasattr(a, "tool") and a.tool == "Read")
            or (isinstance(a, dict) and a.get("tool") == "Read")
        ]
        if reads:
            read_targets = [
                str(
                    getattr(r, "summary", "")
                    if hasattr(r, "summary")
                    else r.get("summary", "")
                )
                for r in reads
            ]
            rebuild_count = len(read_targets) - len(set(read_targets))
            health["context_rebuilds"] = rebuild_count

        # Calculate efficiency (inverse of wasted operations)
        wasted = retries + health["context_rebuilds"]
        health["efficiency"] = max(0.0, 1.0 - (wasted / total))

        # Generate issues
        if health["retry_rate"] > 0.2:
            health["issues"].append(f"High retry rate: {health['retry_rate']:.0%}")
            health["recommendations"].append(
                "Consider reading more context before acting"
            )

        if health["context_rebuilds"] > 2:
            health["issues"].append(
                f"Excessive context rebuilds: {health['context_rebuilds']}"
            )
            health["recommendations"].append("Cache file contents or take notes")

        if health["tool_diversity"] < 0.3:
            health["issues"].append("Low tool diversity")
            health["recommendations"].append("Consider using more specialized tools")

        return health

    def persist_patterns(self, min_count: int = 2) -> list[str]:
        """Detect and persist workflow patterns IN SESSIONS (not as separate files).

        This refactored version stores patterns inline within session HTML files
        to avoid creating 2,890+ individual pattern files.

        Args:
            min_count: Minimum occurrences to persist a pattern

        Returns:
            List of session IDs that had patterns updated
        """
        # Collect tool sequences per session (not globally)
        session_ids_updated: list[str] = []

        for session in self.sdk.session_manager.session_converter.load_all():
            if not session.activity_log:
                continue

            # Extract 3-tool sequences from this session
            tools = [
                a.tool if not isinstance(a, dict) else a.get("tool", "")
                for a in session.activity_log
            ]

            # Count sequences in this session
            sequences: list[tuple[Any, ...]] = []
            for i in range(len(tools) - 2):
                seq = tools[i : i + 3]
                if all(seq):  # No empty tools
                    sequences.append(tuple(seq))

            seq_counts = Counter(sequences)

            # Update session's detected_patterns
            patterns_updated = False
            for seq, count in seq_counts.items():  # type: ignore[assignment]
                if count >= min_count:
                    # Check if pattern already exists in this session
                    existing = next(
                        (
                            p
                            for p in session.detected_patterns
                            if p.get("sequence") == list(seq)
                        ),
                        None,
                    )

                    if existing:
                        # Update existing pattern
                        existing["detection_count"] = count
                        existing["last_detected"] = datetime.now().isoformat()
                        patterns_updated = True
                    else:
                        # Add new pattern to session
                        pattern_type = self._classify_pattern(list(seq))
                        now = datetime.now()
                        session.detected_patterns.append(
                            {
                                "sequence": list(seq),
                                "pattern_type": pattern_type,
                                "detection_count": count,
                                "first_detected": now.isoformat(),
                                "last_detected": now.isoformat(),
                            }
                        )
                        patterns_updated = True

            # Save updated session if patterns were modified
            if patterns_updated:
                self.sdk.session_manager.session_converter.save(session)
                session_ids_updated.append(session.id)

        # Also persist parallel patterns
        parallel_session_ids = self.persist_parallel_patterns(min_count=min_count)
        session_ids_updated.extend(parallel_session_ids)

        return session_ids_updated

    def persist_parallel_patterns(self, min_count: int = 2) -> list[str]:
        """Detect and persist parallel execution patterns IN SESSIONS.

        Identifies when multiple tools are invoked in parallel (same parent_activity_id).
        This is especially useful for detecting orchestrator patterns like parallel Task delegation.

        Args:
            min_count: Minimum occurrences to persist a pattern

        Returns:
            List of session IDs that had parallel patterns updated
        """
        from collections import defaultdict

        session_ids_updated: list[str] = []

        for session in self.sdk.session_manager.session_converter.load_all():
            if not session.activity_log:
                continue

            # Group activities by parent_activity_id
            parent_groups: dict[str, list[Any]] = defaultdict(list)
            for activity in session.activity_log:
                parent_id = (
                    activity.parent_activity_id
                    if not isinstance(activity, dict)
                    else activity.get("parent_activity_id")
                )
                if parent_id:  # Only track activities with a parent
                    parent_groups[parent_id].append(activity)

            # Collect parallel patterns for this session
            parallel_patterns: list[tuple[str, ...]] = []
            for parent_id, activities in parent_groups.items():
                if len(activities) < 2:
                    continue

                # Sort by timestamp
                sorted_activities = sorted(
                    activities,
                    key=lambda a: (
                        a.timestamp
                        if not isinstance(a, dict)
                        else a.get("timestamp", datetime.min)
                    ),
                )

                # Extract tool sequence
                tools = tuple(
                    a.tool if not isinstance(a, dict) else a.get("tool", "")
                    for a in sorted_activities
                )

                # Filter out empty tools
                if all(tools):
                    parallel_patterns.append(tools)

            # Count parallel patterns in this session
            pattern_counts = Counter(parallel_patterns)

            # Update session's detected_patterns with parallel patterns
            patterns_updated = False
            for tools, count in pattern_counts.items():
                if count >= min_count:
                    tool_names = list(tools)

                    # Check if pattern already exists in this session
                    # Parallel patterns have special naming: "Parallel[N]: tool1 || tool2"
                    existing = next(
                        (
                            p
                            for p in session.detected_patterns
                            if p.get("sequence") == tool_names
                            and p.get("is_parallel", False)
                        ),
                        None,
                    )

                    if existing:
                        # Update existing parallel pattern
                        existing["detection_count"] = count
                        existing["last_detected"] = datetime.now().isoformat()
                        patterns_updated = True
                    else:
                        # Add new parallel pattern to session
                        pattern_type = self._classify_pattern(
                            tool_names, is_parallel=True
                        )
                        now = datetime.now()
                        session.detected_patterns.append(
                            {
                                "sequence": tool_names,
                                "pattern_type": pattern_type,
                                "detection_count": count,
                                "first_detected": now.isoformat(),
                                "last_detected": now.isoformat(),
                                "is_parallel": True,
                                "parallel_count": len(tools),
                            }
                        )
                        patterns_updated = True

            # Save updated session if patterns were modified
            if patterns_updated:
                self.sdk.session_manager.session_converter.save(session)
                session_ids_updated.append(session.id)

        return session_ids_updated

    def _classify_pattern(self, sequence: list[str], is_parallel: bool = False) -> str:
        """Classify a pattern as optimal, anti-pattern, or neutral.

        Args:
            sequence: List of tool names in the pattern
            is_parallel: Whether this is a parallel execution pattern

        Returns:
            Pattern classification string
        """
        seq = tuple(sequence)

        # Orchestrator patterns (parallel execution)
        if is_parallel:
            # Parallel Task delegation is optimal (orchestrator pattern)
            if all(tool == "Task" for tool in sequence) and len(sequence) >= 2:
                return "optimal"
            # Mixed parallel operations can also be optimal
            if "Task" in sequence:
                return "optimal"
            # Other parallel patterns are neutral
            return "neutral"

        # Sequential anti-patterns for orchestrators
        # Multiple sequential Tasks without parallelism is an anti-pattern
        if seq == ("Task", "Task", "Task"):
            return "anti-pattern"

        # Known optimal patterns (sequential)
        optimal = [
            ("Read", "Edit", "Bash"),  # Read, modify, test
            ("Grep", "Read", "Edit"),  # Search, understand, modify
            ("Glob", "Read", "Edit"),  # Find, understand, modify
        ]

        # Known anti-patterns (sequential)
        anti = [
            ("Edit", "Edit", "Edit"),  # Too many edits without testing
            ("Bash", "Bash", "Bash"),  # Command spam
            ("Read", "Read", "Read"),  # Excessive reading without action
        ]

        if seq in optimal:
            return "optimal"
        elif seq in anti:
            return "anti-pattern"
        else:
            return "neutral"

    def analyze_for_orchestrator(self, session_id: str) -> dict[str, Any]:
        """Analyze session and return compact feedback for orchestrator.

        This method is called on work item completion to surface:
        - Anti-patterns detected in the session
        - Errors encountered
        - Efficiency metrics
        - Test execution results (pytest)
        - Actionable recommendations

        Args:
            session_id: Session to analyze

        Returns:
            Dict with analysis results for orchestrator feedback
        """
        result: dict[str, Any] = {
            "session_id": session_id,
            "anti_patterns": [],
            "errors": [],
            "error_count": 0,
            "efficiency": 0.8,
            "issues": [],
            "recommendations": [],
            "test_runs": [],
            "test_summary": None,
            "summary": "",
        }

        session = self.sdk.session_manager.get_session(session_id)
        if (
            not session
            or not hasattr(session, "activity_log")
            or not session.activity_log
        ):
            result["summary"] = "No activity data available for analysis"
            return result

        activities = session.activity_log

        # Count errors (success=False)
        errors = []
        for a in activities:
            success = a.success if not isinstance(a, dict) else a.get("success", True)
            if not success:
                tool = a.tool if not isinstance(a, dict) else a.get("tool", "")
                summary = a.summary if not isinstance(a, dict) else a.get("summary", "")
                errors.append({"tool": tool, "summary": summary[:100]})

        result["errors"] = errors[:10]  # Limit to 10 most recent
        result["error_count"] = len(errors)

        # Detect anti-patterns in this session
        tools = [
            a.tool if not isinstance(a, dict) else a.get("tool", "") for a in activities
        ]

        # Known anti-patterns
        anti_patterns = [
            ("Edit", "Edit", "Edit"),
            ("Bash", "Bash", "Bash"),
            ("Read", "Read", "Read"),
        ]

        # Count anti-pattern occurrences
        anti_pattern_counts: Counter[tuple[str, ...]] = Counter()
        for i in range(len(tools) - 2):
            seq = tuple(tools[i : i + 3])
            if seq in anti_patterns:
                anti_pattern_counts[seq] += 1

        for seq, count in anti_pattern_counts.most_common():
            result["anti_patterns"].append(
                {
                    "sequence": list(seq),
                    "count": count,
                    "description": self._describe_anti_pattern(seq),
                }
            )

        # Calculate health metrics
        health = self._calculate_health(session)
        result["efficiency"] = health.get("efficiency", 0.8)
        result["issues"] = health.get("issues", [])
        result["recommendations"] = health.get("recommendations", [])

        # Analyze test runs (pytest)
        test_analysis = self._analyze_test_runs(activities)
        result["test_runs"] = test_analysis["test_runs"]
        result["test_summary"] = test_analysis["summary"]

        # Add test-related issues and recommendations
        if test_analysis.get("issues"):
            result["issues"].extend(test_analysis["issues"])
        if test_analysis.get("recommendations"):
            result["recommendations"].extend(test_analysis["recommendations"])

        # Generate summary
        summary_parts = []
        if result["error_count"] > 0:
            summary_parts.append(f"{result['error_count']} errors")
        if result["anti_patterns"]:
            total_anti = sum(p["count"] for p in result["anti_patterns"])
            summary_parts.append(f"{total_anti} anti-pattern occurrences")
        if result["efficiency"] < 0.7:
            summary_parts.append(f"low efficiency ({result['efficiency']:.0%})")

        # Include test summary in main summary
        if result["test_summary"]:
            summary_parts.append(result["test_summary"])

        if summary_parts:
            result["summary"] = "⚠️ Issues: " + ", ".join(summary_parts)
        else:
            result["summary"] = "✓ Session completed cleanly"

        return result

    def _analyze_test_runs(self, activities: list[Any]) -> dict[str, Any]:
        """Analyze pytest test runs from activity log.

        Args:
            activities: List of ActivityEntry objects

        Returns:
            Dict with test_runs, summary, issues, recommendations
        """
        import re

        result: dict[str, Any] = {
            "test_runs": [],
            "summary": None,
            "issues": [],
            "recommendations": [],
        }

        # Find all pytest runs in Bash activities
        for activity in activities:
            tool = (
                activity.tool
                if not isinstance(activity, dict)
                else activity.get("tool", "")
            )
            summary = (
                activity.summary
                if not isinstance(activity, dict)
                else activity.get("summary", "")
            )
            success = (
                activity.success
                if not isinstance(activity, dict)
                else activity.get("success", True)
            )

            # Check if this is a pytest run
            if tool == "Bash" and (
                "pytest" in summary.lower() or "py.test" in summary.lower()
            ):
                test_run: dict[str, Any] = {
                    "command": summary,
                    "success": success,
                    "passed": None,
                    "failed": None,
                    "skipped": None,
                    "errors": None,
                }

                # Try to extract test results from payload if available
                payload = (
                    activity.payload
                    if not isinstance(activity, dict)
                    else activity.get("payload", {})
                )
                if payload and isinstance(payload, dict):
                    output = payload.get("output", "") or payload.get("stdout", "")
                    if output:
                        # Parse pytest output for results
                        # Example: "5 passed, 2 failed, 1 skipped in 2.34s"
                        # Example: "===== 10 passed in 1.23s ====="
                        passed_match = re.search(r"(\d+)\s+passed", output)
                        failed_match = re.search(r"(\d+)\s+failed", output)
                        skipped_match = re.search(r"(\d+)\s+skipped", output)
                        error_match = re.search(r"(\d+)\s+error", output)

                        if passed_match:
                            test_run["passed"] = int(passed_match.group(1))
                        if failed_match:
                            test_run["failed"] = int(failed_match.group(1))
                        if skipped_match:
                            test_run["skipped"] = int(skipped_match.group(1))
                        if error_match:
                            test_run["errors"] = int(error_match.group(1))

                result["test_runs"].append(test_run)

        # Generate summary and recommendations
        if result["test_runs"]:
            total_runs = len(result["test_runs"])
            successful_runs = sum(1 for r in result["test_runs"] if r["success"])
            failed_runs = total_runs - successful_runs

            # Calculate total test results across all runs
            total_passed = sum(r["passed"] or 0 for r in result["test_runs"])
            total_failed = sum(r["failed"] or 0 for r in result["test_runs"])
            total_errors = sum(r["errors"] or 0 for r in result["test_runs"])

            # Generate summary
            summary_parts = [f"{total_runs} test run{'s' if total_runs > 1 else ''}"]
            if total_passed > 0:
                summary_parts.append(f"{total_passed} passed")
            if total_failed > 0:
                summary_parts.append(f"{total_failed} failed")
            if total_errors > 0:
                summary_parts.append(f"{total_errors} errors")

            result["summary"] = ", ".join(summary_parts)

            # Add issues and recommendations
            if failed_runs > 0:
                result["issues"].append(
                    f"{failed_runs} test run{'s' if failed_runs > 1 else ''} failed"
                )

            if total_runs > 5:
                result["issues"].append(f"High test run count: {total_runs}")
                result["recommendations"].append(
                    "Consider fixing tests in one batch to reduce test iterations"
                )

            if total_failed > 0 and successful_runs == 0:
                result["recommendations"].append(
                    "No passing test runs - verify test environment and dependencies"
                )

            # Positive feedback for good testing practices
            if successful_runs > 0 and failed_runs == 0:
                result["summary"] = f"✓ {result['summary']}"

        return result

    def _describe_anti_pattern(self, seq: tuple) -> str:
        """Return human-readable description of an anti-pattern."""
        descriptions = {
            (
                "Edit",
                "Edit",
                "Edit",
            ): "Multiple edits without testing - run tests between changes",
            ("Bash", "Bash", "Bash"): "Command spam - plan commands before executing",
            (
                "Read",
                "Read",
                "Read",
            ): "Excessive reading - take notes or use grep to find specific content",
        }
        return descriptions.get(seq, f"Repeated {seq[0]} without variation")


def analyze_on_completion(sdk: SDK, session_id: str) -> dict:
    """Analyze session on work item completion and return orchestrator feedback.

    This is the main entry point called by complete_feature().

    Returns:
        Dict with:
        - anti_patterns: List of detected anti-patterns with counts
        - errors: List of error summaries
        - error_count: Total error count
        - efficiency: Efficiency score (0.0-1.0)
        - issues: List of detected issues
        - recommendations: List of recommendations
        - summary: One-line summary for orchestrator
    """
    learning = LearningPersistence(sdk)
    return learning.analyze_for_orchestrator(session_id)


def auto_persist_on_session_end(sdk: SDK, session_id: str) -> dict:
    """Convenience function to auto-persist learning data when session ends.

    Returns:
        Dict with pattern_ids
    """
    learning = LearningPersistence(sdk)

    result: dict[str, object] = {
        "pattern_ids": learning.persist_patterns(),
    }

    return result
