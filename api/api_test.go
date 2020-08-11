package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/smartatransit/feedback/api"
	dbp "github.com/smartatransit/feedback/db"
	"github.com/smartatransit/feedback/db/dbfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("API", func() {
	var (
		log *logrus.Logger
		db  *dbfakes.FakeDB

		client api.Client

		body      interface{}
		bodyBytes []byte

		req   *http.Request
		respW *httptest.ResponseRecorder

		resp *http.Response
	)

	BeforeEach(func() {
		log = logrus.New()
		log.SetOutput(ioutil.Discard)
		db = &dbfakes.FakeDB{}

		body = nil
		bodyBytes = nil

		req, _ = http.NewRequest("", "", nil)
	})

	JustBeforeEach(func() {
		client = api.New(log, db)

		if body != nil {
			var err error
			bodyBytes, err = json.Marshal(body)
			Expect(err).To(BeNil())
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		respW = httptest.NewRecorder()
	})

	Describe("SaveFeedback", func() {
		BeforeEach(func() {
			req.Method = "POST"
			req.Header.Set("X-Smarta-Auth-Session", "r39iefjd0q39f")
			req.Header.Set("X-Smarta-Auth-Role", "anonymous")

			body = &api.SaveFeedbackRequest{
				Kind:    "outAGE",
				Value:   "POSitive",
				Message: "my message",
				Email:   "user@notsmarta.net",
			}
		})

		JustBeforeEach(func() {
			client.SaveFeedback(respW, req)
			resp = respW.Result()
		})

		When("it's not a POST request", func() {
			BeforeEach(func() {
				req.Method = "GET"
			})
			It("fails", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(405))
			})
		})
		When("an auth header is missing", func() {
			BeforeEach(func() {
				req.Header.Del("X-Smarta-Auth-Session")
			})
			It("fails", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(401))
			})
		})
		When("the JSON body is malformed", func() {
			BeforeEach(func() {
				body = nil
				bodyBytes = []byte(`{`)
			})
			It("fails", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(400))
			})
		})
		When("the kind is invalid", func() {
			BeforeEach(func() {
				body.(*api.SaveFeedbackRequest).Kind = "sdf"
			})
			It("fails", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(400))
			})
		})
		When("the value is provided and invalid", func() {
			BeforeEach(func() {
				body.(*api.SaveFeedbackRequest).Value = "sdf"
			})
			It("fails", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(400))
			})
		})
		When("the email is provided and invalid", func() {
			BeforeEach(func() {
				body.(*api.SaveFeedbackRequest).Email = "sdf"
			})
			It("fails", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(400))
			})
		})
		When("the database update fails", func() {
			BeforeEach(func() {
				db.SaveFeedbackReturns(errors.New("insert failed"))
			})
			It("fails", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(500))
			})
		})
		When("all goes well", func() {
			It("succeeds", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(200))

				_, fb := db.SaveFeedbackArgsForCall(0)
				Expect(fb).To(MatchAllFields(Fields{
					"ID":         Ignore(),
					"Silenced":   Ignore(),
					"ReceivedAt": Ignore(),

					"SessionID": Equal("r39iefjd0q39f"),
					"Role":      Equal("anonymous"),
					"Kind":      Equal("outage"),
					"Message":   PointTo(Equal("my message")),
					"Value":     PointTo(Equal("positive")),
					"Email":     PointTo(Equal("user@notsmarta.net")),
				}))
			})
		})
	})

	Describe("Health", func() {
		BeforeEach(func() {
			req.Method = "GET"
		})

		JustBeforeEach(func() {
			client.Health(respW, req)
			resp = respW.Result()
		})

		When("recent outages can't be obtained", func() {
			BeforeEach(func() {
				db.GetRecentOutagesReturns(nil, errors.New("select failed"))
			})
			It("succeeds", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(200))
				var respObj api.HealthResponse
				err := json.NewDecoder(resp.Body).Decode(&respObj)
				Expect(err).To(BeNil())
				Expect(respObj).To(MatchAllFields(Fields{
					"Statuses": ConsistOf(
						MatchAllFields(Fields{
							"Name":        Equal("database"),
							"Description": Equal("postgres backend"),
							"Healthy":     BeFalse(),
							"Metadata":    BeNil(),
						}),
					),
				}))
			})
		})
		When("there are recent outage reports", func() {
			var t time.Time
			BeforeEach(func() {
				t = time.Now()
				db.GetRecentOutagesReturns([]dbp.Feedback{
					{
						ID:         "fweawf",
						Message:    ptrToString("aasdfasdf"),
						ReceivedAt: t,
					},
					{
						ID:         "fweawf-2",
						Message:    ptrToString("aasdfasdf-2"),
						ReceivedAt: t.Add(time.Hour),
					},
				}, nil)
			})
			It("succeeds", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(200))
				var respObj api.HealthResponse
				err := json.NewDecoder(resp.Body).Decode(&respObj)
				Expect(err).To(BeNil())
				Expect(respObj).To(MatchAllFields(Fields{
					"Statuses": ConsistOf(
						MatchAllFields(Fields{
							"Name":        Equal("database"),
							"Description": Equal("postgres backend"),
							"Healthy":     BeTrue(),
							"Metadata":    BeNil(),
						}),
						MatchAllFields(Fields{
							"Name":        Equal("user_outage_reports"),
							"Description": Equal("outage reports directly from users"),
							"Healthy":     BeFalse(),
							"Metadata": MatchAllKeys(Keys{
								"outages": ConsistOf(
									MatchAllKeys(Keys{
										"id":          Equal("fweawf"),
										"message":     Equal("aasdfasdf"),
										"received_at": Ignore(),
									}),
									MatchAllKeys(Keys{
										"id":          Equal("fweawf-2"),
										"message":     Equal("aasdfasdf-2"),
										"received_at": Ignore(),
									}),
								),
							}),
						}),
					),
				}))
			})
		})
		When("there are no recent outage reports", func() {
			It("succeeds", func() {
				Expect(resp.StatusCode).To(BeEquivalentTo(200))
				var respObj api.HealthResponse
				err := json.NewDecoder(resp.Body).Decode(&respObj)
				Expect(err).To(BeNil())
				Expect(respObj).To(MatchAllFields(Fields{
					"Statuses": ConsistOf(
						MatchAllFields(Fields{
							"Name":        Equal("database"),
							"Description": Equal("postgres backend"),
							"Healthy":     BeTrue(),
							"Metadata":    BeNil(),
						}),
						MatchAllFields(Fields{
							"Name":        Equal("user_outage_reports"),
							"Description": Equal("outage reports directly from users"),
							"Healthy":     BeTrue(),
							"Metadata":    BeNil(),
						}),
					),
				}))
			})
		})
	})
})

func ptrToString(s string) *string {
	return &s
}
