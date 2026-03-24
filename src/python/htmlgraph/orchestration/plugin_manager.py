from __future__ import annotations

"""Plugin management for HtmlGraph Claude Code integration.

Centralizes plugin installation, directory management, and validation.
"""

import logging
import subprocess
import sys
from pathlib import Path
from typing import TYPE_CHECKING

logger = logging.getLogger(__name__)

if TYPE_CHECKING:
    pass


class PluginManager:
    """Manage HtmlGraph Claude plugin installation and directories."""

    @staticmethod
    def get_plugin_dir() -> Path:
        """Get the plugin directory path.

        Returns:
            Path to packages/claude-plugin (the plugin root, not .claude-plugin)
        """
        # Resolve from cwd (project root), not from installed package location
        return Path.cwd() / "packages" / "claude-plugin"

    @staticmethod
    def install_or_update(verbose: bool = True) -> None:
        """Install or update HtmlGraph plugin.

        Args:
            verbose: Whether to show progress messages
        """
        if verbose:
            logger.info("\n📦 Installing/upgrading HtmlGraph plugin...\n")

        # Step 0: Check if plugin is already installed and enabled
        try:
            result = subprocess.run(
                ["claude", "plugin", "list"],
                capture_output=True,
                text=True,
                check=False,
            )
            if result.returncode == 0:
                # Check if htmlgraph plugin is installed
                if "htmlgraph" in result.stdout:
                    if verbose:
                        logger.info(
                            "  ✓ Plugin already installed and enabled, skipping install"
                        )
                    return
            else:
                if verbose:
                    logger.info(
                        f"    ⚠ Could not check plugin list: {result.stderr.strip()}"
                    )
        except FileNotFoundError:
            if verbose:
                logger.info("    ⚠ 'claude' command not found")
        except Exception as e:
            if verbose:
                logger.info(f"    ⚠ Error checking plugin list: {e}")

        # Step 1: Try update, fallback to install
        try:
            if verbose:
                logger.info("  Updating plugin to latest version...")
            result = subprocess.run(
                ["claude", "plugin", "update", "htmlgraph"],
                capture_output=True,
                text=True,
                check=False,
            )
            if result.returncode == 0:
                if verbose:
                    logger.info("    ✓ Plugin updated successfully")
            else:
                # Fallback to install
                if (
                    "not installed" in result.stderr.lower()
                    or "not found" in result.stderr.lower()
                ):
                    if verbose:
                        logger.info("    ℹ Plugin not yet installed, installing...")
                    install_result = subprocess.run(
                        ["claude", "plugin", "install", "htmlgraph"],
                        capture_output=True,
                        text=True,
                        check=False,
                    )
                    if install_result.returncode == 0:
                        if verbose:
                            logger.info("    ✓ Plugin installed successfully")
                    elif verbose:
                        logger.info(
                            f"    ⚠ Plugin install: {install_result.stderr.strip()}"
                        )
                elif verbose:
                    logger.info(f"    ⚠ Plugin update: {result.stderr.strip()}")
        except FileNotFoundError:
            if verbose:
                logger.info("    ⚠ 'claude' command not found")
        except Exception as e:
            if verbose:
                logger.info(f"    ⚠ Error updating plugin: {e}")

        if verbose:
            logger.info("\n✓ Plugin installation complete\n")

    @staticmethod
    def validate_plugin_dir(plugin_dir: Path) -> None:
        """Validate that plugin directory exists, exit if not.

        Args:
            plugin_dir: Path to plugin directory

        Raises:
            SystemExit: If plugin directory doesn't exist
        """
        if not plugin_dir.exists():
            logger.warning(f"Error: Plugin directory not found: {plugin_dir}")
            print(
                "Expected location: packages/claude-plugin/.claude-plugin",
                file=sys.stderr,
            )
            sys.exit(1)
