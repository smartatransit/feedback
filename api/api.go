package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/smartatransit/feedback/db"
)

//ValidKinds enumerates valid kinds
var ValidKinds = map[string]struct{}{
	"outage":            {},
	"comment":           {},
	"service_condition": {},
}

//ValidValues enumerates valid kinds
var ValidValues = map[string]struct{}{
	"positive": {},
	"negative": {},
	"neutral":  {},
}

//SaveFeedbackRequest represents a user feedback record
type SaveFeedbackRequest struct {
	Kind    string `json:"kind"`
	Value   string `json:"value"`
	Message string `json:"message"`
	Email   string `json:"email"`
}

//HealthResponse represents a response to the health-check endpoint
type HealthResponse struct {
	Statuses []Status `json:"statuses"`
}

//Status represents a single system status
type Status struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Healthy     bool        `json:"healthy"`
	Metadata    interface{} `json:"metadata,omitempty"`
}

//API exposes the API endpoints
//go:generate counterfeiter . API
type API interface {
	SaveFeedback(w http.ResponseWriter, r *http.Request)
	Health(w http.ResponseWriter, r *http.Request)
}

//Client implements API
type Client struct {
	log *logrus.Logger
	db  db.DB
}

//New returns a new Client
func New(
	log *logrus.Logger,
	db db.DB,
) Client {
	return Client{
		log: log,
		db:  db,
	}
}

var emailRegexp = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)

//SaveFeedback saves a feedback using information from the request body as well
//as from headers forwarded by the API gateway.
func (c Client) SaveFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		c.writeErrorResponse(w, http.StatusMethodNotAllowed, "use POST instead")
		return
	}

	session := r.Header.Get("X-Smarta-Auth-Session")
	role := r.Header.Get("X-Smarta-Auth-Role")
	if len(session) == 0 || len(role) == 0 {
		c.writeErrorResponse(w, http.StatusUnauthorized, "expected X-Smarta-Auth-* headers not present")
		return
	}

	var req SaveFeedbackRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		c.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("malformed JSON request body: %s", err.Error()))
		return
	}

	feedback := db.Feedback{
		SessionID: session,
		Role:      role,
	}

	err = mapSaveFeedbackRequestFieldsOntoFeedback(&feedback, req)
	if err != nil {
		c.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err = c.db.SaveFeedback(r.Context(), feedback)
	if err != nil {
		c.log.Error(err.Error())
		c.writeErrorResponse(w, http.StatusInternalServerError, "failed to save feedback")
		return
	}
}

func mapSaveFeedbackRequestFieldsOntoFeedback(feedback *db.Feedback, req SaveFeedbackRequest) (err error) {
	req.Kind = strings.ToLower(req.Kind)
	req.Value = strings.ToLower(req.Value)

	if _, ok := ValidKinds[req.Kind]; !ok {
		err = fmt.Errorf("invalid value `%s` for `kind`", req.Kind)
		return
	}
	feedback.Kind = req.Kind

	if req.Value != "" {
		if _, ok := ValidValues[req.Value]; !ok {
			err = fmt.Errorf("invalid value `%s` for `value`", req.Value)
			return
		}
		feedback.Value = &req.Value
	}

	if req.Email != "" {
		if !emailRegexp.Match([]byte(req.Email)) {
			err = fmt.Errorf("invalid value `%s` for `email`", req.Email)
			return
		}
		feedback.Email = &req.Email
	}

	if req.Message != "" {
		feedback.Message = &req.Message
	}

	return nil
}

type outageReportMetadata struct {
	Outages []outageReport `json:"outages"`
}

type outageReport struct {
	ID         string    `json:"id"`
	Message    *string   `json:"message,omitempty"`
	ReceivedAt time.Time `json:"received_at"`
}

//Health responds with a variety of internal statuses
func (c Client) Health(w http.ResponseWriter, r *http.Request) {
	var statuses []Status
	defer func() {
		if len(statuses) == 0 {
			statuses = append(statuses, Status{
				Name:        "database",
				Description: "postgres backend",
				Healthy:     false,
			})
		}

		c.writeJSONResponse(w, http.StatusOK, HealthResponse{Statuses: statuses})
	}()

	outageReports, err := c.db.GetRecentOutages(r.Context(), time.Now().Add(-48*time.Hour))
	if err != nil {
		c.log.Error(err.Error())
		return
	}

	statuses = append(statuses, Status{
		Name:        "database",
		Description: "postgres backend",
		Healthy:     true,
	})

	statuses = append(statuses, reportStatusFromFeedbackList(outageReports))
}

func reportStatusFromFeedbackList(outageReports []db.Feedback) (st Status) {
	st.Name = "user_outage_reports"
	st.Description = "outage reports directly from users"
	if len(outageReports) == 0 {
		st.Healthy = true
		return
	}

	repList := []outageReport{}
	for _, rep := range outageReports {
		repList = append(repList, outageReport{
			ID:         rep.ID,
			Message:    rep.Message,
			ReceivedAt: rep.ReceivedAt,
		})
	}

	st.Metadata = outageReportMetadata{
		Outages: repList,
	}
	return
}

type errResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (c Client) writeErrorResponse(w http.ResponseWriter, status int, errMsg string) {
	c.writeJSONResponse(w, status, errResp{
		Status:  status,
		Message: errMsg,
	})
}

func (c Client) writeJSONResponse(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		c.log.Errorf("failed writing response: %s", err.Error())
		return
	}
}
