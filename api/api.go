package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/smartatransit/feedback/db"
)

//Kind is a kind of user beedback
type Kind string

//ValidKinds enumerates valid kinds
var ValidKinds = map[Kind]struct{}{
	OutageKind:           {},
	CommentKind:          {},
	ServiceConditionKind: {},
}

const (
	//OutageKind represents a user-submitted outage
	OutageKind Kind = "OUTAGE"

	//CommentKind represents a user-submitted comment
	CommentKind Kind = "COMMENT"

	//ServiceConditionKind represents a user-submitted service condition
	ServiceConditionKind Kind = "SERVICE_CONDITION"
)

//SaveFeedbackRequest represents a user feedback record
type SaveFeedbackRequest struct {
	Kind    Kind   `json:"kind"`
	Message string `json:"message"`
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
type API interface {
	SaveFeedback(w http.ResponseWriter, r *http.Request)
	Health(w http.ResponseWriter, r *http.Request)
}

//Client implements API
type Client struct {
	db db.DB
}

//New returns a new Client
func New(
	db db.DB,
) Client {
	return Client{db: db}
}

//SaveFeedback saves a feedback using information from the request body as well
//as from headers forwarded by the API gateway.
func (c Client) SaveFeedback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "use POST instead")
		return
	}

	// TODO change the API gateway!
	session := r.Header.Get("X-Smarta-Auth-Session")
	role := r.Header.Get("X-Smarta-Auth-Role")
	if len(session) == 0 || len(role) == 0 {
		writeErrorResponse(w, http.StatusUnauthorized, "expected X-Smarta-* headers not present")
		return
	}

	var reqObj SaveFeedbackRequest
	err := json.NewDecoder(r.Body).Decode(&reqObj)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("malformed JSON request body: %s", err.Error()))
		return
	}

	if len(reqObj.Message) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "`message` is a required field")
		return
	}

	if _, ok := ValidKinds[reqObj.Kind]; !ok {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid value `%s` for `kind`", reqObj.Kind))
		return
	}

	err = c.db.SaveFeedback(r.Context(), db.Feedback{
		SessionID: session,
		Role:      role,
		Kind:      string(reqObj.Kind),
		Message:   reqObj.Message,
	})
	if err != nil {
		// TODO log the error
		writeErrorResponse(w, http.StatusInternalServerError, "failed to save feedback")
		return
	}

	return
}

type outageReportMetadata struct {
	Outages []outageReport `json:"outages"`
}

type outageReport struct {
	Message    string    `json:"message"`
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

		writeJSONResponse(w, http.StatusOK, HealthResponse{Statuses: statuses})
		// TODO push out a response
	}()

	err := c.db.TestConnection(r.Context())
	if err != nil {
		return
	}

	outageReports, err := c.db.GetRecentOutages(r.Context(), time.Now().Add(-48*time.Hour))
	if err != nil {
		return
	}

	statuses = append(statuses, Status{
		Name:        "database",
		Description: "postgres backend",
		Healthy:     true,
	})

	statuses = append(statuses, reportStatusFromFeedbackList(outageReports))
	return
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

func writeErrorResponse(w http.ResponseWriter, status int, errMsg string) {
	writeJSONResponse(w, status, errResp{
		Status:  status,
		Message: errMsg,
	})
}

func writeJSONResponse(w http.ResponseWriter, status int, body interface{}) {
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		// TODO log
		return
	}
}
