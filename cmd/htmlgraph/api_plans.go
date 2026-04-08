package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	dbpkg "github.com/shakestzd/htmlgraph/internal/db"
	"github.com/shakestzd/htmlgraph/internal/planchat"
)

// planListItem is a single entry in the GET /api/plans response.
type planListItem struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`
	FeatureID  string    `json:"feature_id"`
	Approved   int       `json:"approved"`
	Total      int       `json:"total"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// plansListHandler returns a JSON array of all plans sorted by mtime desc.
// GET /api/plans
func plansListHandler(htmlgraphDir string, database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		plansDir := filepath.Join(htmlgraphDir, "plans")
		entries, err := os.ReadDir(plansDir)
		if err != nil {
			if os.IsNotExist(err) {
				respondJSON(w, []planListItem{})
				return
			}
			http.Error(w, fmt.Sprintf("reading plans dir: %v", err), http.StatusInternalServerError)
			return
		}

		var items []planListItem
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
				continue
			}
			planID := strings.TrimSuffix(entry.Name(), ".html")
			planPath := filepath.Join(plansDir, entry.Name())

			item, err := parsePlanListItem(planPath, planID, database)
			if err != nil {
				continue
			}
			items = append(items, item)
		}

		sort.Slice(items, func(i, j int) bool {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		})

		if items == nil {
			items = []planListItem{}
		}
		respondJSON(w, items)
	}
}

// parsePlanListItem reads a plan HTML file and extracts list metadata.
// Merges approval counts from SQLite (live feedback) with HTML (static defaults).
func parsePlanListItem(planPath, planID string, database *sql.DB) (planListItem, error) {
	info, err := os.Stat(planPath)
	if err != nil {
		return planListItem{}, err
	}

	f, err := os.Open(planPath)
	if err != nil {
		return planListItem{}, err
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return planListItem{}, err
	}

	article := doc.Find("article[id]").First()
	status, _ := article.Attr("data-status")
	if status == "" {
		status = "draft"
	}
	featureID, _ := article.Attr("data-feature-id")

	title := strings.TrimSpace(doc.Find("h1").First().Text())
	if title == "" {
		title = planID
	}

	// Count total approve checkboxes from HTML (defines the section count).
	var total int
	doc.Find("input[data-action='approve']").Each(func(_ int, s *goquery.Selection) {
		total++
	})

	// Get live approval count from SQLite (overrides HTML checked attrs).
	approved := 0
	if database != nil {
		feedback, err := dbpkg.GetPlanFeedback(database, planID)
		if err == nil {
			for _, fb := range feedback {
				if fb.Action == "approve" && fb.Value == "true" {
					approved++
				}
			}
		}
		// Also check if finalized in SQLite
		isApproved, _ := dbpkg.IsPlanFullyApproved(database, planID)
		if isApproved && status != "finalized" {
			status = "finalized"
		}
	}

	// Fall back to HTML checked attrs if SQLite has nothing
	if approved == 0 {
		doc.Find("input[data-action='approve']").Each(func(_ int, s *goquery.Selection) {
			if _, exists := s.Attr("checked"); exists {
				approved++
			}
		})
	}

	return planListItem{
		ID:        planID,
		Title:     title,
		Status:    status,
		FeatureID: featureID,
		Approved:  approved,
		Total:     total,
		UpdatedAt: info.ModTime().UTC(),
	}, nil
}

// planStatusResponse is returned by GET /api/plans/{id}/status.
type planStatusResponse struct {
	PlanID        string `json:"plan_id"`
	Status        string `json:"status"`
	ApprovedCount int    `json:"approved_count"`
	TotalSections int    `json:"total_sections"`
}

// planFeedbackResponse is returned by GET /api/plans/{id}/feedback.
type planFeedbackResponse struct {
	PlanID       string                     `json:"plan_id"`
	Status       string                     `json:"status"`
	Sections     map[string]sectionFeedback `json:"sections"`
	Questions    map[string]string          `json:"questions"`
	ChatMessages []chatMessageEntry         `json:"chat_messages,omitempty"`
}

type sectionFeedback struct {
	Approved bool   `json:"approved"`
	Comment  string `json:"comment"`
}

// chatMessageEntry is a single chat message in the feedback response.
type chatMessageEntry struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

// planFeedbackRequest is the body for POST /api/plans/{id}/feedback.
type planFeedbackRequest struct {
	Section    string `json:"section"`
	Action     string `json:"action"`
	Value      string `json:"value"`
	QuestionID string `json:"question_id"`
}

// planFileHandler serves HTML plan files from .htmlgraph/plans/{id}.html.
// GET /plans/{id}.html
func planFileHandler(htmlgraphDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// URL path: /plans/{id}.html — strip the /plans/ prefix.
		name := strings.TrimPrefix(r.URL.Path, "/plans/")
		if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
			http.Error(w, "invalid plan path", http.StatusBadRequest)
			return
		}
		if !strings.HasSuffix(name, ".html") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		planPath := filepath.Join(htmlgraphDir, "plans", name)
		if _, err := os.Stat(planPath); err != nil {
			http.Error(w, "plan not found", http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, planPath)
	}
}

// planStatusHandler returns status information for a plan.
// GET /api/plans/{id}/status
func planStatusHandler(database *sql.DB, htmlgraphDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		planID, err := extractPlanID(r.URL.Path, "/status")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		planPath, err := resolvePlanPath(htmlgraphDir, planID)
		if err != nil {
			http.Error(w, "plan not found", http.StatusNotFound)
			return
		}

		htmlStatus, err := parsePlanHTMLStatus(planPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("reading plan: %v", err), http.StatusInternalServerError)
			return
		}

		approvedCount, totalSections, err := countPlanSections(database, planID)
		if err != nil {
			http.Error(w, fmt.Sprintf("querying feedback: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, planStatusResponse{
			PlanID:        planID,
			Status:        htmlStatus,
			ApprovedCount: approvedCount,
			TotalSections: totalSections,
		})
	}
}

// planFeedbackSubmitHandler stores a feedback entry for a plan section.
// POST /api/plans/{id}/feedback
func planFeedbackSubmitHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req planFeedbackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if req.Section == "" || req.Action == "" {
			http.Error(w, "section and action are required", http.StatusBadRequest)
			return
		}

		planID, err := extractPlanID(r.URL.Path, "/feedback")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := dbpkg.StorePlanFeedback(database, planID, req.Section, req.Action, req.Value, req.QuestionID); err != nil {
			http.Error(w, fmt.Sprintf("storing feedback: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, map[string]string{"status": "ok"})
	}
}

// planFinalizeHandler finalizes a plan once all sections are approved.
// POST /api/plans/{id}/finalize
func planFinalizeHandler(database *sql.DB, htmlgraphDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		planID, err := extractPlanID(r.URL.Path, "/finalize")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		approved, err := dbpkg.IsPlanFullyApproved(database, planID)
		if err != nil {
			http.Error(w, fmt.Sprintf("checking approval: %v", err), http.StatusInternalServerError)
			return
		}
		if !approved {
			http.Error(w, "not all sections approved", http.StatusBadRequest)
			return
		}

		if err := dbpkg.FinalizePlan(database, planID); err != nil {
			http.Error(w, fmt.Sprintf("finalizing plan: %v", err), http.StatusInternalServerError)
			return
		}

		// Write finalized HTML snapshot with all feedback baked in.
		planPath, err := resolvePlanPath(htmlgraphDir, planID)
		if err == nil {
			_ = finalizePlanHTML(planPath, database, planID)
		}

		feedback, err := dbpkg.GetPlanFeedback(database, planID)
		if err != nil {
			http.Error(w, fmt.Sprintf("reading feedback: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, map[string]any{
			"plan_id":  planID,
			"status":   "finalized",
			"feedback": feedback,
		})
	}
}

// planFeedbackReadHandler returns structured feedback for a plan.
// GET /api/plans/{id}/feedback
func planFeedbackReadHandler(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		planID, err := extractPlanID(r.URL.Path, "/feedback")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		entries, err := dbpkg.GetPlanFeedback(database, planID)
		if err != nil {
			http.Error(w, fmt.Sprintf("reading feedback: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, buildFeedbackResponse(planID, entries))
	}
}

// planFeedbackHandler routes GET and POST for /api/plans/{id}/feedback.
func planFeedbackHandler(database *sql.DB) http.HandlerFunc {
	submitH := planFeedbackSubmitHandler(database)
	readH := planFeedbackReadHandler(database)
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			submitH(w, r)
		case http.MethodGet:
			readH(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// planDeleteHandler deletes a draft plan's HTML file and feedback.
// DELETE /api/plans/{id}/delete
func planDeleteHandler(database *sql.DB, htmlgraphDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		planID, err := extractPlanID(r.URL.Path, "/delete")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		planPath, err := resolvePlanPath(htmlgraphDir, planID)
		if err != nil {
			http.Error(w, "plan not found", http.StatusNotFound)
			return
		}

		htmlStatus, err := parsePlanHTMLStatus(planPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("reading plan: %v", err), http.StatusInternalServerError)
			return
		}

		// Only allow deletion of draft or in-progress plans
		if htmlStatus == "finalized" {
			http.Error(w, "Cannot delete a finalized plan", http.StatusBadRequest)
			return
		}

		// Delete the HTML file
		if err := os.Remove(planPath); err != nil {
			http.Error(w, fmt.Sprintf("deleting plan file: %v", err), http.StatusInternalServerError)
			return
		}

		// Delete feedback from SQLite
		if err := dbpkg.DeletePlanFeedback(database, planID); err != nil {
			http.Error(w, fmt.Sprintf("deleting feedback: %v", err), http.StatusInternalServerError)
			return
		}

		respondJSON(w, map[string]string{"status": "deleted", "plan_id": planID})
	}
}

// planChatRequest is the body for POST /api/plans/{id}/chat.
type planChatRequest struct {
	Message string `json:"message"`
}

// planChatHandler streams Claude responses as SSE for a plan chat session.
// POST /api/plans/{id}/chat
func planChatHandler(database *sql.DB, htmlgraphDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		planID, err := extractPlanID(r.URL.Path, "/chat")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req planChatRequest
		body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		if err != nil {
			http.Error(w, "reading request body", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
			return
		}
		if req.Message == "" {
			http.Error(w, "message is required", http.StatusBadRequest)
			return
		}

		// Load plan YAML for context.
		planContext := loadPlanContext(htmlgraphDir, planID)

		// Resolve project dir (parent of .htmlgraph/).
		projectDir := filepath.Dir(htmlgraphDir)

		backend := planchat.New(database, planID, planContext, projectDir)
		if !backend.IsAvailable() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Claude unavailable. Install claude CLI or set ANTHROPIC_API_KEY.",
			})
			return
		}

		// Store user message.
		_ = backend.SaveMessage("user", req.Message)

		// Set SSE headers.
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Stream response.
		chunks, errCh := backend.Send(r.Context(), req.Message)

		var fullResponse strings.Builder
		for chunk := range chunks {
			fullResponse.WriteString(chunk)
			payload, _ := json.Marshal(map[string]string{
				"type": "chunk",
				"text": chunk,
			})
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}

		// Check for errors.
		if err := <-errCh; err != nil {
			payload, _ := json.Marshal(map[string]string{
				"type":  "error",
				"error": err.Error(),
			})
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		}

		// Store assistant message.
		if fullResponse.Len() > 0 {
			_ = backend.SaveMessage("assistant", fullResponse.String())
		}

		// Send done event.
		fmt.Fprintf(w, "data: %s\n\n", `{"type":"done"}`)
		flusher.Flush()
	}
}

// loadPlanContext reads the plan YAML file for use as Claude context.
// Falls back to empty string if the file is not found.
func loadPlanContext(htmlgraphDir, planID string) string {
	yamlPath := filepath.Join(htmlgraphDir, "plans", planID+".yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		// Try HTML fallback for plan context.
		htmlPath := filepath.Join(htmlgraphDir, "plans", planID+".html")
		data, err = os.ReadFile(htmlPath)
		if err != nil {
			return ""
		}
	}
	return string(data)
}

// planRouter dispatches /api/plans/{id}/{action} to the appropriate handler.
// Registered under /api/plans/ in serve.go.
func planRouter(database *sql.DB, htmlgraphDir string) http.HandlerFunc {
	statusH := planStatusHandler(database, htmlgraphDir)
	feedbackH := planFeedbackHandler(database)
	finalizeH := planFinalizeHandler(database, htmlgraphDir)
	deleteH := planDeleteHandler(database, htmlgraphDir)
	chatH := planChatHandler(database, htmlgraphDir)
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/chat"):
			chatH(w, r)
		case strings.HasSuffix(path, "/status"):
			statusH(w, r)
		case strings.HasSuffix(path, "/feedback"):
			feedbackH(w, r)
		case strings.HasSuffix(path, "/finalize"):
			finalizeH(w, r)
		case strings.HasSuffix(path, "/delete"):
			deleteH(w, r)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}
}

// ---- helpers ----------------------------------------------------------------

// extractPlanID parses a plan ID from URL paths of the form
// /api/plans/{id}/{suffix}. Returns an error if the ID is missing.
func extractPlanID(urlPath, suffix string) (string, error) {
	const prefix = "/api/plans/"
	path := strings.TrimSuffix(urlPath, "/")
	if !strings.HasPrefix(path, prefix) {
		return "", fmt.Errorf("unexpected path: %s", urlPath)
	}
	mid := path[len(prefix):]
	mid = strings.TrimSuffix(mid, suffix)
	if mid == "" || strings.Contains(mid, "/") {
		return "", fmt.Errorf("missing or invalid plan ID in path: %s", urlPath)
	}
	return mid, nil
}

// resolvePlanPath returns the absolute path to a plan's HTML file, or an
// error if the file does not exist.
func resolvePlanPath(htmlgraphDir, planID string) (string, error) {
	p := filepath.Join(htmlgraphDir, "plans", planID+".html")
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("plan %s not found", planID)
	}
	return p, nil
}

// parsePlanHTMLStatus reads the plan HTML file and returns the value of
// data-status on the top-level <article> element.
func parsePlanHTMLStatus(planPath string) (string, error) {
	f, err := os.Open(planPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return "", err
	}
	status, _ := doc.Find("article[id]").First().Attr("data-status")
	if status == "" {
		status = "draft"
	}
	return status, nil
}

// updatePlanHTMLStatus rewrites the plan HTML file with data-status set to
// the new value on the top-level <article> element.
func updatePlanHTMLStatus(planPath, newStatus string) error {
	data, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	doc.Find("article[id]").First().SetAttr("data-status", newStatus)
	html, err := doc.Html()
	if err != nil {
		return err
	}
	return os.WriteFile(planPath, []byte(html), 0o644)
}

// finalizePlanHTML writes a snapshot of the finalized plan with all feedback
// baked into the HTML: checkboxes checked, radio buttons selected, comments
// filled, and data-status set to "finalized". The HTML file becomes a
// self-contained record of the finalized plan.
func finalizePlanHTML(planPath string, database *sql.DB, planID string) error {
	data, err := os.ReadFile(planPath)
	if err != nil {
		return err
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	// Set article status to finalized
	doc.Find("article").First().SetAttr("data-status", "finalized")

	// Read all feedback from SQLite
	feedback, err := dbpkg.GetPlanFeedback(database, planID)
	if err != nil {
		return err
	}

	for _, fb := range feedback {
		switch fb.Action {
		case "approve":
			// Check the approval checkbox for this section
			if fb.Value == "true" {
				doc.Find(fmt.Sprintf("input[data-section='%s'][data-action='approve']", fb.Section)).
					SetAttr("checked", "checked")
			}
		case "comment":
			// Set textarea content for this section
			doc.Find(fmt.Sprintf("textarea[data-section='%s']", fb.Section)).
				SetText(fb.Value)
		case "answer":
			// Select the radio button matching this answer
			doc.Find(fmt.Sprintf("input[type='radio'][data-question='%s']", fb.QuestionID)).
				Each(func(_ int, s *goquery.Selection) {
					val, _ := s.Attr("value")
					if val == fb.Value {
						s.SetAttr("checked", "checked")
					} else {
						s.RemoveAttr("checked")
					}
				})
		}
	}

	html, err := doc.Html()
	if err != nil {
		return err
	}
	return os.WriteFile(planPath, []byte(html), 0o644)
}

// countPlanSections returns the count of approved sections and the total
// distinct sections with any feedback for the given plan.
func countPlanSections(database *sql.DB, planID string) (approved, total int, err error) {
	err = database.QueryRow(`
		SELECT
			COUNT(DISTINCT CASE WHEN action = 'approve' AND value = 'true' THEN section END),
			COUNT(DISTINCT section)
		FROM plan_feedback
		WHERE plan_id = ?`, planID,
	).Scan(&approved, &total)
	return
}

// buildFeedbackResponse groups raw feedback entries into the structured
// response consumed by the CLI and other API callers.
func buildFeedbackResponse(planID string, entries []dbpkg.PlanFeedback) planFeedbackResponse {
	sections := make(map[string]sectionFeedback)
	questions := make(map[string]string)
	approvedSections := make(map[string]bool)
	var chatMessages []chatMessageEntry

	for _, e := range entries {
		switch e.Action {
		case "approve":
			sf := sections[e.Section]
			sf.Approved = e.Value == "true"
			sections[e.Section] = sf
			if sf.Approved {
				approvedSections[e.Section] = true
			} else {
				delete(approvedSections, e.Section)
			}
		case "comment":
			sf := sections[e.Section]
			sf.Comment = e.Value
			sections[e.Section] = sf
		case "answer":
			if e.QuestionID != "" {
				questions[e.QuestionID] = e.Value
			}
		case "messages":
			// Chat messages stored as a JSON array under section='chat'.
			if e.Section == "chat" && e.Value != "" {
				var msgs []chatMessageEntry
				if json.Unmarshal([]byte(e.Value), &msgs) == nil {
					chatMessages = msgs
				}
			}
		}
	}

	// Exclude chat section from approval status calculation.
	delete(sections, "chat")
	delete(approvedSections, "chat")

	status := "draft"
	if len(sections) > 0 && len(approvedSections) == len(sections) {
		status = "approved"
	}

	return planFeedbackResponse{
		PlanID:       planID,
		Status:       status,
		Sections:     sections,
		Questions:    questions,
		ChatMessages: chatMessages,
	}
}
