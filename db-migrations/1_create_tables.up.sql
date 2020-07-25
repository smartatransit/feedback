CREATE TYPE kind AS ENUM ('outage', 'comment', 'service_condition');

CREATE TABLE IF NOT EXISTS feedbacks (
  session_id varchar NOT NULL,
  role varchar NOT NULL,
  kind kind NOT NULL,
  message varchar NOT NULL,

  received_moment timestamp DEFAULT NOW() NOT NULL,
  silenced boolean DEFAULT FALSE NOT NULL
);

CREATE INDEX feedbacks_kind_received_idx ON feedbacks (kind, received_moment);
