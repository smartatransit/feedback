CREATE TYPE kind AS ENUM ('outage', 'comment', 'service_condition');
CREATE TYPE value AS ENUM ('positive', 'negative', 'neutral');

CREATE TABLE IF NOT EXISTS feedbacks
(	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	session_id varchar NOT NULL,
	role varchar NOT NULL,
	kind kind NOT NULL,
	value value,
	message varchar,
	email varchar,

	received_moment timestamp DEFAULT NOW() NOT NULL,
	silenced boolean DEFAULT FALSE NOT NULL
);

CREATE INDEX feedbacks_kind_received_idx ON feedbacks (kind, received_moment);
