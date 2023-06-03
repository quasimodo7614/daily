-- psql -h localhost -U postgres -d postgres
-- \c zze

CREATE TABLE completed_items (
                                 id SERIAL PRIMARY KEY,
                                 item TEXT NOT NULL,
                                 description TEXT NOT NULL,
                                 completed_time TIMESTAMP WITH TIME ZONE NOT NULL
);