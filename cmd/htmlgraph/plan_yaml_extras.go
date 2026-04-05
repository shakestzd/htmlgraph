package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shakestzd/htmlgraph/internal/planyaml"
	"github.com/spf13/cobra"
)

func planAddQuestionYAMLCmd() *cobra.Command {
	var description, recommended, options string
	cmd := &cobra.Command{
		Use:   "add-question-yaml <plan-id> <question-text>",
		Short: "Add a question with description and recommended option to a YAML plan",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runAddQuestionYAML(args[0], args[1], description, recommended, options)
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "context paragraph (required)")
	cmd.Flags().StringVar(&recommended, "recommended", "", "recommended option key")
	cmd.Flags().StringVar(&options, "options", "", "comma-separated key:label pairs (min 2)")
	return cmd
}

func runAddQuestionYAML(planID, text, description, recommended, optionsStr string) error {
	if description == "" {
		return fmt.Errorf("--description is required")
	}
	opts := parseQuestionOptions(optionsStr)
	if len(opts) < 2 {
		return fmt.Errorf("--options must have at least 2 entries (got %d)", len(opts))
	}
	if recommended != "" {
		found := false
		for _, o := range opts {
			if o.Key == recommended {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("--recommended %q not found in options", recommended)
		}
	}
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	plan, err := planyaml.Load(planPath)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	qid := "q-" + kebabCase(text, 40)
	plan.Questions = append(plan.Questions, planyaml.PlanQuestion{
		ID: qid, Text: text, Description: description,
		Recommended: recommended, Options: opts, Answer: nil,
	})
	if err := planyaml.Save(planPath, plan); err != nil {
		return fmt.Errorf("save plan: %w", err)
	}
	fmt.Printf("Added question: %s (%d options)\n", qid, len(opts))
	return nil
}

func parseQuestionOptions(s string) []planyaml.QuestionOption {
	if s == "" {
		return nil
	}
	var opts []planyaml.QuestionOption
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		idx := strings.Index(part, ":")
		if idx < 0 {
			continue
		}
		opts = append(opts, planyaml.QuestionOption{
			Key: strings.TrimSpace(part[:idx]), Label: strings.TrimSpace(part[idx+1:]),
		})
	}
	return opts
}

func kebabCase(s string, maxLen int) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if len(s) > maxLen {
		s = s[:maxLen]
		s = strings.TrimRight(s, "-")
	}
	return s
}

func planSetCritiqueYAMLCmd() *cobra.Command {
	var data string
	cmd := &cobra.Command{
		Use:   "set-critique-yaml <plan-id>",
		Short: "Write AI critique data to a YAML plan (from --data or stdin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSetCritiqueYAML(args[0], data)
		},
	}
	cmd.Flags().StringVar(&data, "data", "", "critique JSON (reads stdin if empty)")
	return cmd
}

func runSetCritiqueYAML(planID, dataStr string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	plan, err := planyaml.Load(planPath)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	var jsonBytes []byte
	if dataStr != "" {
		jsonBytes = []byte(dataStr)
	} else {
		jsonBytes, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	}
	var critique planyaml.PlanCritique
	if err := json.Unmarshal(jsonBytes, &critique); err != nil {
		return fmt.Errorf("parse critique JSON: %w", err)
	}
	plan.Critique = &critique
	if err := planyaml.Save(planPath, plan); err != nil {
		return fmt.Errorf("save plan: %w", err)
	}
	fmt.Printf("Critique set for %s: %d assumptions, %d risks\n",
		planID, len(critique.Assumptions), len(critique.Risks))
	return nil
}

func planValidateYAMLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate-yaml <plan-id>",
		Short: "Validate a YAML plan's schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runValidateYAML(args[0])
		},
	}
}

func runValidateYAML(planID string) error {
	htmlgraphDir, err := findHtmlgraphDir()
	if err != nil {
		return err
	}
	planPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	plan, err := planyaml.Load(planPath)
	if err != nil {
		return fmt.Errorf("load plan: %w", err)
	}
	errors := planyaml.Validate(plan)
	if len(errors) > 0 {
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		return fmt.Errorf("%d validation errors", len(errors))
	}
	fmt.Printf("Plan valid: %d slices, %d questions\n", len(plan.Slices), len(plan.Questions))
	return nil
}
