CREATE SCHEMA IF NOT EXISTS s;
CREATE TABLE s.t1 (
    id SERIAL PRIMARY KEY,
    data INT NOT NULL
);
CREATE TABLE s.t2 (
    id SERIAL PRIMARY KEY,
    id_1 INT NOT NULL REFERENCES s.t1(id)
);